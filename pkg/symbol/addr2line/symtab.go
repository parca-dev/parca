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
	// mmapOffset is the offset of mapped segment within ELF file, e.g., 0x1000.
	mmapOffset uint64
	// mmapStart is the virtual address where segment was mapped, e.g., 0x401000.
	mmapStart uint64
	// isPIE indicates whether the ELF file is position independent executable.
	isPIE bool
}

// Symbols creates a new SymtabLiner.
func Symbols(logger log.Logger, filename string, f *elf.File, mmapOffset, mmapStart uint64, demangler *demangle.Demangler) (*SymtabLiner, error) {
	symbols, err := symtab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch symbols from object file: %w", err)
	}

	searcher := symbolsearcher.New(symbols)
	return &SymtabLiner{
		logger:     log.With(logger, "liner", "symtab"),
		searcher:   searcher,
		demangler:  demangler,
		filename:   filename,
		f:          f,
		mmapOffset: mmapOffset,
		mmapStart:  mmapStart,
		isPIE:      isPIE(f, mmapOffset, mmapStart),
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
	if lnr.isPIE {
		if addr < lnr.mmapStart {
			return nil, fmt.Errorf("address %x can't be lower than beginning of segment %x", addr, lnr.mmapStart)
		}
		// Distance between the sampled memory address and
		// beginning of the loaded segment (vm_start memory address).
		segmentDistance := addr - lnr.mmapStart
		// Sampled address adjusted to .symtab address range.
		addr = lnr.mmapOffset + segmentDistance
	}

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

// isPIE indicates whether the program is position independent executable.
// PIE is used by default in gcc for security measures,
// i.e., address space layout randomization.
func isPIE(f *elf.File, mmapOffset, mmapStart uint64) bool {
	// The executable segment usually maps to 0x401000 for non PIE programs,
	// and to a random address such as 0x5646e2188000 for PIE.
	if mmapStart == 0 {
		return false
	}

	// Find the mapped segment in ELF file.
	var segment elf.ProgHeader
	for i := range f.Progs {
		if f.Progs[i].Off == mmapOffset {
			segment = f.Progs[i].ProgHeader
			break
		}
	}
	isReadable := (segment.Flags & elf.PF_R) != 0
	isExecutable := (segment.Flags & elf.PF_X) != 0
	if segment.Type != elf.PT_LOAD || !isReadable || !isExecutable {
		return false
	}

	// In case of PIE, virtual address and file offset are equal
	// when looking at the ELF file,
	// but vm_start shown in /proc/$PID/maps will be a random high address,
	// e.g., 0x5646e2188000.
	//
	// Type Offset   VirtAddr           PhysAddr           FileSiz  MemSiz   Flg Align
	// LOAD 0x001000 0x0000000000001000 0x0000000000001000 0x0001ed 0x0001ed R E 0x1000
	isPIE := segment.Vaddr == segment.Off

	return isPIE
}
