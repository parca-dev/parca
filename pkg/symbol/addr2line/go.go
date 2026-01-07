// Copyright 2022-2026 The Parca Authors
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

package addr2line

import (
	"context"
	"debug/elf"
	"debug/gosym"
	"fmt"
	"runtime/debug"

	"github.com/go-kit/log"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

// GoLiner is a liner which utilizes .gopclntab section to symbolize addresses.
// It doesn't work for inlined functions.
type GoLiner struct {
	logger log.Logger

	symtab   *gosym.Table
	f        *elf.File
	filename string
}

// Go creates a new GoLiner.
func Go(logger log.Logger, filename string, f *elf.File) (*GoLiner, error) {
	tab, err := gosymtab(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create go symtab: %w", err)
	}

	return &GoLiner{
		logger:   log.With(logger, "liner", "go"),
		symtab:   tab,
		f:        f,
		filename: filename,
	}, nil
}

func (gl *GoLiner) Close() error {
	return gl.f.Close()
}

func (gl *GoLiner) File() string {
	return gl.filename
}

func (gl *GoLiner) PCRange() ([2]uint64, error) {
	minSet := false
	var min, max uint64

	for _, f := range gl.symtab.Funcs {
		if !minSet {
			min = f.Entry
			minSet = true
		}
		if f.End > max {
			max = f.End
		}
	}

	return [2]uint64{min, max}, nil
}

// PCToLines looks up the line number information for a program counter (memory address).
func (gl *GoLiner) PCToLines(ctx context.Context, addr uint64) (lines []profile.LocationLine, err error) {
	defer func() {
		// PCToLine panics with "invalid memory address or nil pointer dereference",
		//	- when it refers to an address that doesn't actually exist.
		if r := recover(); r != nil {
			fmt.Println("recovered stack stares:\n", string(debug.Stack()))
			err = fmt.Errorf("recovering from panic in Go add2line: %v", r)
		}
	}()

	name := "?"
	// TODO(kakkoyun): Do we need to consider the base address for any part of Go binaries?
	file, line, fn := gl.symtab.PCToLine(addr)
	if fn != nil {
		name = fn.Name
	}

	// TODO(kakkoyun): These lines miss the inline functions.
	// - Find a way to symbolize inline functions.
	lines = append(lines, profile.LocationLine{
		Line: int64(line),
		Function: &pb.Function{
			Name:     name,
			Filename: file,
		},
	})
	return lines, nil
}

// gosymtab returns the Go symbol table (.gosymtab section) decoded from the ELF file.
func gosymtab(f *elf.File) (*gosym.Table, error) {
	var (
		textStart = uint64(0)

		symtab  []byte
		pclntab []byte
		err     error
	)
	if sect := f.Section(".text"); sect != nil {
		textStart = sect.Addr
	}

	sectionName := ".gosymtab"
	sect := f.Section(".gosymtab")
	if sect == nil {
		// try .data.rel.ro.gosymtab, for PIE binaries
		sectionName = ".data.rel.ro.gosymtab"
		sect = f.Section(".data.rel.ro.gosymtab")
	}
	if sect != nil {
		if symtab, err = sect.Data(); err != nil {
			return nil, fmt.Errorf("read %s section: %w", sectionName, err)
		}
	} else {
		// if both sections failed, try the symbol
		symtab = symbolData(f, "runtime.symtab", "runtime.esymtab")
	}

	sectionName = ".gopclntab"
	sect = f.Section(".gopclntab")
	if sect == nil {
		// try .data.rel.ro.gopclntab, for PIE binaries
		sectionName = ".data.rel.ro.gopclntab"
		sect = f.Section(".data.rel.ro.gopclntab")
	}
	if sect != nil {
		if pclntab, err = sect.Data(); err != nil {
			return nil, fmt.Errorf("read %s section: %w", sectionName, err)
		}
	} else {
		// if both sections failed, try the symbol
		pclntab = symbolData(f, "runtime.pclntab", "runtime.epclntab")
	}

	runtimeTextAddr, ok := runtimeTextAddr(f)
	if ok {
		textStart = runtimeTextAddr
	}

	return gosym.NewTable(symtab, gosym.NewLineTable(pclntab, textStart))
}

func symbolData(f *elf.File, start, end string) []byte {
	elfSyms, err := f.Symbols()
	if err != nil {
		return nil
	}
	var addr, eaddr uint64
	for _, s := range elfSyms {
		switch s.Name {
		case start:
			addr = s.Value
		case end:
			eaddr = s.Value
		}
		if addr != 0 && eaddr != 0 {
			break
		}
	}
	if addr == 0 || eaddr < addr {
		return nil
	}
	size := eaddr - addr
	data := make([]byte, size)
	for _, prog := range f.Progs {
		if prog.Vaddr <= addr && addr+size-1 <= prog.Vaddr+prog.Filesz-1 {
			if _, err := prog.ReadAt(data, int64(addr-prog.Vaddr)); err != nil {
				return nil
			}
			return data
		}
	}
	return nil
}

func runtimeTextAddr(f *elf.File) (uint64, bool) {
	elfSyms, err := f.Symbols()
	if err != nil {
		return 0, false
	}

	for _, s := range elfSyms {
		if s.Name != "runtime.text" {
			continue
		}

		return s.Value, true
	}

	return 0, false
}
