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
//

package addr2line

import (
	"debug/elf"
	"errors"
	"fmt"
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
)

// SymtabLiner is a liner which utilizes .symtab and .dynsym sections.
type SymtabLiner struct {
	logger log.Logger

	// symbols contains sorted symbols.
	symbols   []elf.Symbol
	demangler demangle.Demangler
}

// Symbols creates a new SymtabLiner.
func Symbols(logger log.Logger, f *elf.File, demangler demangle.Demangler) (*SymtabLiner, error) {
	symbols, err := symtab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch symbols from object file: %w", err)
	}

	return &SymtabLiner{
		logger:    log.With(logger, "liner", "symtab"),
		symbols:   symbols,
		demangler: demangler,
	}, nil
}

// PCToLines looks up the line number information for a program counter (memory address).
func (lnr *SymtabLiner) PCToLines(addr uint64) (lines []profile.LocationLine, err error) {
	i := sort.Search(len(lnr.symbols), func(i int) bool {
		return lnr.symbols[i].Value > addr
	})

	if i < 1 || i > len(lnr.symbols) {
		level.Debug(lnr.logger).Log("msg", "failed to find symbol for address", "addr", addr)
		return nil, errors.New("failed to find symbol for address")
	}

	var (
		file = "?"
		line int64 // 0
	)

	lines = append(lines, profile.LocationLine{
		Line: line,
		Function: lnr.demangler.Demangle(&pb.Function{
			SystemName: lnr.symbols[i-1].Name,
			Filename:   file,
		}),
	})
	return lines, nil
}

// symtab returns symbols from the symbol table extracted from the ELF file f.
// The symbols are sorted by their memory addresses in ascending order
// to facilitate searching.
func symtab(objFile *elf.File) ([]elf.Symbol, error) {
	syms, sErr := objFile.Symbols()

	if sErr != nil {
		return nil, fmt.Errorf("failed to read symbol sections: %w", sErr)
	}

	sort.SliceStable(syms, func(i, j int) bool {
		return syms[i].Value < syms[j].Value
	})

	return syms, nil
}
