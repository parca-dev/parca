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

	"github.com/google/pprof/profile"
)

func sourceLines(data *dwarf.Data, addr uint64, doDemangle bool) ([]profile.Line, error) {
	// Reader returns a new Reader for Data.
	// The reader is positioned at byte offset 0 in the DWARF “info” section.
	er := data.Reader()
	entry, err := er.SeekPC(addr)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, errors.New("failed to find a corresponding dwarf entry for given address")
	}
	fmt.Printf("SEEK Tag: %s\n", entry.Tag.GoString())

	lr, err := data.LineReader(entry)
	if err != nil {
		return nil, err
	}

	le := dwarf.LineEntry{}
	for {
		err := lr.Next(&le)
		if err != nil {
			break
		}
		fmt.Printf("SEEK Address: %x - %v File: %s Line: %d IsStmt: %v PrologueEnd: %v BasicBlock: %v\n", le.Address, le.Address, le.File.Name, le.Line, le.IsStmt, le.PrologueEnd, le.BasicBlock)
	}

	var inline = false
	lines := []profile.Line{}
	for {
		cu, err := er.Next()
		if err != nil {
			if err == io.EOF {
				// We've reached the end of DWARF entries
				break
			}
			continue
		}
		if cu == nil {
			break
		}
		if cu.Tag == dwarf.TagCompileUnit {
			break
		}

		// Check if this entry is a function
		if cu.Tag == dwarf.TagSubprogram {
			fmt.Printf("LOOP TAG: %s, Address: %x\n", cu.Tag.GoString(), cu.Offset)
			functionName := getFunctionName(cu)

			for _, field := range cu.Field {
				fmt.Println(field.Attr, field.Val, field.Class)
			}

			// Decode CU's line table.
			lr, err := data.LineReader(cu)
			if err != nil {
				return nil, err
			} else if lr == nil {
				//fmt.Println("Line reader empty")
				continue
			}
			le := dwarf.LineEntry{}
			for {
				err := lr.Next(&le)
				if err != nil {
					break
				}
				fmt.Printf("LOOP Address: %x - %v File: %s Line: %d IsStmt: %v PrologueEnd: %v\n", le.Address, le.Address, le.File.Name, le.Line, le.IsStmt, le.PrologueEnd)
			}

			lines = append(lines, profile.Line{
				Line: int64(le.Line),
				Function: &profile.Function{
					Name:     functionName,
					Filename: le.File.Name,
				},
			})
			continue
		}

		if entry.Tag == dwarf.TagInlinedSubroutine {
			fmt.Printf("LOOP TAG: %s\n", cu.Tag.GoString())
			// Only some entry types, such as TagCompileUnit or TagSubprogram, have PC ranges;
			//  for others, this will return nil with no error.
			rg, err := data.Ranges(cu)
			if err != nil {
				continue
			}
			fmt.Printf("LOOP Ranges ANY, %v\n", rg)
			if len(rg) == 1 {
				fmt.Printf("LOOP Ranges, Address: %v Addr: %x\n", addr, addr)
				fmt.Printf("LOOP Ranges, Low: %x High: %x\n", rg[0][0], rg[0][1])
				fmt.Printf("LOOP Range TAG: %s\n", cu.Tag.GoString())
				if rg[0][0] <= addr && rg[0][1] > addr {
					inline = true
					break
				}
			}
		}
	}

	if inline {
		//	err := lr.SeekPC(addr, &line)
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	err = lr.SeekPC(line.Address-1, &line)
		//	if err != nil {
		//		return nil, err
		//	}
		//
		//	if line.Line == 0 {
		//		err := lr.SeekPC(addr, &line)
		//		if err != nil {
		//			return nil, err
		//		}
		//		var line2 dwarf.LineEntry
		//		lr.Next(&line2)
		//		if line2.Line != 0 {
		//			line = line2
		//		}
		//	}
		//} else {
		//	err = lr.SeekPC(addr, &line)
		//	if err != nil {
		//		return nil, err
		//	}
	}

	return lines, nil
}

func getFunctionName(entry *dwarf.Entry) string {
	var functionName string
	for _, field := range entry.Field {
		if field.Attr == dwarf.AttrName {
			functionName = field.Val.(string)
			// TODO(kakkoyun): Remove!
			fmt.Println(functionName)
		}
	}
	return functionName
}

func DWARF(_ *profile.Mapping, path string) (func(addr uint64) ([]profile.Line, error), error) {
	// TODO(kakkoyun): Handle offset, start and limit?
	//objFile, err := s.bu.Open(file, m.Start, m.Limit, m.Offset)
	//if err != nil {
	//	return nil, fmt.Errorf("open object file: %w", err)
	//}

	exe, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	//syms, err := exe.Symbols()
	//if err != nil {
	//	return nil, fmt.Errorf("failed to read symbols: %w", err)
	//}

	data, err := exe.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to read DWARF data: %w", err)
	}

	return func(addr uint64) ([]profile.Line, error) {
		// TODO(kakkoyun): Multiple frames? Inlined functions?
		lines, err := sourceLines(data, addr, true)
		if err != nil {
			return nil, err
		}

		if len(lines) == 0 {
			return nil, errors.New("could not find any frames for given address")
		}

		return lines, nil
	}, nil
}
