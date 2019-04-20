package api

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"net/http"

	"github.com/Go-SIP/conprof/storage/tsdb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/tsdb/labels"
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

type QueryResult struct {
	Series []Series `json:"series"`
}

type Series struct {
	LabelSet        string  `json:"labelset"`
	LabelSetEncoded string  `json:"labelsetEncoded"`
	Timestamps      []int64 `json:"timestamps"`
}

func (a *API) QueryRange(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	q, err := a.db.Querier(math.MinInt64, math.MaxInt64)
	if err != nil {
		level.Error(a.logger).Log("err", err)
	}

	seriesSet, err := q.Select(labels.NewMustRegexpMatcher("job", ".+"))
	if err != nil {
		level.Error(a.logger).Log("err", err)
	}

	res := &QueryResult{}
	for seriesSet.Next() {
		series := seriesSet.At()
		ls := series.Labels()
		filteredLabels := labels.Labels{}
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
