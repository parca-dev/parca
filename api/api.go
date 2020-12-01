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
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/conprof/conprof/internal/pprof/measurement"
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
	Labels     map[string]string `json:"labels"`
	Timestamps []int64           `json:"timestamps"`
}

func (a *API) QueryRange(r *http.Request) (interface{}, []error, *ApiError) {
	ctx := r.Context()

	fromString := r.URL.Query().Get("from")
	from, err := strconv.ParseInt(fromString, 10, 64)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: fmt.Errorf("failed to parse \"from\" time: %w", err)}
	}

	toString := r.URL.Query().Get("to")
	to, err := strconv.ParseInt(toString, 10, 64)
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: fmt.Errorf("failed to parse \"to\" time: %w", err)}
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
		return nil, nil, &ApiError{Typ: ErrorBadData, Err: err}
	}

	set := q.Select(true, nil, sel...)
	res := []Series{}
	for set.Next() {
		series := set.At()
		ls := series.Labels()

		resSeries := Series{Labels: ls.Map()}
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
		return nil, nil, &ApiError{Typ: ErrorInternal, Err: set.Err()}
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

func (a *API) SingleProfileQuery(r *http.Request) (*profile.Profile, *ApiError) {
	ctx := r.Context()

	return a.profileByParameters(
		ctx,
		"single",
		r.URL.Query().Get("time"),
		r.URL.Query().Get("query"),
		"",
		"",
	)
}

func (a *API) profileByParameters(ctx context.Context, mode, time, query, from, to string) (*profile.Profile, *ApiError) {
	switch mode {
	case "merge":
		f, err := strconv.ParseInt(from, 10, 64)
		if err != nil {
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		t, err := strconv.ParseInt(to, 10, 64)
		if err != nil {
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		if t < f {
			err := errors.New("to timestamp must not be before from time")
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		sel, err := parser.ParseMetricSelector(query)
		if err != nil {
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		return a.mergeProfiles(ctx, f, t, sel)
	case "single":
		t, err := strconv.ParseInt(time, 10, 64)
		if err != nil {
			err = fmt.Errorf("unable to parse time: %w", err)
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		sel, err := parser.ParseMetricSelector(query)
		if err != nil {
			err = fmt.Errorf("unable to parse query: %w", err)
			return nil, &ApiError{Typ: ErrorBadData, Err: err}
		}

		profile, err := a.findProfile(ctx, t, sel)
		// TODO(bwplotka): Handle warnings.
		if err != nil {
			err = fmt.Errorf("unable to find profile: %w", err)
			return nil, &ApiError{Typ: ErrorInternal, Err: err}
		}
		if profile == nil {
			return nil, &ApiError{Typ: ErrorNotFound, Err: errors.New("profile not found")}
		}

		return profile, nil
	default:
		return nil, &ApiError{Typ: ErrorBadData, Err: errors.New("no mode specified")}
	}
}

func (a *API) DiffProfiles(r *http.Request) (*profile.Profile, *ApiError) {
	ctx := r.Context()

	profileA, apiErr := a.profileByParameters(ctx,
		r.URL.Query().Get("mode_a"),
		r.URL.Query().Get("time_a"),
		r.URL.Query().Get("query_a"),
		r.URL.Query().Get("from_a"),
		r.URL.Query().Get("to_a"),
	)
	if apiErr != nil {
		return nil, apiErr
	}

	profileB, apiErr := a.profileByParameters(ctx,
		r.URL.Query().Get("mode_b"),
		r.URL.Query().Get("time_b"),
		r.URL.Query().Get("query_b"),
		r.URL.Query().Get("from_b"),
		r.URL.Query().Get("to_b"),
	)
	if apiErr != nil {
		return nil, apiErr
	}

	// compare totals of profiles, skip this to subtract profiles from each other
	profileA.SetLabel("pprof::base", []string{"true"})

	profileA.Scale(-1)

	profiles := []*profile.Profile{profileA, profileB}

	// Merge profiles.
	if err := measurement.ScaleProfiles(profiles); err != nil {
		return nil, &ApiError{Typ: ErrorInternal, Err: err}
	}

	p, err := profile.Merge(profiles)
	if err != nil {
		return nil, &ApiError{Typ: ErrorInternal, Err: err}
	}

	return p, nil
}

func (a *API) mergeProfiles(ctx context.Context, from, to int64, sel []*labels.Matcher) (*profile.Profile, *ApiError) {
	q, err := a.db.Querier(ctx, from, to)
	if err != nil {
		return nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	set := q.Select(false, nil, sel...)
	profiles := []*profile.Profile{}
	for set.Next() {
		series := set.At()
		i := series.Iterator()
		for i.Next() {
			_, b := i.At()
			p, err := profile.ParseData(b)
			if err != nil {
				level.Error(a.logger).Log("err", err)
			}
			profiles = append(profiles, p)
		}

		if err := i.Err(); err != nil {
			level.Error(a.logger).Log("err", err)
		}
	}
	if err := set.Err(); err != nil {
		return nil, &ApiError{Typ: ErrorInternal, Err: set.Err()}
	}

	// TODO(brancz): This will eventually need to be batched to limit memory use per request.
	p, err := profile.Merge(profiles)
	if err != nil {
		return nil, &ApiError{Typ: ErrorInternal, Err: err}
	}

	return p, nil
}

func (a *API) MergeProfiles(r *http.Request) (*profile.Profile, *ApiError) {
	ctx := r.Context()

	return a.profileByParameters(
		ctx,
		"merge",
		"",
		r.URL.Query().Get("query"),
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
	)
}

func (a *API) Query(r *http.Request) (interface{}, []error, *ApiError) {
	var (
		profile *profile.Profile
		apiErr  *ApiError
	)
	switch r.URL.Query().Get("mode") {
	case "diff":
		profile, apiErr = a.DiffProfiles(r)
		if apiErr != nil {
			return nil, nil, apiErr
		}
	case "merge":
		profile, apiErr = a.MergeProfiles(r)
		if apiErr != nil {
			return nil, nil, apiErr
		}
	case "single":
		profile, apiErr = a.SingleProfileQuery(r)
		if apiErr != nil {
			return nil, nil, apiErr
		}
	default:
		profile, apiErr = a.SingleProfileQuery(r)
		if apiErr != nil {
			return nil, nil, apiErr
		}
	}

	switch r.URL.Query().Get("report") {
	case "meta":
		meta, err := generateMetaReport(profile)
		if err != nil {
			return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
		}
		return meta, nil, nil
	case "top":
		top, err := generateTopReport(profile, r.URL.Query().Get("sample_index"))
		if err != nil {
			return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
		}
		return top, nil, nil
	case "flamegraph":
		fg, err := generateFlamegraphReport(profile, r.URL.Query().Get("sample_index"))
		if err != nil {
			return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
		}
		return fg, nil, nil
	case "proto":
		return &protoRenderer{profile: profile}, nil, nil
	case "svg":
		return &svgRenderer{
			logger:      a.logger,
			profile:     profile,
			sampleIndex: r.URL.Query().Get("sample_index"),
		}, nil, nil
	default:
		return &svgRenderer{
			logger:      a.logger,
			profile:     profile,
			sampleIndex: r.URL.Query().Get("sample_index"),
		}, nil, nil
	}
}

type valueType struct {
	Type string `json:"type,omitempty"`
}

type metaReport struct {
	SampleTypes       []valueType `json:"sampleTypes"`
	DefaultSampleType string      `json:"defaultSampleType"`
}

func generateMetaReport(profile *profile.Profile) (*metaReport, error) {
	index, err := profile.SampleIndexByName("")
	if err != nil {
		return nil, err
	}

	res := &metaReport{
		SampleTypes:       []valueType{},
		DefaultSampleType: profile.SampleType[index].Type,
	}
	for _, t := range profile.SampleType {
		res.SampleTypes = append(res.SampleTypes, valueType{t.Type})
	}

	return res, nil
}

type protoRenderer struct {
	profile *profile.Profile
}

func (r *protoRenderer) Render(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/vnd.google.protobuf+gzip")
	w.Header().Set("Content-Disposition", "attachment;filename=profile.pb.gz")
	err := r.profile.Write(w)
	if err != nil {
		chooseRenderer(nil, nil, &ApiError{Typ: ErrorExec, Err: err}).Render(w)
		return
	}
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
