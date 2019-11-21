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
	"math"
	"net/http"
	"strconv"

	"github.com/conprof/tsdb"
	tsdbLabels "github.com/conprof/tsdb/labels"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql"
)

type API struct {
	logger log.Logger
	db     *tsdb.DB
}

func New(logger log.Logger, db *tsdb.DB) *API {
	return &API{
		logger: logger,
		db:     db,
	}
}

type LabelNamesResult struct {
	LabelNames []string `json:"data"`
}

type QueryResult struct {
	Series []Series `json:"series"`
}

type Series struct {
	LabelSet        string  `json:"labelset"`
	LabelSetEncoded string  `json:"labelsetEncoded"`
	Timestamps      []int64 `json:"timestamps"`
}

func (a *API) LabelNames(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	q, err := a.db.Querier(math.MinInt64, math.MaxInt64)
	if err != nil {
		http.Error(w, "Querier Error", http.StatusInternalServerError)
	}
	defer q.Close()
	names, err := q.LabelNames()
	if err != nil {
		http.Error(w, "Querier Error", http.StatusInternalServerError)
	}

	res := &LabelNamesResult{LabelNames: names}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func (a *API) QueryRange(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fromString := r.URL.Query().Get("from")
	from, err := strconv.Atoi(fromString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	toString := r.URL.Query().Get("to")
	to, err := strconv.Atoi(toString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	q, err := a.db.Querier(int64(from), int64(to))
	if err != nil {
		level.Error(a.logger).Log("err", err)
	}

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := promql.ParseMetricSelector(queryString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	ms := make([]tsdbLabels.Matcher, 0, len(sel))

	for _, om := range sel {
		ms = append(ms, convertMatcher(om))
	}

	seriesSet, err := q.Select(ms...)
	if err != nil {
		level.Error(a.logger).Log("err", err)
	}

	res := &QueryResult{Series: []Series{}}
	for seriesSet.Next() {
		series := seriesSet.At()
		ls := series.Labels()
		filteredLabels := tsdbLabels.Labels{}
		for _, l := range ls {
			if l.Name != "" {
				filteredLabels = append(filteredLabels, l)
			}
		}

		resSeries := Series{LabelSet: filteredLabels.String(), LabelSetEncoded: base64.URLEncoding.EncodeToString([]byte(filteredLabels.String()))}
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

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func convertMatcher(m *labels.Matcher) tsdbLabels.Matcher {
	switch m.Type {
	case labels.MatchEqual:
		return tsdbLabels.NewEqualMatcher(m.Name, m.Value)

	case labels.MatchNotEqual:
		return tsdbLabels.Not(tsdbLabels.NewEqualMatcher(m.Name, m.Value))

	case labels.MatchRegexp:
		res, err := tsdbLabels.NewRegexpMatcher(m.Name, "^(?:"+m.Value+")$")
		if err != nil {
			panic(err)
		}
		return res

	case labels.MatchNotRegexp:
		res, err := tsdbLabels.NewRegexpMatcher(m.Name, "^(?:"+m.Value+")$")
		if err != nil {
			panic(err)
		}
		return tsdbLabels.Not(res)
	}
	panic("storage.convertMatcher: invalid matcher type")
}
