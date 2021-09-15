package debuginfo

import (
	"debug/elf"
	"debug/gosym"
	"errors"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"

	"github.com/parca-dev/parca/internal/pprof/binutils"
)

type symbolizer struct {
	logger log.Logger
	bu     *binutils.Binutils
}

func (s *symbolizer) createAdd2Line(m *profile.Mapping, binPath string) (add2Line, error) {
	// Go binaries has a special case. They use ".gopclntab" section to symbolize addresses.
	// Keep that section and other identifying sections in the debug information file.
	isGo, err := isGoBinary(binPath)
	if err != nil {
		level.Debug(s.logger).Log(
			"msg", "failed to determine if binary is a Go binary",
			"path", binPath,
			"err", err,
		)
	}

	var sourceLine add2Line
	switch {
	case isGo:
		sourceLine, err = s.goBinary(binPath)
		if err != nil {
			level.Error(s.logger).Log(
				"msg", "failed to create go binary add2Line, falling back to binary add2Line",
				"path", binPath,
				"err", err,
			)
			sourceLine, err = s.compiledBinary(m, binPath)
		}
	default:
		sourceLine, err = s.compiledBinary(m, binPath)
	}

	return sourceLine, err
}

func (s *symbolizer) compiledBinary(m *profile.Mapping, binPath string) (add2Line, error) {
	objFile, err := s.bu.Open(binPath, m.Start, m.Limit, m.Offset)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to open object file",
			"path", binPath,
			"start", m.Start,
			"limit", m.Limit,
			"offset", m.Offset,
			"err", err,
		)
		return nil, fmt.Errorf("open object file: %w", err)
	}

	return func(addr uint64) ([]profile.Line, error) {
		frames, err := objFile.SourceLine(addr)
		if err != nil {
			level.Debug(s.logger).Log("msg", "failed to open object file",
				"path", binPath,
				"start", m.Start,
				"limit", m.Limit,
				"offset", m.Offset,
				"address", addr,
				"err", err,
			)
			return nil, err
		}

		if len(frames) == 0 {
			return nil, errors.New("could not find any frames for given address")
		}

		lines := []profile.Line{}
		for _, frame := range frames {
			lines = append(lines, profile.Line{
				Line: int64(frame.Line),
				Function: &profile.Function{
					Name:     frame.Func,
					Filename: frame.File,
				},
			})
		}
		return lines, nil
	}, nil
}

func (s *symbolizer) goBinary(binPath string) (add2Line, error) {
	level.Debug(s.logger).Log("msg", "symbolizing a Go binary", "path", binPath)
	table, err := gosymtab(binPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create go symbtab: %w", err)
	}

	return func(addr uint64) (lines []profile.Line, err error) {
		defer func() {
			// PCToLine panics with "invalid memory address or nil pointer dereference",
			//	- when it refers to an address that doesn't actually exist.
			if r := recover(); r != nil {
				err = fmt.Errorf("recovering from panic in go binary add2line: %v", r)
			}
		}()

		file, line, fn := table.PCToLine(addr)
		lines = append(lines, profile.Line{
			Line: int64(line),
			Function: &profile.Function{
				Name:     fn.Name,
				Filename: file,
			},
		})
		return lines, nil
	}, nil
}

// Simplified version of rsc.io/goversion/version.
func isGoBinary(path string) (bool, error) {
	exe, err := elf.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	for _, s := range exe.Sections {
		if s.Name == ".note.go.buildid" {
			return true, nil
		}
	}

	syms, err := exe.Symbols()
	if err != nil {
		return false, fmt.Errorf("failed to read symbols: %w", err)
	}
	for _, sym := range syms {
		name := sym.Name
		if name == "runtime.main" || name == "main.main" {
			return true, nil
		}
		if name == "runtime.buildVersion" {
			return true, nil
		}
	}

	return false, err
}

func gosymtab(path string) (*gosym.Table, error) {
	exe, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	var pclntab []byte
	if sec := exe.Section(".gopclntab"); sec != nil {
		pclntab, err = sec.Data()
		if err != nil {
			return nil, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	var text uint64 = 0
	if sec := exe.Section(".text"); sec != nil {
		text = sec.Addr
	}

	table, err := gosym.NewTable(nil, gosym.NewLineTable(pclntab, text))
	if err != nil {
		return nil, fmt.Errorf("failed to build symtab or pclinetab: %w", err)
	}
	return table, nil
}
