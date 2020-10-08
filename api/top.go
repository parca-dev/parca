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
	Items  []textItem `json:"items,omitempty"`
}

func generateTopReport(p *profile.Profile) (*topReport, error) {
	numLabelUnits, _ := p.NumLabelUnits()
	p.Aggregate(false, true, true, true, false)

	value, meanDiv, sample, err := sampleFormat(p, "", false)
	if err != nil {
		return nil, err
	}

	stype := sample.Type

	rep := report.NewDefault(p, report.Options{
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
