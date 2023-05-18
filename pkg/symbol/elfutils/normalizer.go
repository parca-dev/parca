package elfutils

import (
	"debug/elf"
)

// BaseAddress returns a base address for the ELF file so
// sampled addresses can be normalized by subtracting the base
// from those addresses.
// This normalization is needed for PIE (position independent executable)
// which is used by default in gcc for security measures,
// i.e., address space layout randomization.
func BaseAddress(f *elf.File, start, offset uint64) uint64 {
	// Find the mapped segment in ELF file.
	var segment elf.ProgHeader
	for i := range f.Progs {
		if f.Progs[i].Off == offset {
			segment = f.Progs[i].ProgHeader
			break
		}
	}
	isReadable := (segment.Flags & elf.PF_R) != 0
	isExecutable := (segment.Flags & elf.PF_X) != 0
	if segment.Type != elf.PT_LOAD || !isReadable || !isExecutable {
		return 0
	}

	// In case of PIE, virtual address and file offset are equal
	// when looking at the ELF file,
	// but vm_start shown in /proc/$PID/maps will be a random high address,
	// e.g., 0x5646e2188000.
	//
	// Type Offset   VirtAddr           PhysAddr           FileSiz  MemSiz   Flg Align
	// LOAD 0x001000 0x0000000000001000 0x0000000000001000 0x0001ed 0x0001ed R E 0x1000
	// vs
	// LOAD 0x001000 0x0000000000401000 0x0000000000401000 0x00019d 0x00019d R E 0x1000
	//
	// The executable segment usually maps to 0x401000 for non PIE programs,
	// and to a random address such as 0x5646e2188000 for PIE.
	isPIE := segment.Vaddr == segment.Off
	if !isPIE {
		return 0
	}

	return start - segment.Vaddr
}
