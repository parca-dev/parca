package addr2line

import (
	"debug/elf"
	"fmt"

	"github.com/google/pprof/profile"
)

func Symtab(path string) (func(addr uint64) ([]profile.Line, error), error) {
	exe, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open elf: %w", err)
	}
	defer exe.Close()

	syms, err := exe.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read symbols: %w", err)
	}

	symbolsMap := make(map[uint64]elf.Symbol)
	for _, sym := range syms {
		//if symbol.Size == 0 {
		//	continue
		//}
		//
		//// skip over any symbols that aren't functinons/methods
		//if symbol.Info != byte(2) && symbol.Info != byte(18) {
		//	continue
		//}

		if sym.Info != 0 && len(sym.Name) > 0 && sym.Value > 0 {
			symbolsMap[sym.Value] = sym
		}
	}

	// This is the best we can achieve. We just have the function name.
	return func(addr uint64) (lines []profile.Line, err error) {
		return []profile.Line{{
			Function: &profile.Function{
				Name: symbolsMap[addr].Name,
			},
		}}, nil
	}, nil
}
