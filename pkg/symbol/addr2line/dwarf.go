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
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"runtime/debug"

	"github.com/go-kit/log"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

// DwarfLiner is a symbolizer that uses DWARF debug info to symbolize addresses.
type DwarfLiner struct {
	logger log.Logger

	debugData *dwarf.Data
	dbgFile   elfutils.DebugInfoFile
	f         *elf.File
	filename  string
}

// DWARF creates a new DwarfLiner.
func DWARF(logger log.Logger, filename string, f *elf.File, demangler *demangle.Demangler) (*DwarfLiner, error) {
	debugData, err := f.DWARF()
	if err != nil {
		return nil, fmt.Errorf("failed to read DWARF data: %w", err)
	}

	dbgFile, err := elfutils.NewDebugInfoFile(debugData, demangler)
	if err != nil {
		return nil, err
	}

	return &DwarfLiner{
		logger:    log.With(logger, "liner", "dwarf"),
		dbgFile:   dbgFile,
		debugData: debugData,
		f:         f,
		filename:  filename,
	}, nil
}

func (dl *DwarfLiner) Close() error {
	return dl.f.Close()
}

func (dl *DwarfLiner) File() string {
	return dl.filename
}

func (dl *DwarfLiner) PCRange() ([2]uint64, error) {
	r := dl.debugData.Reader()

	minSet := false
	var min, max uint64
	for {
		e, err := r.Next()
		if err != nil {
			return [2]uint64{}, fmt.Errorf("read DWARF entry: %w", err)
		}
		if e == nil {
			break
		}

		ranges, err := dl.debugData.Ranges(e)
		if err != nil {
			return [2]uint64{}, err
		}
		for _, pcs := range ranges {
			if !minSet {
				min = pcs[0]
				minSet = true
			}
			if pcs[1] > max {
				max = pcs[1]
			}
			if pcs[0] < min {
				min = pcs[0]
			}
		}
	}

	return [2]uint64{min, max}, nil
}

// PCToLines returns the resolved source lines for a program counter (memory address).
func (dl *DwarfLiner) PCToLines(addr uint64) (lines []profile.LocationLine, err error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered stack trace:\n", string(debug.Stack()))
			err = fmt.Errorf("recovering from panic in DWARF add2line: %v", r)
		}
	}()

	lines, err = dl.dbgFile.SourceLines(addr)
	if err != nil {
		return nil, err
	}
	return lines, nil
}
