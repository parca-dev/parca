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
	"github.com/conprof/conprof/internal/pprof/report"
	"github.com/google/pprof/profile"
)

type textItem struct {
	Name        string `json:"name,omitempty"`
	InlineLabel string `json:"inlineLabel,omitempty"`
	Flat        int64  `json:"flat,omitempty"`
	Cum         int64  `json:"cum,omitempty"`
	FlatFormat  string `json:"flatFormat,omitempty"`
	CumFormat   string `json:"cumFormat,omitempty"`
}

type topReport struct {
	Labels []string   `json:"labels,omitempty"`
	Total  int64      `json:"total"`
	Items  []textItem `json:"items,omitempty"`
}

func generateTopReport(p *profile.Profile, sampleIndex string) (*topReport, error) {
	numLabelUnits, _ := p.NumLabelUnits()
	err := p.Aggregate(true, true, false, false, false)
	if err != nil {
		return nil, err
	}

	value, meanDiv, sample, err := sampleFormat(p, sampleIndex, false)
	if err != nil {
		return nil, err
	}

	stype := sample.Type

	rep := report.New(p, &report.Options{
		OutputFormat:  report.Text,
		OutputUnit:    "minimum",
		Ratio:         1,
		NumLabelUnits: numLabelUnits,

		SampleValue:       value,
		SampleMeanDivisor: meanDiv,
		SampleType:        stype,
		SampleUnit:        sample.Unit,

		NodeCount:    500,
		NodeFraction: 0.005,
		EdgeFraction: 0.001,
	})

	items, labels := report.TextItems(rep)
	res := &topReport{
		Labels: labels,
		Total:  rep.Total(),
		Items:  make([]textItem, 0, len(items)),
	}

	for _, i := range items {
		res.Items = append(res.Items, textItem{
			Name:        i.Name,
			InlineLabel: i.InlineLabel,
			Flat:        i.Flat,
			Cum:         i.Cum,
			FlatFormat:  i.FlatFormat,
			CumFormat:   i.CumFormat,
		})
	}

	return res, nil
}
