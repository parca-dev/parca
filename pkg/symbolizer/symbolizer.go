// Copyright 2022-2026 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package symbolizer

import (
	"bufio"
	"context"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	profilepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/addr2line"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var (
	ErrNotValidElf = errors.New("not a valid ELF file")
	ErrNoDebuginfo = errors.New("no debug info found")
	ErrLinerFailed = errors.New("liner creation failed")
)

type DebuginfoMetadata interface {
	SetQuality(ctx context.Context, buildID string, typ debuginfopb.DebuginfoType, quality *debuginfopb.DebuginfoQuality) error
	Fetch(ctx context.Context, buildID string, typ debuginfopb.DebuginfoType) (*debuginfopb.Debuginfo, error)
}

// liner is the interface implemented by symbolizers
// which read an object file (symbol table or debug information) and return
// source code lines by a given memory address.
type liner interface {
	PCToLines(ctx context.Context, pc uint64) ([]profile.LocationLine, error)
	Close() error
}

type Option func(*Symbolizer)

func WithDemangleMode(mode string) Option {
	return func(s *Symbolizer) {
		s.demangler = demangle.NewDemangler(mode, false)
	}
}

type Symbolizer struct {
	logger log.Logger

	debuginfo             DebuginfoFetcher
	cache                 SymbolizerCache
	metadata              DebuginfoMetadata
	externalAddr2linePath string

	demangler *demangle.Demangler

	tmpDir string
}

type DebuginfoFetcher interface {
	// Fetch ensures that the debug info for the given build ID is available on
	// a local filesystem and returns a path to it.
	FetchDebuginfo(ctx context.Context, dbginfo *debuginfopb.Debuginfo) (io.ReadCloser, error)
}

type SymbolizerCache interface {
	Get(ctx context.Context, buildID string, addr uint64) ([]profile.LocationLine, bool, error)
	Set(ctx context.Context, buildID string, addr uint64, lines []profile.LocationLine) error
}

func New(
	logger log.Logger,
	metadata DebuginfoMetadata,
	cache SymbolizerCache,
	debuginfo DebuginfoFetcher,
	tmpDir string,
	externalAddr2linePath string,
	opts ...Option,
) *Symbolizer {
	const (
		defaultDemangleMode = "simple"
	)

	s := &Symbolizer{
		logger:                log.With(logger, "component", "symbolizer"),
		cache:                 cache,
		debuginfo:             debuginfo,
		externalAddr2linePath: externalAddr2linePath,
		tmpDir:                tmpDir,
		metadata:              metadata,
		demangler:             demangle.NewDemangler(defaultDemangleMode, false),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

type SymbolizationRequestMappingAddrs struct {
	// This slice is used to store the symbolization result directly.
	Locations []*profile.Location
}

type SymbolizationRequest struct {
	BuildID  string
	Mappings []SymbolizationRequestMappingAddrs
}

func (s *Symbolizer) Symbolize(
	ctx context.Context,
	req SymbolizationRequest,
) error {
	if err := s.symbolize(ctx, req); err != nil {
		level.Debug(s.logger).Log("msg", fmt.Sprintf("failed to symbolize: BuildID %s", req.BuildID), "err", err)
	}

	return nil
}

func (s *Symbolizer) symbolize(
	ctx context.Context,
	req SymbolizationRequest,
) error {
	level.Debug(s.logger).Log("msg", "symbolizing", "build_id", req.BuildID)

	path, f, quality, err := s.getDebuginfo(ctx, req.BuildID)
	if err != nil {
		return fmt.Errorf("get debuginfo: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			level.Debug(s.logger).Log("msg", "failed to close debuginfo file", "err", err)
		}
	}()

	l := s.newLiner(path, f, quality, req.BuildID)
	defer l.Close()

	ei, err := profile.ExecutableInfoFromELF(f)
	if err != nil {
		return fmt.Errorf("executable info from ELF: %w", err)
	}

	for _, mapping := range req.Mappings {
		for _, loc := range mapping.Locations {
			addr, err := NormalizeAddress(loc.Address, ei, profile.Mapping{
				StartAddr: loc.Mapping.Start,
				EndAddr:   loc.Mapping.Limit,
				Offset:    loc.Mapping.Offset,
			})
			if err != nil {
				return fmt.Errorf("normalize address: %w", err)
			}

			loc.Lines, err = l.PCToLines(ctx, addr)
			if err != nil {
				level.Debug(s.logger).Log("msg", "failed to get lines", "err", err)
			}
		}
	}

	return nil
}

func (s *Symbolizer) getDebuginfo(ctx context.Context, buildID string) (string, *elf.File, *debuginfopb.DebuginfoQuality, error) {
	dbginfo, err := s.metadata.Fetch(ctx, buildID, debuginfopb.DebuginfoType_DEBUGINFO_TYPE_DEBUGINFO_UNSPECIFIED)
	if err != nil {
		return "", nil, nil, fmt.Errorf("fetching metadata: %w", err)
	}

	if dbginfo.Quality != nil {
		if dbginfo.Quality.NotValidElf {
			return "", nil, nil, ErrNotValidElf
		}
		if !dbginfo.Quality.HasDwarf && !dbginfo.Quality.HasGoPclntab && !dbginfo.Quality.HasSymtab && !dbginfo.Quality.HasDynsym {
			return "", nil, nil, fmt.Errorf("check previously reported debuginfo quality: %w", ErrNoDebuginfo)
		}
	}

	switch dbginfo.Source {
	case debuginfopb.Debuginfo_SOURCE_UPLOAD:
		switch dbginfo.Upload.State {
		case debuginfopb.DebuginfoUpload_STATE_UPLOADED:
			// Good to proceed
		case debuginfopb.DebuginfoUpload_STATE_PURGED:
			return "", nil, nil, debuginfo.ErrDebuginfoPurged
		default:
			return "", nil, nil, debuginfo.ErrNotUploadedYet
		}
	case debuginfopb.Debuginfo_SOURCE_DEBUGINFOD:
		// Nothing to do here, just covering all cases.
	default:
		return "", nil, nil, debuginfo.ErrUnknownDebuginfoSource
	}

	targetPath := filepath.Join(s.tmpDir, buildID)
	if _, err := os.Stat(targetPath); errors.Is(err, os.ErrNotExist) {
		// Fetch the debug info for the build ID.
		rc, err := s.debuginfo.FetchDebuginfo(ctx, dbginfo)
		if err != nil {
			return "", nil, nil, fmt.Errorf("fetch debuginfo (BuildID: %q): %w", buildID, err)
		}
		defer rc.Close()

		f, err := os.CreateTemp(s.tmpDir, "parca-symbolizer-*")
		if err != nil {
			return "", nil, nil, fmt.Errorf("create temp file: %w", err)
		}

		level.Debug(s.logger).Log("msg", fmt.Sprintf("Copying %s to tmp file %s\n", buildID, f.Name()))
		defer func() {
			f.Close()
			os.Remove(f.Name())
		}()

		if _, err := io.Copy(f, rc); err != nil {
			return "", nil, nil, fmt.Errorf("copy debuginfo to temp file: %w", err)
		}

		if err := f.Close(); err != nil {
			return "", nil, nil, fmt.Errorf("close temp file: %w", err)
		}

		if err := os.Rename(f.Name(), targetPath); err != nil {
			return "", nil, nil, fmt.Errorf("rename temp file: %w", err)
		}
	}

	e, err := elf.Open(targetPath)
	if err != nil {
		if merr := s.metadata.SetQuality(ctx, buildID, debuginfopb.DebuginfoType_DEBUGINFO_TYPE_DEBUGINFO_UNSPECIFIED, &debuginfopb.DebuginfoQuality{
			NotValidElf: true,
		}); merr != nil {
			level.Debug(s.logger).Log("msg", "failed to set metadata quality", "err", merr)
		}

		return "", nil, nil, fmt.Errorf("open temp file as ELF: %w", err)
	}

	if dbginfo.Quality == nil {
		dbginfo.Quality = &debuginfopb.DebuginfoQuality{
			HasDwarf:     elfutils.HasDWARF(e),
			HasGoPclntab: elfutils.HasGoPclntab(e),
			HasSymtab:    elfutils.HasSymtab(e),
			HasDynsym:    elfutils.HasDynsym(e),
		}
		if err := s.metadata.SetQuality(ctx, buildID, debuginfopb.DebuginfoType_DEBUGINFO_TYPE_DEBUGINFO_UNSPECIFIED, dbginfo.Quality); err != nil {
			if err := e.Close(); err != nil {
				level.Debug(s.logger).Log("msg", "failed to close debuginfo file", "err", err)
			}
			return "", nil, nil, fmt.Errorf("set quality: %w", err)
		}
		if !dbginfo.Quality.HasDwarf && !dbginfo.Quality.HasGoPclntab && !dbginfo.Quality.HasSymtab && !dbginfo.Quality.HasDynsym {
			if err := e.Close(); err != nil {
				level.Debug(s.logger).Log("msg", "failed to close debuginfo file", "err", err)
			}
			return "", nil, nil, fmt.Errorf("check debuginfo quality: %w", ErrNoDebuginfo)
		}
	}

	return targetPath, e, dbginfo.Quality, nil
}

type cachedLiner struct {
	logger                log.Logger
	demangler             *demangle.Demangler
	filepath              string
	f                     *elf.File
	quality               *debuginfopb.DebuginfoQuality
	buildID               string
	externalAddr2linePath string

	// this is the concrete liner
	liner liner

	cache SymbolizerCache
}

type externalExecutableLiner struct {
	cmd       exec.Cmd
	inputPipe io.WriteCloser
	scanner   *bufio.Scanner

	logger    log.Logger
	demangler *demangle.Demangler
}

func (c *cachedLiner) newExternalExecutableLiner(
	filepath string,
) (liner, error) {
	// Create cmd with pipes and scanner for reading output line by line
	addr2lineCmd := exec.Command(c.externalAddr2linePath, "--exe", filepath, "-afiC")
	addr2lineInput, err := addr2lineCmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating input pipe for addr2line: %w", err)
	}
	addr2lineOutput, err := addr2lineCmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("error creating output pipe for addr2line: %w", err)
	}

	// 100 KiB buffer, addr2line output can be very large sometimes
	buf := make([]byte, 100*1024)
	scanner := bufio.NewScanner(addr2lineOutput)
	scanner.Buffer(buf, 100*1024)

	// Start command to load executable info and wait for input
	err = addr2lineCmd.Start()
	if err != nil {
		return nil, fmt.Errorf("error starting addr2line: %w", err)
	}

	return &externalExecutableLiner{
		cmd:       *addr2lineCmd,
		inputPipe: addr2lineInput,
		scanner:   scanner,
		logger:    c.logger,
		demangler: c.demangler,
	}, nil
}

func (l *externalExecutableLiner) Close() error {
	err := l.inputPipe.Close()
	if err != nil {
		level.Warn(l.logger).Log("msg", "error closing input pipe of addr2line process", "err", err)
		return err
	}

	err = l.cmd.Wait()
	if err != nil {
		level.Warn(l.logger).Log("msg", "error waiting for addr2line process to exit", "err", err)
		return err
	}
	level.Debug(l.logger).Log("msg", "addr2line process exited successfully, closed external executable liner")

	return nil
}

func (l *externalExecutableLiner) PCToLines(ctx context.Context, pc uint64) ([]profile.LocationLine, error) {
	// Write the address to addr2line stdin twice in order to get addresses as a proper delimiter
	for i := 0; i < 2; i++ {
		_, err := io.WriteString(l.inputPipe, fmt.Sprintf("0x%x\n", pc))
		if err != nil {
			return nil, fmt.Errorf("error writing to addr2line stdin: %w", err)
		}
	}

	// Auxiliary function to call Scan() and handle errors
	scan := func() error {
		if !l.scanner.Scan() {
			if err := l.scanner.Err(); err != nil {
				return fmt.Errorf("error scanning addr2line output: %w", err)
			}
			return fmt.Errorf("unexpected EOF while scanning addr2line output")
		}
		return nil
	}

	lines := []profile.LocationLine{}
	// addr2line output consists of a variable number of pairs of lines per address
	// The first line of each block of output is the address, if we have to skip additional lines
	// it means the previous read loop left some output unread
	err := scan()
	if err != nil {
		return nil, err
	}
	addressLine := l.scanner.Text()
	for !strings.HasPrefix(addressLine, "0x") {
		err := scan()
		if err != nil {
			return nil, err
		}
		addressLine = l.scanner.Text()
	}

	keepReading := true
	err = scan()
	if err != nil {
		return nil, err
	}
	for keepReading {
		line1 := l.scanner.Text()

		err := scan()
		if err != nil {
			return nil, err
		}
		line2 := l.scanner.Text()

		line1 = strings.TrimSuffix(line1, "\n")
		line2 = strings.TrimSuffix(line2, "\n")

		fileLocation := strings.Split(line2, ":")
		lineNum, err := strconv.Atoi(fileLocation[1])
		if err != nil {
			lineNum = 0
		}

		lines = append(lines, profile.LocationLine{
			Line: int64(lineNum),
			Function: &profilepb.Function{
				StartLine: int64(lineNum),
				Filename:  fileLocation[0],
				Name:      line1,
			},
		})

		err = scan()
		if err != nil {
			return nil, err
		}
		keepReading = !strings.HasPrefix(l.scanner.Text(), "0x")
	}

	return lines, nil
}

// newConcreteLiner creates a new liner for the given mapping and object file path.
func (s *Symbolizer) newLiner(
	filepath string,
	f *elf.File,
	quality *debuginfopb.DebuginfoQuality,
	buildID string,
) liner {
	return &cachedLiner{
		logger:                s.logger,
		demangler:             s.demangler,
		filepath:              filepath,
		f:                     f,
		quality:               quality,
		buildID:               buildID,
		externalAddr2linePath: s.externalAddr2linePath,

		cache: s.cache,
	}
}

func (c *cachedLiner) Close() error {
	if c.liner != nil {
		return c.liner.Close()
	}

	return nil
}

func (c *cachedLiner) PCToLines(ctx context.Context, pc uint64) ([]profile.LocationLine, error) {
	lines, ok, err := c.cache.Get(ctx, c.buildID, pc)
	if err != nil {
		return nil, fmt.Errorf("get from cache: %w", err)
	}
	if ok {
		return lines, nil
	}

	if c.liner == nil {
		// delay liner creation until first use, we may not need it if we find
		// the result in the cache
		c.liner, err = c.newConcreteLiner(c.filepath, c.f, c.quality)
		if err != nil {
			return nil, fmt.Errorf("new concrete liner: %w", err)
		}
	}

	lines, err = c.liner.PCToLines(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("liner pctolines: %w", err)
	}

	if err := c.cache.Set(ctx, c.buildID, pc, lines); err != nil {
		return nil, fmt.Errorf("set cache: %w", err)
	}

	return lines, nil
}

func (c *cachedLiner) newConcreteLiner(filepath string, _ *elf.File, quality *debuginfopb.DebuginfoQuality) (liner, error) {
	switch {
	case c.externalAddr2linePath != "":
		level.Debug(c.logger).Log("msg", fmt.Sprintf("using external addr2line liner with binary: %s", c.externalAddr2linePath))
		return c.newExternalExecutableLiner(filepath)
	case quality.HasDwarf:
		lnr, err := addr2line.DWARF(c.logger, c.filepath, c.f, c.demangler)
		if err != nil {
			return nil, fmt.Errorf("failed to create DWARF liner: %w", err)
		}

		return lnr, nil
		// TODO CHECK plt
	case quality.HasGoPclntab:
		lnr, err := addr2line.Go(c.logger, c.filepath, c.f)
		if err != nil {
			return nil, fmt.Errorf("failed to create Go liner: %w", err)
		}

		return lnr, nil
	case quality.HasSymtab || quality.HasDynsym:
		lnr, err := addr2line.Symbols(c.logger, c.filepath, c.f, c.demangler)
		if err != nil {
			return nil, fmt.Errorf("failed to create Symtab liner: %w", err)
		}

		return lnr, nil
	default:
		return nil, ErrLinerFailed
	}
}
