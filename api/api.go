package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/conprof/conprof/storage/tsdb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql"
	tsdbLabels "github.com/prometheus/tsdb/labels"
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
