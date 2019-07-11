package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/conprof/conprof/filestorage"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql"
)

type Storage interface {
	ListSeries(from, to time.Time, matchers ...*labels.Matcher) (map[string][]filestorage.FileMedatada, error)
}

type API struct {
	logger  log.Logger
	storage Storage
}

func New(logger log.Logger, storage Storage) *API {
	return &API{
		logger:  logger,
		storage: storage,
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

	queryString := r.URL.Query().Get("query")
	level.Debug(a.logger).Log("query", queryString, "from", from, "to", to)
	sel, err := promql.ParseMetricSelector(queryString)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	fromTime := timestamp.Time(int64(from))
	toTime := timestamp.Time(int64(to))

	seriesSet, err := a.storage.ListSeries(fromTime, toTime, sel...)
	if err != nil {
		level.Error(a.logger).Log("err", err)
	}

	res := &QueryResult{Series: []Series{}}
	for labelsString, files := range seriesSet {
		resSeries := Series{LabelSet: labelsString, LabelSetEncoded: base64.URLEncoding.EncodeToString([]byte(labelsString))}
		for _, file := range files {
			resSeries.Timestamps = append(resSeries.Timestamps, file.Time.Unix()*1000)
		}

		res.Series = append(res.Series, resSeries)
	}

	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		level.Error(a.logger).Log("msg", "error marshaling json", "err", err)
	}
}
