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

package elfutils

import (
	"debug/elf"
	"fmt"
	"strings"
)

func HasDWARF(path string) (bool, error) {
	exe, err := elf.Open(path)
	if err != nil {
		return false, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	sections, err := getDWARFSections(exe)
	if err != nil {
		return false, fmt.Errorf("failed to read DWARF sections: %w", err)
	}

	return len(sections) > 0, nil
}

// A simplified and modified version of debug/elf.DWARF().
func getDWARFSections(f *elf.File) (map[string]struct{}, error) {
	dwarfSuffix := func(s *elf.Section) string {
		switch {
		case strings.HasPrefix(s.Name, ".debug_"):
			return s.Name[7:]
		case strings.HasPrefix(s.Name, ".zdebug_"):
			return s.Name[8:]
		case strings.HasPrefix(s.Name, "__debug_"): // macos
			return s.Name[8:]
		default:
			return ""
		}
	}

	// There are many DWARf sections, but these are the ones
	// the debug/dwarf package started with "abbrev", "info", "str", "line", "ranges".
	// Possible candidates for future: "loc", "loclists", "rnglists"
	sections := map[string]*string{"abbrev": nil, "info": nil, "str": nil, "line": nil, "ranges": nil}
	exists := map[string]struct{}{}
	for _, s := range f.Sections {
		suffix := dwarfSuffix(s)
		if suffix == "" {
			continue
		}
		if _, ok := sections[suffix]; !ok {
			continue
		}
		exists[suffix] = struct{}{}
	}

	return exists, nil
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
		// TODO(kakkoyun): Optimize. Don't read just check existence!
		pclntab, err = sec.Data()
		if err != nil {
			return false, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	return len(pclntab) > 0, nil
}
