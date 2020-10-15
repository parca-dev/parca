// Copyright 2020 The conprof Authors
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

package api

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"

	"github.com/conprof/conprof/internal/pprof/plugin"
	"github.com/conprof/conprof/internal/pprof/report"
	"github.com/google/pprof/profile"
)

type svgRenderer struct {
	profile *profile.Profile
}

func (r *svgRenderer) Render(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/svg+xml")
	numLabelUnits, _ := r.profile.NumLabelUnits()
	r.profile.Aggregate(false, true, true, true, false)

	value, meanDiv, sample, err := sampleFormat(r.profile, "", false)
	if err != nil {
		chooseRenderer(nil, nil, &ApiError{Typ: ErrorExec, Err: err}).Render(w)
		return
	}

	stype := sample.Type

	rep := report.NewDefault(r.profile, report.Options{
		OutputFormat:  report.Dot,
		OutputUnit:    "minimum",
		Ratio:         1,
		NumLabelUnits: numLabelUnits,

		SampleValue:       value,
		SampleMeanDivisor: meanDiv,
		SampleType:        stype,
		SampleUnit:        sample.Unit,

		NodeCount:    80,
		NodeFraction: 0.005,
		EdgeFraction: 0.001,
	})

	input := bytes.NewBuffer(nil)
	if err := report.Generate(input, rep, &fakeObjTool{}); err != nil {
		chooseRenderer(nil, nil, &ApiError{Typ: ErrorExec, Err: err}).Render(w)
		return
	}

	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = input, w, os.Stderr
	if err := cmd.Run(); err != nil {
		chooseRenderer(nil, nil, &ApiError{Typ: ErrorExec, Err: err}).Render(w)
		return
	}
}

type sampleValueFunc func([]int64) int64

// sampleFormat returns a function to extract values out of a profile.Sample,
// and the type/units of those values.
func sampleFormat(p *profile.Profile, sampleIndex string, mean bool) (value, meanDiv sampleValueFunc, v *profile.ValueType, err error) {
	if len(p.SampleType) == 0 {
		return nil, nil, nil, fmt.Errorf("profile has no samples")
	}
	index, err := p.SampleIndexByName(sampleIndex)
	if err != nil {
		return nil, nil, nil, err
	}
	value = valueExtractor(index)
	if mean {
		meanDiv = valueExtractor(0)
	}
	v = p.SampleType[index]
	return
}

func valueExtractor(ix int) sampleValueFunc {
	return func(v []int64) int64 {
		return v[ix]
	}
}

type fakeObjTool struct {
}

func (t *fakeObjTool) Open(file string, start, limit, offset uint64) (plugin.ObjFile, error) {
	panic("Unimplemented")
	return nil, nil
}

func (t *fakeObjTool) Disasm(file string, start, end uint64, intelSyntax bool) ([]plugin.Inst, error) {
	panic("Unimplemented")
	return nil, nil
}
