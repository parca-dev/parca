// Copyright 2018 The conprof Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package pprofui

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Go-SIP/conprof/storage/tsdb"
	"github.com/alecthomas/template"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/driver"
	"github.com/google/pprof/profile"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/tsdb/labels"
	"github.com/spf13/pflag"
)

const tpl = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>List Profiles</title>
</head>
<body>
{{range $series, $element := .Series}}
<h4>{{ $series }}</h4>
{{range $element }}
<div><a href="/{{ with (index $.EscapedSeriesNames $series) }}{{ . }}{{ end }}/{{ . }}/">{{ . }}</a></div>{{else}}<div><strong>no rows</strong></div>
{{end}}
{{end}}
</body>
</html>`

// A Server serves up the pprof web ui. A request to /<profiletype>
// generates a profile of the desired type and redirects to the UI for
// it at /<profiletype>/<id>. Valid profile types at the time of
// writing include `profile` (cpu), `goroutine`, `threadcreate`,
// `heap`, `block`, and `mutex`.
type Server struct {
	logger log.Logger
	db     *tsdb.DB
}

// NewServer creates a new Server backed by the supplied Storage.
func NewServer(logger log.Logger, db *tsdb.DB) *Server {
	s := &Server{
		logger: logger,
		db:     db,
	}

	return s
}

func (s *Server) parsePath(reqPath string) (series string, timestamp string, remainingPath string) {
	parts := strings.Split(path.Clean(strings.TrimPrefix(reqPath, "/")), "/")
	if len(parts) < 2 {
		return "", "", ""
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "" || r.URL.Path == "/" {
		t, err := template.New("webpage").Parse(tpl)
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		q, err := s.db.Querier(math.MinInt64, math.MaxInt64)
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		seriesSet, err := q.Select(labels.NewMustRegexpMatcher("job", ".+"))
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		seriesMap := map[string][]string{}
		for seriesSet.Next() {
			series := seriesSet.At()
			i := series.Iterator()
			ls := series.Labels()
			sampleTimestamps := []string{}
			for i.Next() {
				t, _ := i.At()
				sampleTimestamps = append(sampleTimestamps, intToString(t))
			}
			err = i.Err()
			if err != nil {
				level.Error(s.logger).Log("err", err, "series", ls.String())
			}
			filteredLabels := labels.Labels{}
			for _, l := range ls {
				if l.Name != "" {
					filteredLabels = append(filteredLabels, l)
				}
			}
			seriesMap[filteredLabels.String()] = sampleTimestamps
		}
		err = seriesSet.Err()
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		escapedSeriesNames := make(map[string]string, len(seriesMap))
		for k := range seriesMap {
			escapedSeriesNames[k] = base64.URLEncoding.EncodeToString([]byte(k))
		}

		data := struct {
			Series             map[string][]string
			EscapedSeriesNames map[string]string
		}{
			Series:             seriesMap,
			EscapedSeriesNames: escapedSeriesNames,
		}

		err = t.Execute(w, data)
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}
		return
	}

	series, timestamp, remainingPath := s.parsePath(r.URL.Path)
	decodedSeriesName, err := base64.URLEncoding.DecodeString(series)
	if err != nil {
		msg := fmt.Sprintf("could not decode series name: %s", err)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	seriesLabelsString := string(decodedSeriesName)
	seriesLabels, err := promql.ParseMetricSelector(seriesLabelsString)
	if err != nil {
		msg := fmt.Sprintf("failed to parse series labels %v with error %v", seriesLabelsString, err)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	m := make(labels.Selector, len(seriesLabels))
	for i, l := range seriesLabels {
		m[i] = labels.NewEqualMatcher(l.Name, l.Value)
	}

	server := func(args *driver.HTTPServerArgs) error {
		handler, ok := args.Handlers["/"+remainingPath]
		if !ok {
			return errors.Errorf("unknown endpoint %s", remainingPath)
		}
		handler.ServeHTTP(w, r)
		return nil
	}

	storageFetcher := func(_ string, _, _ time.Duration) (*profile.Profile, string, error) {
		var p *profile.Profile

		q, err := s.db.Querier(0, math.MaxInt64)
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		ss, err := q.Select(m...)
		if err != nil {
			level.Error(s.logger).Log("err", err)
		}

		ss.Next()
		s := ss.At()
		i := s.Iterator()
		t, err := stringToInt(timestamp)
		if err != nil {
			return nil, "", err
		}
		i.Seek(t)
		_, buf := i.At()
		p, err = profile.Parse(bytes.NewReader(buf))
		return p, "", err
	}

	// Invoke the (library version) of `pprof` with a number of stubs.
	// Specifically, we pass a fake FlagSet that plumbs through the
	// given args, a UI that logs any errors pprof may emit, a fetcher
	// that simply reads the profile we downloaded earlier, and a
	// HTTPServer that pprof will pass the web ui handlers to at the
	// end (and we let it handle this client request).
	if err := driver.PProf(&driver.Options{
		Flagset: &pprofFlags{
			FlagSet: pflag.NewFlagSet("pprof", pflag.ExitOnError),
			args: []string{
				"--symbolize", "none",
				"--http", "localhost:0",
				"", // we inject our own target
			},
		},
		UI:         &fakeUI{},
		Fetch:      fetcherFn(storageFetcher),
		HTTPServer: server,
	}); err != nil {
		_, _ = w.Write([]byte(err.Error()))
	}

	return
}

type fetcherFn func(_ string, _, _ time.Duration) (*profile.Profile, string, error)

func (f fetcherFn) Fetch(s string, d, t time.Duration) (*profile.Profile, string, error) {
	return f(s, d, t)
}

func intToString(i int64) string {
	return strconv.FormatInt(i, 10)
}
func stringToInt(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	return int64(i), err
}
