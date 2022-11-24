// Copyright 2022 The Parca Authors
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

package elfutils

import (
	"debug/dwarf"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/go-delve/delve/pkg/dwarf/godwarf"
	"github.com/go-delve/delve/pkg/dwarf/reader"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

// DebugInfoFile is the interface implemented by symbolizers that use DWARF debug info.
type DebugInfoFile interface {
	// SourceLines returns the resolved source lines for a given address.
	SourceLines(addr uint64) ([]profile.LocationLine, error)
}

// debugInfoFile is a symbolizer that uses DWARF debug info to symbolize addresses.
type debugInfoFile struct {
	demangler *demangle.Demangler

	debugData           *dwarf.Data
	lineEntries         map[dwarf.Offset][]dwarf.LineEntry
	subprograms         map[dwarf.Offset][]*godwarf.Tree
	abstractSubprograms map[dwarf.Offset]*dwarf.Entry
	lineFiles           map[dwarf.Offset][]*dwarf.LineFile
}

// NewDebugInfoFile creates a new DebugInfoFile symbolizer.
func NewDebugInfoFile(f *elf.File, demangler *demangle.Demangler) (DebugInfoFile, error) {
	debugData, err := f.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to read DWARF data: %w", err)
	}

	result := &debugInfoFile{
		demangler: demangler,

		debugData:           debugData,
		lineEntries:         make(map[dwarf.Offset][]dwarf.LineEntry),
		subprograms:         make(map[dwarf.Offset][]*godwarf.Tree),
		abstractSubprograms: make(map[dwarf.Offset]*dwarf.Entry),
		lineFiles:           make(map[dwarf.Offset][]*dwarf.LineFile),
	}
	if err = result.buildAbstractSubprograms(); err != nil {
		return nil, err
	}
	return result, nil
}

// buildAbstractSubprograms will range over all compile unit, build abstractSubprograms
// cause inline function will cover multi package.
func (f *debugInfoFile) buildAbstractSubprograms() error {
	er := f.debugData.Reader()
	_, err := er.Next()
	if err != nil {
		return errors.New("failed to read entry")
	}
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

		if entry.Tag == dwarf.TagSubprogram {
			for _, field := range entry.Field {
				if field.Attr == dwarf.AttrInline {
					f.abstractSubprograms[entry.Offset] = entry
					break
				}
			}
		}
	}
	return nil
}

// SourceLines returns the resolved source lines for a program counter (memory address).
//
// It reads DWARF sections (info, line) which include several lookup tables and
// tries to find the name of the function that address belongs to.
// After that it tries to find the corresponding source file and line information.
func (f *debugInfoFile) SourceLines(addr uint64) ([]profile.LocationLine, error) {
	// The reader is positioned at byte offset 0 in the DWARF “info” section.
	// It allows reading Entry structures that are arranged in a tree.
	er := f.debugData.Reader()
	// SeekPC returns the Entry for the compilation unit that includes program counter,
	// and positions the reader to read the children of that unit.
	cu, err := er.SeekPC(addr)
	if err != nil {
		return nil, err
	}
	if cu == nil {
		return nil, errors.New("failed to find a corresponding dwarf entry for given address")
	}

	if err := f.ensureLookUpTablesBuilt(cu); err != nil {
		return nil, err
	}

	lines := []profile.LocationLine{}
	var tr *godwarf.Tree
	for _, t := range f.subprograms[cu.Offset] {
		if t.ContainsPC(addr) {
			tr = t
			break
		}
	}
	if tr == nil {
		return lines, nil
	}

	// If pc is 0 then all inlined calls will be returned.
	for _, ch := range reader.InlineStack(tr, addr) {
		var name string
		if ch.Tag == dwarf.TagSubprogram {
			name = tr.Entry.Val(dwarf.AttrName).(string)
		} else {
			abstractOrigin := f.abstractSubprograms[ch.Entry.Val(dwarf.AttrAbstractOrigin).(dwarf.Offset)]
			name = getFunctionName(abstractOrigin)
		}

		file := f.lineFiles[cu.Offset][ch.Entry.Val(dwarf.AttrCallFile).(int64)].Name
		line := ch.Entry.Val(dwarf.AttrCallLine).(int64)
		lines = append(lines, profile.LocationLine{
			Line: line,
			Function: f.demangler.Demangle(&pb.Function{
				Name:     name,
				Filename: file,
			}),
		})
	}

	name, ok := tr.Entry.Val(dwarf.AttrName).(string)
	if !ok {
		name = ""
	}

	// address correspond line must in last position
	file, line := findLineInfo(f.lineEntries[cu.Offset], addr)
	lines = append(lines, profile.LocationLine{
		Line: line,
		Function: f.demangler.Demangle(&pb.Function{
			Name:     name,
			Filename: file,
		}),
	})

	moveLinesForwardOneStep(lines)
	return lines, nil
}

// moveLinesForwardOneStep move each LocationLine's line filed move forward one step
// to get right line number.
func moveLinesForwardOneStep(lines []profile.LocationLine) {
	if len(lines) <= 1 {
		return
	}
	var last int64
	last = lines[len(lines)-1].Line
	for i := range lines {
		cur := lines[i].Line
		lines[i].Line = last
		last = cur
	}
}

func (f *debugInfoFile) ensureLookUpTablesBuilt(cu *dwarf.Entry) error {
	if _, ok := f.lineEntries[cu.Offset]; ok {
		// Already created.
		return nil
	}

	// The reader is positioned at byte offset 0 in the DWARF “line” section.
	lr, err := f.debugData.LineReader(cu)
	if err != nil {
		return err
	}
	if lr == nil {
		return errors.New("failed to initialize line reader")
	}

	f.lineFiles[cu.Offset] = lr.Files()

	for {
		le := dwarf.LineEntry{}
		err := lr.Next(&le)
		if err != nil {
			break
		}
		if le.IsStmt {
			f.lineEntries[cu.Offset] = append(f.lineEntries[cu.Offset], le)
		}
	}

	er := f.debugData.Reader()
	// The reader is positioned at byte offset of compile unit in the DWARF “info” section.
	er.Seek(cu.Offset)
	entry, err := er.Next()
	if err != nil || entry == nil {
		return errors.New("failed to read entry for compile unit")
	}

	if entry.Tag != dwarf.TagCompileUnit {
		return errors.New("failed to find entry for compile unit")
	}

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
			// Extract the tree of debug_info entries rooted at given offset.
			tr, err := godwarf.LoadTree(entry.Offset, f.debugData, 0)
			if err != nil {
				return fmt.Errorf("failed to extract dwarf tree: %w", err)
			}

			f.subprograms[cu.Offset] = append(f.subprograms[cu.Offset], tr)
		}
	}

	return nil
}

// findLineInfo looks up a file name and a line number
// in an ordered list DWARF entries (rows in a DWARF "line" table)
// by pc.
func findLineInfo(entries []dwarf.LineEntry, pc uint64) (string, int64) {
	var (
		file = "?"
		line int64 // 0
	)

	next := sort.Search(len(entries), func(i int) bool {
		return entries[i].Address > pc
	})

	if i := next - 1; i >= 0 && next != len(entries) {
		// entries[i].address <= pc < entries[i + 1]
		e := entries[i]
		file, line = e.File.Name, int64(e.Line)
	}

	return file, line
}

func getFunctionName(entry *dwarf.Entry) string {
	name := "?"
	if entry != nil {
		for _, field := range entry.Field {
			if field.Attr == dwarf.AttrName {
				name = field.Val.(string)
			}
		}
	}
	return name
}
