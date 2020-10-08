// Copyright 2018 The conprof Authors
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
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/conprof/conprof/internal/pprof/plugin"
	"github.com/conprof/conprof/internal/pprof/report"
	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/route"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
)

var defaultMetadataTimeRange = 24 * time.Hour

type API struct {
	logger   log.Logger
	db       storage.Queryable
	reloadCh chan struct{}
}

func New(logger log.Logger, db storage.Queryable, reloadCh chan struct{}) *API {
	return &API{
		logger:   logger,
		db:       db,
		reloadCh: reloadCh,
	}
}

type Series struct {
	Labels          map[string]string `json:"labels"`
	LabelSetEncoded string            `json:"labelsetEncoded"`
	Timestamps      []int64           `json:"timestamps"`
}

func (a *API) QueryRange(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()

	fromString := r.URL.Query().Get("from")
	from, err := strconv.ParseInt(fromString, 10, 64)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	toString := r.URL.Query().Get("to")
	to, err := strconv.ParseInt(toString, 10, 64)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	if to < from {
		err := errors.New("to timestamp must not be before from time")
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, from, to)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := parser.ParseMetricSelector(queryString)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	set := q.Select(false, nil, sel...)
	res := []Series{}
	for set.Next() {
		series := set.At()
		ls := series.Labels()
		filteredLabels := labels.Labels{}
		m := make(map[string]string)
		for _, l := range ls {
			if l.Name != "" {
				filteredLabels = append(filteredLabels, l)
				m[l.Name] = l.Value
			}
		}

		resSeries := Series{Labels: m, LabelSetEncoded: base64.URLEncoding.EncodeToString([]byte(filteredLabels.String()))}
		i := series.Iterator()
		for i.Next() {
			t, _ := i.At()
			resSeries.Timestamps = append(resSeries.Timestamps, t)
		}

		if err := i.Err(); err != nil {
			level.Error(a.logger).Log("err", err, "series", ls.String())
		}

		res = append(res, resSeries)
	}
	if err := set.Err(); err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: set.Err()}
	}

	return res, set.Warnings(), nil
}

func (a *API) findProfile(ctx context.Context, time int64, sel []*labels.Matcher) (*profile.Profile, error) {
	q, err := a.db.Querier(ctx, time, time)
	if err != nil {
		return nil, err
	}

	set := q.Select(false, nil, sel...)
	for set.Next() {
		series := set.At()
		i := series.Iterator()
		for i.Next() {
			t, b := i.At()
			if t == time {
				return profile.ParseData(b)
			}
		}
		err = i.Err()
		if err != nil {
			return nil, err
		}
	}

	return nil, set.Err()
}

func (a *API) Query(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()

	timeString := r.URL.Query().Get("time")
	time, err := strconv.ParseInt(timeString, 10, 64)
	if err != nil {
		err = fmt.Errorf("unable to parse time: %w", err)
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	queryString := r.URL.Query().Get("query")

	level.Debug(a.logger).Log("query", queryString, "time", time)
	sel, err := parser.ParseMetricSelector(queryString)
	if err != nil {
		err = fmt.Errorf("unable to parse query: %w", err)
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	profile, err := a.findProfile(ctx, time, sel)
	// TODO(bwplotka): Handle warnings.
	if err != nil {
		err = fmt.Errorf("unable to find profile: %w", err)
		return nil, nil, &ApiError{Typ: ErrorInternal, Err: err}
	}

	switch r.URL.Query().Get("report") {
	case "top":
		rep, err := generateTopReport(profile)
		if err != nil {
			return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
		}
		return rep, nil, nil
	case "svg":
		return &svgRenderer{profile: profile}, nil, nil
	default:
		return &svgRenderer{profile: profile}, nil, nil
	}
}

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

// PrometheusResult allows compatibility with official Prometheus format https://prometheus.io/docs/prometheus/latest/querying/api/#format-overview.
type PrometheusResult struct {
	Err error
}

func (r PrometheusResult) MarshalJSON() ([]byte, error) {
	s := struct {
		Status string `json:"status"`
		Error  string `json:"error,omitempty"`
	}{Status: "success"}

	if r.Err != nil {
		s.Status = "error"
		s.Error = r.Err.Error()
	}

	return json.Marshal(s)
}

type SeriesResult struct {
	PrometheusResult

	Series []labels.Labels `json:"data"`
}

type LabelNamesResult struct {
	PrometheusResult
}

func parseMetadataTimeRange(r *http.Request, defaultMetadataTimeRange time.Duration) (time.Time, time.Time, error) {
	// If start and end time not specified as query parameter, we get the range from the beginning of time by default.
	var defaultStartTime, defaultEndTime time.Time
	if defaultMetadataTimeRange == 0 {
		defaultStartTime = timestamp.Time(math.MinInt64)
		defaultEndTime = timestamp.Time(math.MaxInt64)
	} else {
		now := time.Now()
		defaultStartTime = now.Add(-defaultMetadataTimeRange)
		defaultEndTime = now
	}

	start, err := parseTimeParam(r, "start", defaultStartTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseTimeParam(r, "end", defaultEndTime)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, errors.New("end timestamp must not be before start time")
	}
	return start, end, nil
}

func parseTimeParam(r *http.Request, paramName string, defaultValue time.Time) (time.Time, error) {
	val := r.FormValue(paramName)
	if val == "" {
		return defaultValue, nil
	}
	result, err := parseTime(val)
	if err != nil {
		return time.Time{}, errors.Wrapf(err, "Invalid time value for '%s'", paramName)
	}
	return result, nil
}

func parseTime(s string) (time.Time, error) {
	if t, err := strconv.ParseFloat(s, 64); err == nil {
		s, ns := math.Modf(t)
		ns = math.Round(ns*1000) / 1000
		return time.Unix(int64(s), int64(ns*float64(time.Second))), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Time{}, errors.Errorf("cannot parse %q to a valid timestamp", s)
}

func (a *API) Series(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		return nil, nil, &ApiError{Typ: ErrorInternal, Err: errors.Wrap(err, "parse form")}
	}

	if len(r.Form["match[]"]) == 0 {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: errors.New("no match[] parameter provided")}
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	var matcherSets [][]*labels.Matcher
	for _, s := range r.Form["match[]"] {
		matchers, err := parser.ParseMetricSelector(s)
		if err != nil {
			return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
		}
		matcherSets = append(matcherSets, matchers)
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	var (
		metrics = []labels.Labels{}
		sets    []storage.SeriesSet
	)
	for _, mset := range matcherSets {
		sets = append(sets, q.Select(false, nil, mset...))
	}

	set := storage.NewMergeSeriesSet(sets, storage.ChainedSeriesMerge)
	for set.Next() {
		metrics = append(metrics, set.At().Labels())
	}
	if set.Err() != nil {
		return nil, nil, &ApiError{Typ: ErrorInternal, Err: err}
	}

	return metrics, nil, nil
}

func (a *API) LabelNames(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	names, warnings, err := q.LabelNames()
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	return names, warnings, nil
}

func (a *API) LabelValues(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()
	name := route.Param(ctx, "name")

	if !model.LabelNameRE.MatchString(name) {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: errors.Errorf("invalid label name: %q", name)}
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	names, warnings, err := q.LabelValues(name)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	return names, warnings, nil
}

func (a *API) Reload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.reloadCh <- struct{}{}
}
