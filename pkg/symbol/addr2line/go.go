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

package addr2line

import (
	"debug/elf"
	"debug/gosym"
	"errors"
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
		return nil, fmt.Errorf("failed to create go symbtab: %w", err)
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
func (gl *GoLiner) PCToLines(addr uint64) (lines []profile.LocationLine, err error) {
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
func gosymtab(objFile *elf.File) (*gosym.Table, error) {
	// The .gopclntab section contains tables and meta data required for symbolization,
	// see https://github.com/DataDog/go-profiler-notes/blob/main/stack-traces.md#gopclntab.
	var err error
	var pclntab []byte
	if sec := objFile.Section(".gopclntab"); sec != nil {
		if sec.Type == elf.SHT_NOBITS {
			return nil, errors.New(".gopclntab section has no bits")
		}

		pclntab, err = sec.Data()
		if err != nil {
			return nil, fmt.Errorf("could not find .gopclntab section: %w", err)
		}
	}

	if len(pclntab) <= 0 {
		return nil, errors.New(".gopclntab section has no bits")
	}

	var symtab []byte
	if sec := objFile.Section(".gosymtab"); sec != nil {
		symtab, _ = sec.Data()
	}

	var text uint64
	if sec := objFile.Section(".text"); sec != nil {
		text = sec.Addr
	}

	table, err := gosym.NewTable(symtab, gosym.NewLineTable(pclntab, text))
	if err != nil {
		return nil, fmt.Errorf("failed to build symtab or pclinetab: %w", err)
	}
	return table, nil
}
