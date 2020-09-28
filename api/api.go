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
	"net/http"
	"strconv"

	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

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
	}

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := parser.ParseMetricSelector(queryString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	seriesSet := q.Select(false, nil, sel...)
	res := &QueryResult{Series: []Series{}}
	for seriesSet.Next() {
		series := seriesSet.At()
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

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}

func (a *API) Reload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	a.reloadCh <- struct{}{}
}
