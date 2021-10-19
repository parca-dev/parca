// Copyright 2021 The Parca Authors
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
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/go-delve/delve/pkg/dwarf/godwarf"
	"github.com/go-delve/delve/pkg/dwarf/reader"
	"github.com/google/pprof/profile"
	lru "github.com/hashicorp/golang-lru"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

const (
	dwarfTreeCacheSize = 8192 // size of the dwarfTree cache of each mapping
)

type dwarfLiner struct {
	dwarfTreeCache simplelru.LRUCache
	demangler      *demangle.Demangler
	data           *dwarf.Data
}

func DWARF(demangler *demangle.Demangler, _ *profile.Mapping, path string) (*dwarfLiner, error) {
	// TODO(kakkoyun): Handle offset, start and limit for dynamically linked libraries.
	//objFile, err := s.bu.Open(file, m.Start, m.Limit, m.Offset)
	//if err != nil {
	//	return nil, fmt.Errorf("open object file: %w", err)
	//}
	objFile, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer objFile.Close()

	data, err := objFile.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to read DWARF data: %w", err)
	}
	cache, err := lru.New(dwarfTreeCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dwarf tree cache: %w", err)

	}
	return &dwarfLiner{
		demangler:      demangler,
		dwarfTreeCache: cache,
		data:           data,
	}, nil
}

func (dl *dwarfLiner) PCToLines(addr uint64) (lines []profile.Line, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovering from panic in DWARF binary add2line: %v", r)
		}
	}()

	lines, err = dl.sourceLines(addr)
	if err != nil {
		return nil, err
	}

	if len(lines) == 0 {
		return nil, errors.New("could not find any frames for given address")
	}

	return lines, nil
}

func (dl *dwarfLiner) sourceLines(addr uint64) ([]profile.Line, error) {
	// The reader is positioned at byte offset 0 in the DWARF “info” section.
	er := dl.data.Reader()
	cu, err := er.SeekPC(addr)
	if err != nil {
		return nil, err
	}
	if cu == nil {
		return nil, errors.New("failed to find a corresponding dwarf entry for given address")
	}

	// The reader is positioned at byte offset 0 in the DWARF “line” section.
	lr, err := dl.data.LineReader(cu)
	if err != nil {
		return nil, err
	}

	lineEntries := []dwarf.LineEntry{}
	for {
		le := dwarf.LineEntry{}
		err := lr.Next(&le)
		if err != nil {
			break
		}
		if le.IsStmt {
			lineEntries = append(lineEntries, le)
		}
	}

	subprograms := []*godwarf.Tree{}
	abstractSubprograms := map[dwarf.Offset]*dwarf.Entry{}
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
			break
		}

		if entry.Tag == dwarf.TagSubprogram {
			for _, field := range entry.Field {
				if field.Attr == dwarf.AttrInline {
					abstractSubprograms[entry.Offset] = entry
					continue outer
				}
			}

			tr, err := dl.getDWARFTree(entry)
			if err != nil {
				return nil, fmt.Errorf("failed to extract dwarf tree: %w", err)
			}

			if tr.ContainsPC(addr) {
				subprograms = append(subprograms, tr)
			}
		}
	}

	lines := []profile.Line{}
	for _, tr := range subprograms {
		name := tr.Entry.Val(dwarf.AttrName).(string)
		file, line := findLineInfo(lineEntries, tr.Ranges)
		lines = append(lines, profile.Line{
			Line: line,
			Function: dl.demangler.Demangle(&profile.Function{
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
				abstractOrigin := abstractSubprograms[ch.Entry.Val(dwarf.AttrAbstractOrigin).(dwarf.Offset)]
				name = getFunctionName(abstractOrigin)
			}

			file, line := findLineInfo(lineEntries, ch.Ranges)
			lines = append(lines, profile.Line{
				Line: line,
				Function: dl.demangler.Demangle(&profile.Function{
					Name:     name,
					Filename: file,
				}),
			})
		}
	}

	return lines, nil
}

func (dl *dwarfLiner) getDWARFTree(entry *dwarf.Entry) (*godwarf.Tree, error) {
	if tr, ok := dl.dwarfTreeCache.Get(entry.Offset); ok {
		return tr.(*godwarf.Tree), nil
	}
	tr, err := godwarf.LoadTree(entry.Offset, dl.data, 0)
	if err != nil {
		return nil, err
	}
	dl.dwarfTreeCache.Add(entry.Offset, tr)
	return tr, nil
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
