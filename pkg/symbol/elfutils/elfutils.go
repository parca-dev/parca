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
