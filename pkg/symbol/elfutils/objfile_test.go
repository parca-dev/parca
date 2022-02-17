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
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenMalformedELF(t *testing.T) {
	// Test that opening a malformed ELF objFile will report an error containing
	// the word "ELF".
	_, err := Open(filepath.Join("../../../internal/pprof/binutils/testdata", "malformed_elf"), 0, 0, 0)
	if err == nil {
		t.Fatalf("Open: unexpected success")
	}
	if !strings.Contains(err.Error(), "ELF") {
		t.Errorf("Open: got %v, want error containing 'ELF'", err)
	}
}

func TestComputeBase(t *testing.T) {
	tinyExecFile := &elf.File{
		FileHeader: elf.FileHeader{Type: elf.ET_EXEC},
		Progs: []*elf.Prog{
			{ProgHeader: elf.ProgHeader{Type: elf.PT_PHDR, Flags: elf.PF_R | elf.PF_X, Off: 0x40, Vaddr: 0x400040, Paddr: 0x400040, Filesz: 0x1f8, Memsz: 0x1f8, Align: 8}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_INTERP, Flags: elf.PF_R, Off: 0x238, Vaddr: 0x400238, Paddr: 0x400238, Filesz: 0x1c, Memsz: 0x1c, Align: 1}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_X, Off: 0, Vaddr: 0, Paddr: 0, Filesz: 0xc80, Memsz: 0xc80, Align: 0x200000}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_W, Off: 0xc80, Vaddr: 0x200c80, Paddr: 0x200c80, Filesz: 0x1f0, Memsz: 0x1f0, Align: 0x200000}},
		},
	}
	tinyBadBSSExecFile := &elf.File{
		FileHeader: elf.FileHeader{Type: elf.ET_EXEC},
		Progs: []*elf.Prog{
			{ProgHeader: elf.ProgHeader{Type: elf.PT_PHDR, Flags: elf.PF_R | elf.PF_X, Off: 0x40, Vaddr: 0x400040, Paddr: 0x400040, Filesz: 0x1f8, Memsz: 0x1f8, Align: 8}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_INTERP, Flags: elf.PF_R, Off: 0x238, Vaddr: 0x400238, Paddr: 0x400238, Filesz: 0x1c, Memsz: 0x1c, Align: 1}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_X, Off: 0, Vaddr: 0, Paddr: 0, Filesz: 0xc80, Memsz: 0xc80, Align: 0x200000}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_W, Off: 0xc80, Vaddr: 0x200c80, Paddr: 0x200c80, Filesz: 0x100, Memsz: 0x1f0, Align: 0x200000}},
			{ProgHeader: elf.ProgHeader{Type: elf.PT_LOAD, Flags: elf.PF_R | elf.PF_W, Off: 0xd80, Vaddr: 0x400d80, Paddr: 0x400d80, Filesz: 0x90, Memsz: 0x90, Align: 0x200000}},
		},
	}

	for _, tc := range []struct {
		desc       string
		file       *elf.File
		openErr    error
		mapping    *mapping
		addr       uint64
		wantError  bool
		wantBase   uint64
		wantIsData bool
	}{
		{
			desc:       "no elf mapping, no error",
			mapping:    nil,
			addr:       0x1000,
			wantBase:   0,
			wantIsData: false,
		},
		{
			desc:      "address outside mapping bounds means error",
			file:      &elf.File{},
			mapping:   &mapping{start: 0x2000, limit: 0x5000, offset: 0x1000},
			addr:      0x1000,
			wantError: true,
		},
		{
			desc:      "elf.Open failing means error",
			file:      &elf.File{FileHeader: elf.FileHeader{Type: elf.ET_EXEC}},
			openErr:   errors.New("elf.Open failed"),
			mapping:   &mapping{start: 0x2000, limit: 0x5000, offset: 0x1000},
			addr:      0x4000,
			wantError: true,
		},
		{
			desc:       "no loadable segments, no error",
			file:       &elf.File{FileHeader: elf.FileHeader{Type: elf.ET_EXEC}},
			mapping:    &mapping{start: 0x2000, limit: 0x5000, offset: 0x1000},
			addr:       0x4000,
			wantBase:   0,
			wantIsData: false,
		},
		{
			desc:      "unsupported executable type, Get Base returns error",
			file:      &elf.File{FileHeader: elf.FileHeader{Type: elf.ET_NONE}},
			mapping:   &mapping{start: 0x2000, limit: 0x5000, offset: 0x1000},
			addr:      0x4000,
			wantError: true,
		},
		{
			desc:       "tiny objFile select executable segment by offset",
			file:       tinyExecFile,
			mapping:    &mapping{start: 0x5000000, limit: 0x5001000, offset: 0x0},
			addr:       0x5000c00,
			wantBase:   0x5000000,
			wantIsData: false,
		},
		{
			desc:       "tiny objFile select data segment by offset",
			file:       tinyExecFile,
			mapping:    &mapping{start: 0x5200000, limit: 0x5201000, offset: 0x0},
			addr:       0x5200c80,
			wantBase:   0x5000000,
			wantIsData: true,
		},
		{
			desc:      "tiny objFile offset outside any segment means error",
			file:      tinyExecFile,
			mapping:   &mapping{start: 0x5200000, limit: 0x5201000, offset: 0x0},
			addr:      0x5200e70,
			wantError: true,
		},
		{
			desc:       "tiny objFile with bad BSS segment selects data segment by offset in initialized section",
			file:       tinyBadBSSExecFile,
			mapping:    &mapping{start: 0x5200000, limit: 0x5201000, offset: 0x0},
			addr:       0x5200d79,
			wantBase:   0x5000000,
			wantIsData: true,
		},
		{
			desc:      "tiny objFile with bad BSS segment with offset in uninitialized section means error",
			file:      tinyBadBSSExecFile,
			mapping:   &mapping{start: 0x5200000, limit: 0x5201000, offset: 0x0},
			addr:      0x5200d80,
			wantError: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			elfOpen = func(_ string) (*elf.File, error) {
				return tc.file, tc.openErr
			}
			t.Cleanup(func() {
				elfOpen = elf.Open
			})
			f := objFile{m: tc.mapping}
			err := f.computeBase(tc.addr)
			if (err != nil) != tc.wantError {
				t.Errorf("got error %v, want any error=%v", err, tc.wantError)
			}
			if err != nil {
				return
			}
			if f.base != tc.wantBase {
				t.Errorf("got base %x, want %x", f.base, tc.wantBase)
			}
			if f.isData != tc.wantIsData {
				t.Errorf("got isData %v, want %v", f.isData, tc.wantIsData)
			}
		})
	}
}

func TestELFObjAddr(t *testing.T) {
	// The exe_linux_64 has two loadable program headers:
	//  LOAD           0x0000000000000000 0x0000000000400000 0x0000000000400000
	//                 0x00000000000006fc 0x00000000000006fc  R E    0x200000
	//  LOAD           0x0000000000000e10 0x0000000000600e10 0x0000000000600e10
	//                 0x0000000000000230 0x0000000000000238  RW     0x200000
	name := filepath.Join("../../../internal/pprof/binutils/testdata", "exe_linux_64")

	for _, tc := range []struct {
		desc                 string
		start, limit, offset uint64
		wantOpenError        bool
		addr                 uint64
		wantObjAddr          uint64
		wantAddrError        bool
	}{
		{"exec mapping, good address", 0x5400000, 0x5401000, 0, false, 0x5400400, 0x400400, false},
		{"exec mapping, address outside segment", 0x5400000, 0x5401000, 0, false, 0x5400800, 0, true},
		{"short data mapping, good address", 0x5600e00, 0x5602000, 0xe00, false, 0x5600e10, 0x600e10, false},
		{"short data mapping, address outside segment", 0x5600e00, 0x5602000, 0xe00, false, 0x5600e00, 0x600e00, false},
		{"page aligned data mapping, good address", 0x5600000, 0x5602000, 0, false, 0x5601000, 0x601000, false},
		{"page aligned data mapping, address outside segment", 0x5600000, 0x5602000, 0, false, 0x5601048, 0, true},
		{"bad objFile offset, no matching segment", 0x5600000, 0x5602000, 0x2000, false, 0x5600e10, 0, true},
		{"large mapping size, match by sample offset", 0x5600000, 0x5603000, 0, false, 0x5600e10, 0x600e10, false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			o, err := open(name, tc.start, tc.limit, tc.offset)
			if (err != nil) != tc.wantOpenError {
				t.Errorf("openELF got error %v, want any error=%v", err, tc.wantOpenError)
			}
			if err != nil {
				return
			}
			got, err := o.ObjAddr(tc.addr)
			if (err != nil) != tc.wantAddrError {
				t.Errorf("ObjAddr got error %v, want any error=%v", err, tc.wantAddrError)
			}
			if err != nil {
				return
			}
			if got != tc.wantObjAddr {
				t.Errorf("got ObjAddr %x; want %x\n", got, tc.wantObjAddr)
			}
		})
	}
}
