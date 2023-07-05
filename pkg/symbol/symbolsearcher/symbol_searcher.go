// Copyright 2022-2023 The Parca Authors
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

package symbolsearcher

import (
	"debug/elf"
	"errors"
	"sort"
	"strings"
)

type Searcher struct {
	symbols []elf.Symbol
}

func New(syms []elf.Symbol) Searcher {
	newSyms := make([]elf.Symbol, 0, len(syms))
	for _, s := range syms {
		if isFunction(s) {
			newSyms = append(newSyms, s)
		}
	}

	// slice stable sort to keep output consistent
	sort.SliceStable(newSyms, func(i, j int) bool {
		if newSyms[i].Value != newSyms[j].Value {
			return newSyms[i].Value < newSyms[j].Value
		}
		return chooseBestSymbol(newSyms[i], newSyms[j])
	})
	return Searcher{
		symbols: newSyms,
	}
}

func (s Searcher) Search(addr uint64) (string, error) {
	i := sort.Search(len(s.symbols), func(i int) bool {
		sym := s.symbols[i]
		return sym.Value > addr
	})
	if i == 0 ||
		// addr < sym[i-1]
		addr < s.symbols[i-1].Value {
		return "", errors.New("failed to find symbol for address")
	}

	// sym[i-1] <= addr < sym[i]
	i--
	return s.symbols[i].Name, nil
}

func (s Searcher) PCRange() ([2]uint64, error) {
	if len(s.symbols) == 0 {
		return [2]uint64{}, errors.New("no symbols found")
	}

	return [2]uint64{
		s.symbols[0].Value,
		s.symbols[len(s.symbols)-1].Value + s.symbols[len(s.symbols)-1].Size,
	}, nil
}

// copy from symbol-elf.c/elf_sym__is_function.
func isFunction(s elf.Symbol) bool {
	return elf.ST_TYPE(s.Info) == elf.STT_FUNC && s.Name != "" && s.Section != elf.SHN_UNDEF
}

// copy from symbol.c/choose_best_symbol.
func chooseBestSymbol(syma, symb elf.Symbol) bool {
	/* Prefer a symbol with non zero length */
	if symb.Size == 0 && syma.Size > 0 {
		return false
	} else if syma.Size == 0 && symb.Size > 0 {
		return true
	}

	/* Prefer a non weak symbol over a weak one */
	a := elf.ST_BIND(syma.Info) == elf.STB_WEAK
	b := elf.ST_BIND(symb.Info) == elf.STB_WEAK
	if b && !a {
		return false
	}
	if a && !b {
		return true
	}

	/* Prefer a global symbol over a non global one */
	a = elf.ST_BIND(syma.Info) == elf.STB_GLOBAL
	b = elf.ST_BIND(symb.Info) == elf.STB_GLOBAL
	if a && !b {
		return false
	}
	if b && !a {
		return true
	}

	/* Prefer a symbol with less underscores */
	aCount := prefixUnderscoresCount(syma.Name)
	bCount := prefixUnderscoresCount(symb.Name)
	if bCount > aCount {
		return false
	} else if aCount > bCount {
		return true
	}

	/* Choose the symbol with the longest name */
	na := len(syma.Name)
	nb := len(symb.Name)
	if na > nb {
		return false
	} else if na < nb {
		return true
	}

	/* Avoid "SyS" kernel syscall aliases */
	if strings.HasPrefix(syma.Name, "SyS") || strings.HasPrefix(syma.Name, "compat_SyS") {
		return true
	}

	/* Finally, if we can't distinguish them in any other way, try to
	   get consistent results by sorting the symbols by name.  */
	return syma.Name < symb.Name
}

func prefixUnderscoresCount(s string) int {
	n := 0
	for len(s) > 0 {
		sub := s[:1]
		if sub != "_" {
			break
		}
		n++
		s = s[1:]
	}
	return n
}
