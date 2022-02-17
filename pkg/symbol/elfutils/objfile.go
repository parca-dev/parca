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

package elfutils

import (
	"debug/elf"
	"errors"
	"fmt"
	"sync"

	"github.com/parca-dev/parca/internal/pprof/elfexec"
)

type objFile struct {
	path    string
	buildID string

	// Ensures the base, baseErr and isData are computed once.
	baseOnce sync.Once
	base     uint64
	baseErr  error

	isData bool
	m      *mapping
}

func (f *objFile) ObjAddr(addr uint64) (uint64, error) {
	f.baseOnce.Do(func() { f.baseErr = f.computeBase(addr) })
	if f.baseErr != nil {
		return 0, f.baseErr
	}
	return addr - f.base, nil
}

func (f *objFile) BuildID() string {
	return f.buildID
}

// computeBase computes the relocation base for the given binary objFile only if
// the mapping field is set. It populates the base and isData fields and
// returns an error.
func (f *objFile) computeBase(addr uint64) error {
	if f == nil || f.m == nil {
		return nil
	}
	if addr < f.m.start || addr >= f.m.limit {
		return fmt.Errorf("specified address %x is outside the mapping range [%x, %x] for objFile %q", addr, f.m.start, f.m.limit, f.path)
	}
	ef, err := elfOpen(f.path)
	if err != nil {
		return fmt.Errorf("error parsing %s: %v", f.path, err)
	}
	defer ef.Close()

	ph, err := f.m.findProgramHeader(ef, addr)
	if err != nil {
		return fmt.Errorf("failed to find program header for objFile %q, ELF mapping %#v, address %x: %v", f.path, *f.m, addr, err)
	}

	base, err := elfexec.GetBase(&ef.FileHeader, ph, f.m.stextOffset, f.m.start, f.m.limit, f.m.offset)
	if err != nil {
		return err
	}
	f.base = base
	f.isData = ph != nil && ph.Flags&elf.PF_X == 0
	return nil
}

// mapping stores the parameters of a runtime mapping that are needed to
// identify the ELF segment associated with a mapping.
type mapping struct {
	// Runtime mapping parameters.
	start, limit, offset uint64
	// Offset of _stext symbol. Only defined for kernel images, nil otherwise.
	stextOffset *uint64
}

// findProgramHeader returns the program segment that matches the current
// mapping and the given address, or an error if it cannot find a unique program
// header.
func (m *mapping) findProgramHeader(ef *elf.File, addr uint64) (*elf.ProgHeader, error) {
	// For user space executables, we try to find the actual program segment that
	// is associated with the given mapping. Skip this search if limit <= start.
	// We cannot use just a check on the start address of the mapping to tell if
	// it's a kernel / .ko module mapping, because with quipper address remapping
	// enabled, the address would be in the lower half of the address space.

	if m.stextOffset != nil || m.start >= m.limit || m.limit >= (uint64(1)<<63) {
		// For the kernel, find the program segment that includes the .text section.
		return elfexec.FindTextProgHeader(ef), nil
	}

	// Fetch all the loadable segments.
	var phdrs []elf.ProgHeader
	for i := range ef.Progs {
		if ef.Progs[i].Type == elf.PT_LOAD {
			phdrs = append(phdrs, ef.Progs[i].ProgHeader)
		}
	}
	// Some ELF files don't contain any loadable program segments, e.g. .ko
	// kernel modules. It's not an error to have no header in such cases.
	if len(phdrs) == 0 {
		return nil, nil
	}
	// Get all program headers associated with the mapping.
	headers := elfexec.ProgramHeadersForMapping(phdrs, m.offset, m.limit-m.start)
	if len(headers) == 0 {
		return nil, errors.New("no program header matches mapping info")
	}
	if len(headers) == 1 {
		return headers[0], nil
	}

	// Use the objFile offset corresponding to the address to symbolize, to narrow
	// down the header.
	return elfexec.HeaderForFileOffset(headers, addr-m.start+m.offset)
}
