// Copyright 2020 The Parca Authors
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

package addr2line

import (
	"debug/dwarf"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/go-delve/delve/pkg/dwarf/godwarf"
	"github.com/go-delve/delve/pkg/dwarf/reader"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/internal/go/debug/elf"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

var ErrLocationFailedBefore = errors.New("failed to symbolized location")

type dwarfLiner struct {
	logger    log.Logger
	demangler *demangle.Demangler

	mapping             *pb.Mapping
	data                *dwarf.Data
	lineEntries         map[dwarf.Offset][]dwarf.LineEntry
	subprograms         map[dwarf.Offset][]*godwarf.Tree
	abstractSubprograms map[dwarf.Offset]*dwarf.Entry

	attemptThreshold int
	attempts         map[uint64]int
	failed           map[uint64]struct{}
}

func DWARF(logger log.Logger, demangler *demangle.Demangler, attemptThreshold int, m *pb.Mapping, path string) (*dwarfLiner, error) {
	// TODO(kakkoyun): Handle offset, start and limit for dynamically linked libraries.
	//f, err := s.bu.Open(file, m.Start, m.Limit, m.Offset)
	//if err != nil {
	//	return nil, fmt.Errorf("open object file: %w", err)
	//}
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer f.Close()

	data, err := f.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to read DWARF data: %w", err)
	}

	return &dwarfLiner{
		logger:    logger,
		demangler: demangler,

		mapping: m,
		data:    data,

		lineEntries:         map[dwarf.Offset][]dwarf.LineEntry{},
		subprograms:         map[dwarf.Offset][]*godwarf.Tree{},
		abstractSubprograms: map[dwarf.Offset]*dwarf.Entry{},

		attemptThreshold: attemptThreshold,
		attempts:         map[uint64]int{},
		failed:           map[uint64]struct{}{},
	}, nil
}

func (dl *dwarfLiner) ensureLookUpTablesBuilt(cu *dwarf.Entry) error {
	if _, ok := dl.lineEntries[cu.Offset]; ok {
		// Already created.
		return nil
	}

	// The reader is positioned at byte offset 0 in the DWARF “line” section.
	lr, err := dl.data.LineReader(cu)
	if err != nil {
		return err
	}
	if lr == nil {
		return errors.New("failed to initialize line reader")
	}

	for {
		le := dwarf.LineEntry{}
		err := lr.Next(&le)
		if err != nil {
			break
		}
		if le.IsStmt {
			dl.lineEntries[cu.Offset] = append(dl.lineEntries[cu.Offset], le)
		}
	}

	er := dl.data.Reader()
	// The reader is positioned at byte offset of compile unit in the DWARF “info” section.
	er.Seek(cu.Offset)
	entry, err := er.Next()
	if err != nil || entry == nil {
		return errors.New("failed to read entry for compile unit")
	}

	if entry.Tag != dwarf.TagCompileUnit {
		return errors.New("failed to find entry for compile unit")
	}

outer:
	for {
		entry, err := er.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		if entry == nil {
			break
		}
		if entry.Tag == dwarf.TagCompileUnit {
			// Reached to another compile unit.
			break
		}

		if entry.Tag == dwarf.TagSubprogram {
			for _, field := range entry.Field {
				if field.Attr == dwarf.AttrInline {
					dl.abstractSubprograms[entry.Offset] = entry
					continue outer
				}
			}

			tr, err := godwarf.LoadTree(entry.Offset, dl.data, 0)
			if err != nil {
				return fmt.Errorf("failed to extract dwarf tree: %w", err)
			}

			dl.subprograms[cu.Offset] = append(dl.subprograms[cu.Offset], tr)
		}
	}

	return nil
}

func (dl *dwarfLiner) PCToLines(addr uint64) (lines []metastore.LocationLine, err error) {
	// Check if we already attempt to symbolize this location and failed.
	if _, failedBefore := dl.failed[addr]; failedBefore {
		level.Debug(dl.logger).Log("msg", "location already had been attempted to be symbolized and failed, skipping")
		return nil, ErrLocationFailedBefore
	}

	defer func() {
		if r := recover(); r != nil {
			err = dl.handleError(addr, fmt.Errorf("recovering from panic in DWARF binary add2line: %v", r))
		}
	}()

	lines, err = dl.sourceLines(addr)
	if err != nil {
		return nil, dl.handleError(addr, err)
	}
	if len(lines) == 0 {
		dl.failed[addr] = struct{}{}
		delete(dl.attempts, addr)
		return nil, errors.New("could not find any frames for given address")
	}

	return lines, nil
}

func (dl *dwarfLiner) handleError(addr uint64, err error) error {
	if prev, ok := dl.attempts[addr]; ok {
		prev++
		if prev >= dl.attemptThreshold {
			dl.failed[addr] = struct{}{}
			delete(dl.attempts, addr)
		} else {
			dl.attempts[addr] = prev
		}
		return err
	}
	// First failed attempt
	dl.attempts[addr] = 1
	return err
}

func (dl *dwarfLiner) sourceLines(addr uint64) ([]metastore.LocationLine, error) {
	// The reader is positioned at byte offset 0 in the DWARF “info” section.
	er := dl.data.Reader()
	cu, err := er.SeekPC(addr)
	if err != nil {
		return nil, err
	}
	if cu == nil {
		return nil, errors.New("failed to find a corresponding dwarf entry for given address")
	}

	if err := dl.ensureLookUpTablesBuilt(cu); err != nil {
		return nil, err
	}

	lines := []metastore.LocationLine{}
	var tr *godwarf.Tree
	for _, t := range dl.subprograms[cu.Offset] {
		if t.ContainsPC(addr) {
			tr = t
			break
		}
	}
	if tr == nil {
		return lines, nil
	}

	name := tr.Entry.Val(dwarf.AttrName).(string)
	file, line := findLineInfo(dl.lineEntries[cu.Offset], tr.Ranges)
	lines = append(lines, metastore.LocationLine{
		Line: line,
		Function: dl.demangler.Demangle(&pb.Function{
			Name:     name,
			Filename: file,
		}),
	})

	// If pc is 0 then all inlined calls will be returned.
	for _, ch := range reader.InlineStack(tr, addr) {
		var name string
		if ch.Tag == dwarf.TagSubprogram {
			name = tr.Entry.Val(dwarf.AttrName).(string)
		} else {
			abstractOrigin := dl.abstractSubprograms[ch.Entry.Val(dwarf.AttrAbstractOrigin).(dwarf.Offset)]
			name = getFunctionName(abstractOrigin)
		}

		file, line := findLineInfo(dl.lineEntries[cu.Offset], ch.Ranges)
		lines = append(lines, metastore.LocationLine{
			Line: line,
			Function: dl.demangler.Demangle(&pb.Function{
				Name:     name,
				Filename: file,
			}),
		})
	}

	return lines, nil
}

func findLineInfo(entries []dwarf.LineEntry, rg [][2]uint64) (string, int64) {
	file := "?"
	var line int64 = 0
	i := sort.Search(len(entries), func(i int) bool {
		return entries[i].Address >= rg[0][0]
	})
	if i >= len(entries) {
		return file, line
	}

	le := dwarf.LineEntry{}
	pc := entries[i].Address
	if rg[0][0] <= pc && pc < rg[0][1] {
		le = entries[i]
		return le.File.Name, int64(le.Line)
	}

	return file, line
}

func getFunctionName(entry *dwarf.Entry) string {
	var name string
	for _, field := range entry.Field {
		if field.Attr == dwarf.AttrName {
			name = field.Val.(string)
		}
	}
	return name
}
