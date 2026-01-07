// Copyright 2024-2026 The Parca Authors
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

package symbolizer

import (
	"debug/elf"
	"fmt"

	"github.com/parca-dev/parca/pkg/profile"
)

func NormalizeAddress(addr uint64, ei profile.ExecutableInfo, m profile.Mapping) (uint64, error) {
	base, err := CalculateBase(ei, m, addr)
	if err != nil {
		return addr, fmt.Errorf("calculate base: %w", err)
	}

	return addr - base, nil
}

// Base determines the base address to subtract from virtual
// address to get symbol table address. For an executable, the base
// is 0. Otherwise, it's a shared library, and the base is the
// address where the mapping starts. The kernel needs special handling.
func CalculateBase(ei profile.ExecutableInfo, m profile.Mapping, addr uint64) (uint64, error) {
	h, err := ei.FindProgramHeader(m, addr)
	if err != nil {
		return 0, fmt.Errorf("find program header: %w", err)
	}

	if h == nil {
		return 0, nil
	}

	if m.StartAddr == 0 && m.Offset == 0 && (m.EndAddr == ^uint64(0) || m.EndAddr == 0) {
		// Some tools may introduce a fake mapping that spans the entire
		// address space. Assume that the address has already been
		// adjusted, so no additional base adjustment is necessary.
		return 0, nil
	}

	//nolint:exhaustive
	switch elf.Type(ei.ElfType) {
	case elf.ET_EXEC:
		return m.StartAddr - m.Offset + h.Off - h.Vaddr, nil
	case elf.ET_REL:
		if m.Offset != 0 {
			return 0, fmt.Errorf("don't know how to handle mapping.Offset")
		}
		return m.StartAddr, nil
	case elf.ET_DYN:

		// The program header, if not nil, indicates the offset in the file where
		// the executable segment is located (loadSegment.Off), and the base virtual
		// address where the first byte of the segment is loaded
		// (loadSegment.Vaddr). A file offset fx maps to a virtual (symbol) address
		// sx = fx - loadSegment.Off + loadSegment.Vaddr.
		//
		// Thus, a runtime virtual address x maps to a symbol address
		// sx = x - start + offset - loadSegment.Off + loadSegment.Vaddr.
		return m.StartAddr - m.Offset + h.Off - h.Vaddr, nil
	}

	return 0, fmt.Errorf("don't know how to handle FileHeader.Type %v", elf.Type(ei.ElfType))
}
