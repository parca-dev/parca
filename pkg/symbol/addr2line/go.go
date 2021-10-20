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
	"debug/elf"
	"debug/gosym"
	"errors"
	"fmt"

	"github.com/google/pprof/profile"
)

func Go(path string) (func(addr uint64) ([]profile.Line, error), error) {
	tab, err := gosymtab(path)
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

		file, line, fn := tab.PCToLine(addr)
		name := "?"
		if fn != nil {
			name = fn.Name
		} else {
			file = "?"
			line = 0
		}

		// TODO(kakkoyun): Find a way to symbolize inline functions.
		lines = append(lines, profile.Line{
			Line: int64(line),
			Function: &profile.Function{
				Name:     name,
				Filename: file,
			},
		})

		return lines, nil
	}, nil
}

func gosymtab(path string) (*gosym.Table, error) {
	objFile, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer objFile.Close()

	var pclntab []byte
	if sec := objFile.Section(".gopclntab"); sec != nil {
		if sec.Type == elf.SHT_NOBITS {
			return nil, errors.New(".gopclntab section has no bits")
		}

		pclntab, err = sec.Data()
		if err != nil {
			return nil, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	if len(pclntab) <= 0 {
		return nil, errors.New(".gopclntab section has no bits")
	}

	var symtab []byte
	if sec := objFile.Section(".gosymtab"); sec != nil {
		symtab, _ = sec.Data()
	}

	var text uint64 = 0
	if sec := objFile.Section(".text"); sec != nil {
		text = sec.Addr
	}

	table, err := gosym.NewTable(symtab, gosym.NewLineTable(pclntab, text))
	if err != nil {
		return nil, fmt.Errorf("failed to build symtab or pclinetab: %w", err)
	}
	return table, nil
}
