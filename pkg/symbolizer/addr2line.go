// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package symbolizer

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

const (
	defaultAddr2line = "addr2line"

	// addr2line may produce multiple lines of output. We
	// use this sentinel to identify the end of the output.
	sentinel = ^uint64(0)
)

// addr2Liner is a connection to an addr2line command for obtaining
// address and line number information from a binary.
// copy from github.com/google/pprof@v0.0.0-20221118152302-e6195bd50e26/internal/binutils/addr2liner.go:39
type addr2Liner struct {
	mu   sync.Mutex
	rw   lineReaderWriter
	base uint64
}

// lineReaderWriter is an interface to abstract the I/O to an addr2line
// process. It writes a line of input to the job, and reads its output
// one line at a time.
type lineReaderWriter interface {
	write(string) error
	readLine() (string, error)
	close()
}

type addr2LinerJob struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
}

func (a *addr2LinerJob) write(s string) error {
	_, err := fmt.Fprint(a.in, s+"\n")
	return err
}

func (a *addr2LinerJob) readLine() (string, error) {
	s, err := a.out.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

// close releases any resources used by the addr2liner object.
func (a *addr2LinerJob) close() {
	a.in.Close()
	a.cmd.Wait()
}

// newAddr2liner starts the given addr2liner command reporting
// information about the given executable file. If file is a shared
// library, base should be the address at which it was mapped in the
// program under consideration.
func newAddr2Liner(cmd, file string, base uint64) (*addr2Liner, error) {
	if cmd == "" {
		cmd = defaultAddr2line
	}

	j := &addr2LinerJob{
		cmd: exec.Command(cmd, "-aif", "-e", file),
	}

	var err error
	if j.in, err = j.cmd.StdinPipe(); err != nil {
		return nil, err
	}

	outPipe, err := j.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	j.out = bufio.NewReader(outPipe)
	if err := j.cmd.Start(); err != nil {
		return nil, err
	}

	a := &addr2Liner{
		rw:   j,
		base: base,
	}

	return a, nil
}

// readFrame parses the addr2line output for a single address. It
// returns a populated plugin.Frame and whether it has reached the end of the
// data.
func (d *addr2Liner) readFrame() (Frame, bool) {
	funcname, err := d.rw.readLine()
	if err != nil {
		return Frame{}, true
	}
	if strings.HasPrefix(funcname, "0x") {
		// If addr2line returns a hex address we can assume it is the
		// sentinel. Read and ignore next two lines of output from
		// addr2line
		d.rw.readLine()
		d.rw.readLine()
		return Frame{}, true
	}

	fileline, err := d.rw.readLine()
	if err != nil {
		return Frame{}, true
	}

	linenumber := 0

	if funcname == "??" {
		funcname = ""
	}

	if fileline == "??:0" {
		fileline = ""
	} else {
		if i := strings.LastIndex(fileline, ":"); i >= 0 {
			// Remove discriminator, if present
			if disc := strings.Index(fileline, " (discriminator"); disc > 0 {
				fileline = fileline[:disc]
			}
			// If we cannot parse a number after the last ":", keep it as
			// part of the filename.
			if line, err := strconv.Atoi(fileline[i+1:]); err == nil {
				linenumber = line
				fileline = fileline[:i]
			}
		}
	}

	return Frame{
		Func: funcname,
		File: fileline,
		Line: linenumber}, false
}

func (d *addr2Liner) rawAddrInfo(addr uint64) ([]Frame, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := d.rw.write(fmt.Sprintf("%x", addr-d.base)); err != nil {
		return nil, err
	}

	if err := d.rw.write(fmt.Sprintf("%x", sentinel)); err != nil {
		return nil, err
	}

	resp, err := d.rw.readLine()
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(resp, "0x") {
		return nil, fmt.Errorf("unexpected addr2line output: %s", resp)
	}

	var stack []Frame
	for {
		frame, end := d.readFrame()
		if end {
			break
		}

		if frame != (Frame{}) {
			stack = append(stack, frame)
		}
	}
	return stack, err
}

// addrInfo returns the stack frame information for a specific program
// address. It returns nil if the address could not be identified.
func (d *addr2Liner) addrInfo(addr uint64) ([]Frame, error) {
	stack, err := d.rawAddrInfo(addr)
	if err != nil {
		return nil, err
	}

	return stack, nil
}

// A Frame describes a single line in a source file.
// this struct is in internal package, so we copy it.
// copy from github.com/google/pprof@v0.0.0-20221118152302-e6195bd50e26/internal/plugin/plugin.go:161.
type Frame struct {
	Func string // name of function
	File string // source file name
	Line int    // line in file
}
