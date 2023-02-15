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
//

package addr2line

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/go-kit/log"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/symbolsearcher"
)

const (
	pltSuffix = "@plt" // add pltSuffix for plt symbol to keep consistent with perf
)

// SymtabLiner is a liner which utilizes .symtab and .dynsym sections.
type SymtabLiner struct {
	logger log.Logger

	demangler *demangle.Demangler
	searcher  symbolsearcher.Searcher

	filename string
	f        *elf.File
}

// Symbols creates a new SymtabLiner.
func Symbols(logger log.Logger, filename string, f *elf.File, demangler *demangle.Demangler) (*SymtabLiner, error) {
	symbols, err := symtab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch symbols from object file: %w", err)
	}

	searcher := symbolsearcher.New(symbols)
	return &SymtabLiner{
		logger:    log.With(logger, "liner", "symtab"),
		searcher:  searcher,
		demangler: demangler,
		filename:  filename,
		f:         f,
	}, nil
}

func (lnr *SymtabLiner) Close() error {
	return lnr.f.Close()
}

func (lnr *SymtabLiner) File() string {
	return lnr.filename
}

func (lnr *SymtabLiner) PCRange() ([2]uint64, error) {
	return lnr.searcher.PCRange()
}

// PCToLines looks up the line number information for a program counter (memory address).
func (lnr *SymtabLiner) PCToLines(addr uint64) (lines []profile.LocationLine, err error) {
	name, err := lnr.searcher.Search(addr)
	if err != nil {
		return nil, err
	}

	var (
		file = "?"
		line int64 // 0
	)

	// plt symbol suffix with pltSuffix
	// to demangle name, we should remove the pltSuffix first
	// and then add it to demangled name
	isplt := strings.HasSuffix(name, pltSuffix)
	result := lnr.demangler.Demangle(&pb.Function{
		SystemName: strings.TrimSuffix(name, pltSuffix),
		Filename:   file,
	})
	if isplt {
		result.Name = result.Name + pltSuffix
	}
	lines = append(lines, profile.LocationLine{
		Line:     line,
		Function: result,
	})
	return lines, nil
}

// symtab returns symbols from the symbol table extracted from the ELF file f.
// The symbols are sorted by their memory addresses in ascending order
// to facilitate searching.
func symtab(objFile *elf.File) ([]elf.Symbol, error) {
	syms, sErr := objFile.Symbols()
	dynSyms, dErr := objFile.DynamicSymbols()

	var pltSymbols []elf.Symbol
	// see symbol-elf.c/dso__synthesize_plt_symbols
	if pltRelSection, pltSection := objFile.Section(".rela.plt"), objFile.Section(".plt"); dErr == nil &&
		pltRelSection != nil && pltSection != nil &&
		objFile.Sections[pltRelSection.Link].Type == elf.SHT_DYNSYM {
		data, err := io.ReadAll(pltRelSection.Open())
		if err != nil {
			return nil, fmt.Errorf("failed to data of .rela.plt section:%s", err)
		}
		var rela elf.Rela64
		b := bytes.NewReader(data)

		// in perf script, it use pltSection.Offset
		// but the computeBase return symbol address, which require pltSection.Addr
		// - ET_EXEC, pltSection.Addr and  pltSection.Offset not same
		// - ET_DYNSYM, pltSection.Addr and  pltSection.Offset is same
		off := pltSection.Addr
		for b.Len() > 0 {
			off += pltSection.Entsize
			err := binary.Read(b, objFile.ByteOrder, &rela)
			if err != nil {
				return nil, fmt.Errorf("read plt section error, err:%s, section:%v", err, *pltRelSection)
			}
			// see applyRelocationsAMD64 go1.19.3/src/debug/elf/file.go:664
			i := rela.Info >> 32
			if i > uint64(len(dynSyms)) || i == 0 {
				continue
			}
			s := dynSyms[i-1]

			pltSymbols = append(pltSymbols, elf.Symbol{
				Name:    s.Name + pltSuffix,
				Info:    elf.ST_INFO(elf.STB_GLOBAL, elf.STT_FUNC),
				Section: elf.SectionIndex(1), // just to pass elfSymIsFunction's section check
				Value:   off,
				Size:    pltRelSection.Entsize,
				Version: "",
				Library: "",
			})
		}
	}

	if sErr != nil && dErr != nil {
		return nil, fmt.Errorf("failed to read symbol sections: %w", sErr)
	}

	syms = append(syms, append(dynSyms, pltSymbols...)...)
	return syms, nil
}
