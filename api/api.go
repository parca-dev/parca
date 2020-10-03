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
	"encoding/base64"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/route"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	thanosapi "github.com/thanos-io/thanos/pkg/api"
	extpromhttp "github.com/thanos-io/thanos/pkg/extprom/http"
	"github.com/thanos-io/thanos/pkg/server/http/middleware"

	"github.com/conprof/db/storage"
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

func (a *API) QueryRange(r *http.Request) (interface{}, []error, *thanosapi.ApiError) {
	ctx := r.Context()

	fromString := r.URL.Query().Get("from")
	from, err := strconv.ParseInt(fromString, 10, 64)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
	}

	toString := r.URL.Query().Get("to")
	to, err := strconv.ParseInt(toString, 10, 64)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, from, to)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
	}

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := parser.ParseMetricSelector(queryString)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
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
		err = i.Err()
		if err != nil {
			level.Error(a.logger).Log("err", err, "series", ls.String())
		}

		res = append(res, resSeries)
	}
	if set.Err() != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: set.Err()}
	}

	return res, set.Warnings(), nil
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

func (a *API) Series(r *http.Request) (interface{}, []error, *thanosapi.ApiError) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorInternal, Err: errors.Wrap(err, "parse form")}
	}

	if len(r.Form["match[]"]) == 0 {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: errors.New("no match[] parameter provided")}
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
	}

	var matcherSets [][]*labels.Matcher
	for _, s := range r.Form["match[]"] {
		matchers, err := parser.ParseMetricSelector(s)
		if err != nil {
			return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
		}
		matcherSets = append(matcherSets, matchers)
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
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
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorInternal, Err: err}
	}

	return metrics, nil, nil
}

func (a *API) LabelNames(r *http.Request) (interface{}, []error, *thanosapi.ApiError) {
	ctx := r.Context()

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
	}

	names, warnings, err := q.LabelNames()
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
	}

	return names, warnings, nil
}

func (a *API) LabelValues(r *http.Request) (interface{}, []error, *thanosapi.ApiError) {
	ctx := r.Context()
	name := route.Param(ctx, "name")

	if !model.LabelNameRE.MatchString(name) {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: errors.Errorf("invalid label name: %q", name)}
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorBadData, Err: err}
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
	}

	names, warnings, err := q.LabelValues(name)
	if err != nil {
		return nil, nil, &thanosapi.ApiError{Typ: thanosapi.ErrorExec, Err: err}
	}

	return names, warnings, nil
}

func (a *API) Reload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.reloadCh <- struct{}{}
}

// TODO: add tracer
// Instr returns a http HandlerFunc with the instrumentation middleware.
func GetInstr(
	_ log.Logger,
	ins extpromhttp.InstrumentationMiddleware,
) func(name string, f thanosapi.ApiFunc) httprouter.Handle {
	instr := func(name string, f thanosapi.ApiFunc) httprouter.Handle {
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			thanosapi.SetCORS(w)
			if data, warnings, err := f(r); err != nil {
				thanosapi.RespondError(w, err, data)
			} else if data != nil {
				thanosapi.Respond(w, data, warnings)
			} else {
				w.WriteHeader(http.StatusNoContent)
			}
		})
		return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
			ins.NewHandler(name, gziphandler.GzipHandler(middleware.RequestID(hf))).ServeHTTP(w, r)
		}
	}
	return instr
}
