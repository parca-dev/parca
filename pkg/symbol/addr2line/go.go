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
	"github.com/parca-dev/parca/internal/go/cmd/objfile"
)

type liner interface {
	// PCToLine given a pc, returns the corresponding file, line, and function data.
	// If unknown, returns "",0,nil.
	PCToLine(uint64) (string, int, *gosym.Func)
}

func Go(path string) (func(addr uint64) ([]profile.Line, error), error) {
	var tab liner
	//tab, err := gosymtab(path)
	//if err != nil {
	//	return nil, fmt.Errorf("failed to create go symbtab: %w", err)
	//}

	// TODO(kakkoyun): Check if it worth the hassle!
	tab, err := gopclntab(path)
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

		// TODO(kakkoyun): Find a way for inline functions.
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

func gopclntab(path string) (liner, error) {
	f, err := objfile.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read objfile: %w", err)
	}
	defer f.Close()

	tab, err := f.PCLineTable()
	if err != nil {
		return nil, fmt.Errorf("failed to create pclinetab objfile: %w", err)
	}
	return tab, nil
}

func gosymtab(path string) (liner, error) {
	exe, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	var pclntab []byte
	if sec := exe.Section(".gopclntab"); sec != nil {
		if sec.Type == elf.SHT_NOBITS {
			return nil, errors.New(".gopclntab section has no bits")
		}

		// TODO(kakkoyun): Don't read just check existence!
		pclntab, err = sec.Data()
		if err != nil {
			return nil, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	if len(pclntab) <= 0 {
		return nil, errors.New(".gopclntab section has no bits")
	}

	var symtab []byte
	if sec := exe.Section(".gosymtab"); sec != nil {
		// TODO(kakkoyun): Don't read just check existence!
		symtab, _ = sec.Data()
	}

	var text uint64 = 0
	if sec := exe.Section(".text"); sec != nil {
		text = sec.Addr
	}

	table, err := gosym.NewTable(symtab, gosym.NewLineTable(pclntab, text))
	if err != nil {
		return nil, fmt.Errorf("failed to build symtab or pclinetab: %w", err)
	}
	return table, nil
}

func IsSymbolizableGoObjFile(path string) (bool, error) {
	// Checks ".note.go.buildid" section and symtab better to keep those sections in object file.
	exe, err := elf.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	isGo := false
	for _, s := range exe.Sections {
		if s.Name == ".note.go.buildid" {
			isGo = true
		}
	}

	// In case ".note.go.buildid" section is stripped, check for symbols.
	if !isGo {
		syms, err := exe.Symbols()
		if err != nil {
			return false, fmt.Errorf("failed to read symbols: %w", err)
		}
		for _, sym := range syms {
			name := sym.Name
			if name == "runtime.main" || name == "main.main" {
				isGo = true
			}
			if name == "runtime.buildVersion" {
				isGo = true
			}
		}
	}

	if !isGo {
		return false, nil
	}

	// Check if the Go binary symbolizable.
	// Go binaries has a special case. They use ".gopclntab" section to symbolize addresses.
	var pclntab []byte
	if sec := exe.Section(".gopclntab"); sec != nil {
		// TODO(kakkoyun): Don't read just check existence!
		pclntab, err = sec.Data()
		if err != nil {
			return false, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	return len(pclntab) > 0, nil
}
