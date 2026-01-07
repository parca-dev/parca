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

package profile

import (
	"debug/elf"
	"errors"
	"fmt"
)

// A ProgHeader represents a single ELF program header.
type ProgHeader struct {
	Off   uint64
	Vaddr uint64
	Memsz uint64
}

type ExecutableInfo struct {
	ElfType          elf.Type
	TextProgHdrIndex int16
	ProgHeaders      []ProgHeader
}

type Mapping struct {
	StartAddr uint64
	EndAddr   uint64
	Offset    uint64
	File      string
}

func ExecutableInfoFromELF(f *elf.File) (ExecutableInfo, error) {
	// Fetch all the loadable segments.
	var phdrs []elf.ProgHeader
	for i := range f.Progs {
		if f.Progs[i].Type == elf.PT_LOAD {
			phdrs = append(phdrs, f.Progs[i].ProgHeader)
		}
	}

	idx := findTextProgHeader(f, phdrs)

	progHeaders := make([]ProgHeader, len(phdrs))
	for i, p := range phdrs {
		progHeaders[i] = ProgHeader{
			Off:   p.Off,
			Vaddr: p.Vaddr,
			Memsz: p.Memsz,
		}
	}

	return ExecutableInfo{
		ElfType:          f.Type,
		TextProgHdrIndex: idx,
		ProgHeaders:      progHeaders,
	}, nil
}

// findTextProgHeader finds the program segment header containing the .text
// section or -1 if the segment cannot be found.
func findTextProgHeader(f *elf.File, phdrs []elf.ProgHeader) int16 {
	for _, s := range f.Sections {
		if s.Name == ".text" {
			// Find the LOAD segment containing the .text section.
			for i, p := range phdrs {
				// Type           Offset   VirtAddr           PhysAddr           FileSiz  MemSiz   Flg Align
				// LOAD           0x001000 0x0000000000001000 0x0000000000001000 0x0001ed 0x0001ed R E 0x1000
				if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 && s.Addr >= p.Vaddr && s.Addr < p.Vaddr+p.Memsz {
					return int16(i)
				}
			}
		}
	}
	return -1
}

// FindProgramHeader returns the program segment that matches the current
// mapping and the given address, or an error if it cannot find a unique program
// header.
func (ei ExecutableInfo) FindProgramHeader(m Mapping, addr uint64) (*ProgHeader, error) {
	// For user space executables, we try to find the actual program segment that
	// is associated with the given mapping. Skip this search if limit <= start.
	if m.StartAddr >= m.EndAddr || uint64(m.EndAddr) >= (uint64(1)<<63) {
		return ei.textProgramHeader(), nil
	}

	// Some ELF files don't contain any loadable program segments, e.g. .ko
	// kernel modules. It's not an error to have no header in such cases.
	if len(ei.ProgHeaders) == 0 {
		return nil, nil //nolint:nilnil
	}
	// Get all program headers associated with the mapping.
	headers := ei.programHeadersForMapping(uint64(m.Offset), uint64(m.EndAddr)-uint64(m.StartAddr))
	if len(headers) == 0 {
		return nil, errors.New("no program header matches mapping info")
	}
	if len(headers) == 1 {
		return headers[0], nil
	}

	// Use the file offset corresponding to the address to symbolize, to narrow
	// down the header.
	return headerForFileOffset(headers, addr-uint64(m.StartAddr)+uint64(m.Offset))
}

func (ei ExecutableInfo) textProgramHeader() *ProgHeader {
	if ei.TextProgHdrIndex == -1 {
		return nil
	}
	return &ei.ProgHeaders[ei.TextProgHdrIndex]
}

// ProgramHeadersForMapping returns the program segment headers that overlap
// the runtime mapping with file offset mapOff and memory size mapSz. We skip
// over segments zero file size because their file offset values are unreliable.
// Even if overlapping, a segment is not selected if its aligned file offset is
// greater than the mapping file offset, or if the mapping includes the last
// page of the segment, but not the full segment and the mapping includes
// additional pages after the segment end.
// The function returns a slice of pointers to the headers in the input
// slice, which are valid only while phdrs is not modified or discarded.
//
// The original ProgHeaders must be Filesz > 0 and p.Type == elf.PT_LOAD.
func (ei ExecutableInfo) programHeadersForMapping(mapOff, mapSz uint64) []*ProgHeader {
	const (
		// pageSize defines the virtual memory page size used by the loader. This
		// value is dependent on the memory management unit of the CPU. The page
		// size is 4KB virtually on all the architectures that we care about, so we
		// define this metric as a constant. If we encounter architectures where
		// page sie is not 4KB, we must try to guess the page size on the system
		// where the profile was collected, possibly using the architecture
		// specified in the ELF file header.
		pageSize       = 4096
		pageOffsetMask = pageSize - 1
	)
	mapLimit := mapOff + mapSz
	var headers []*ProgHeader
	for i := range ei.ProgHeaders {
		p := &ei.ProgHeaders[i]
		segLimit := p.Off + p.Memsz
		// The segment must overlap the mapping.
		if mapOff < segLimit && p.Off < mapLimit {
			// If the mapping offset is strictly less than the page aligned segment
			// offset, then this mapping comes from a different segment, fixes
			// b/179920361.
			alignedSegOffset := uint64(0)
			if p.Off > (p.Vaddr & pageOffsetMask) {
				alignedSegOffset = p.Off - (p.Vaddr & pageOffsetMask)
			}
			if mapOff < alignedSegOffset {
				continue
			}
			// If the mapping starts in the middle of the segment, it covers less than
			// one page of the segment, and it extends at least one page past the
			// segment, then this mapping comes from a different segment.
			if mapOff > p.Off && (segLimit < mapOff+pageSize) && (mapLimit >= segLimit+pageSize) {
				continue
			}
			headers = append(headers, p)
		}
	}
	return headers
}

// HeaderForFileOffset attempts to identify a unique program header that
// includes the given file offset. It returns an error if it cannot identify a
// unique header.
func headerForFileOffset(headers []*ProgHeader, fileOffset uint64) (*ProgHeader, error) {
	var ph *ProgHeader
	for _, h := range headers {
		if fileOffset >= h.Off && fileOffset < h.Off+h.Memsz {
			if ph != nil {
				// Assuming no other bugs, this can only happen if we have two or
				// more small program segments that fit on the same page, and a
				// segment other than the last one includes uninitialized data, or
				// if the debug binary used for symbolization is stripped of some
				// sections, so segment file sizes are smaller than memory sizes.
				return nil, fmt.Errorf("found second program header (%#v) that matches file offset %x, first program header is %#v. Is this a stripped binary, or does the first program segment contain uninitialized data?", *h, fileOffset, *ph)
			}
			ph = h
		}
	}
	if ph == nil {
		return nil, fmt.Errorf("no program header matches file offset %x", fileOffset)
	}
	return ph, nil
}
