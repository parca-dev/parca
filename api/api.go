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
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
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

type QueryResult struct {
	Series []Series `json:"series"`
}

type Series struct {
	Labels          map[string]string `json:"labels"`
	LabelSetEncoded string            `json:"labelsetEncoded"`
	Timestamps      []int64           `json:"timestamps"`
}

func (a *API) QueryRange(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	fromString := r.URL.Query().Get("from")
	from, err := strconv.ParseInt(fromString, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request, unable to parse from %s", err.Error()), http.StatusBadRequest)
		return
	}

	toString := r.URL.Query().Get("to")
	to, err := strconv.ParseInt(toString, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Bad Request, unable to parse to %s", err.Error()), http.StatusBadRequest)
		return
	}

	q, err := a.db.Querier(ctx, from, to)
	if err != nil {
		level.Error(a.logger).Log("err", err)
		return
	}

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := parser.ParseMetricSelector(queryString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	set := q.Select(false, nil, sel...)
	res := &QueryResult{Series: []Series{}}
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

		res.Series = append(res.Series, resSeries)
	}
	// TODO(bwplotka): Handle warnings.
	if set.Err() != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(set.Err(), "exec"))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(res); err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
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

	Names []string `json:"data"`
}

type LabelValuesResult struct {
	PrometheusResult

	Values []string `json:"data"`
}

func (a *API) respondPromError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)

	if eerr := json.NewEncoder(w).Encode(PrometheusResult{Err: err}); eerr != nil {
		level.Error(a.logger).Log("msg", "error marshalling json while handling other error", "marshaling_err", eerr, "err", err)
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

func (a *API) Series(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "parse form"))
		return
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		a.respondPromError(w, http.StatusBadRequest, err)
		return
	}

	var matcherSets [][]*labels.Matcher
	for _, s := range r.Form["match[]"] {
		matchers, err := parser.ParseMetricSelector(s)
		if err != nil {
			a.respondPromError(w, http.StatusBadRequest, err)
			return
		}
		matcherSets = append(matcherSets, matchers)
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "new querier"))
		return
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
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "exec"))
		return
	}

	if err = json.NewEncoder(w).Encode(SeriesResult{Series: metrics}); err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func (a *API) LabelNames(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "parse form"))
		return
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		a.respondPromError(w, http.StatusBadRequest, err)
		return
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "new querier"))
		return
	}

	// TODO(bwplotka): Handle warnings.
	names, _, err := q.LabelNames()
	if err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "retrieve label names"))
		return
	}

	if err = json.NewEncoder(w).Encode(LabelNamesResult{Names: names}); err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func (a *API) LabelValues(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx := r.Context()

	name := ps.ByName("label_name")
	if !model.LabelNameRE.MatchString(name) {
		a.respondPromError(w, http.StatusBadRequest, errors.Errorf("invalid label name %q", name))
	}

	if err := r.ParseForm(); err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "parse form"))
		return
	}

	start, end, err := parseMetadataTimeRange(r, defaultMetadataTimeRange)
	if err != nil {
		a.respondPromError(w, http.StatusBadRequest, err)
		return
	}

	q, err := a.db.Querier(ctx, timestamp.FromTime(start), timestamp.FromTime(end))
	if err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "new querier"))
		return
	}

	// TODO(bwplotka): Handle warnings.
	names, _, err := q.LabelValues(name)
	if err != nil {
		a.respondPromError(w, http.StatusInternalServerError, errors.Wrap(err, "retrieve label names"))
		return
	}

	if err = json.NewEncoder(w).Encode(LabelValuesResult{Values: names}); err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func (a *API) Reload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.reloadCh <- struct{}{}
}
