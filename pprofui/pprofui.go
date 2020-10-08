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
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/driver"
	"github.com/google/pprof/profile"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/spf13/pflag"

	"github.com/conprof/db/storage"
)

type pprofUI struct {
	logger log.Logger
	db     storage.Queryable
}

// NewServer creates a new Server backed by the supplied Storage.
func New(logger log.Logger, db storage.Queryable) *pprofUI {
	s := &pprofUI{
		logger: logger,
		db:     db,
	}

	return s
}

func parsePath(reqPath string) (series string, timestamp string, remainingPath string) {
	parts := strings.Split(path.Clean(strings.TrimPrefix(reqPath, "/pprof/")), "/")
	if len(parts) < 2 {
		return "", "", ""
	}
	return parts[0], parts[1], strings.Join(parts[2:], "/")
}

func (p *pprofUI) selectProfile(m labels.Selector, timestamp int64) ([]byte, error) {
	q, err := p.db.Querier(context.TODO(), 0, math.MaxInt64)
	if err != nil {
		level.Error(p.logger).Log("err", err)
		return nil, err
	}

	ss := q.Select(false, nil, m...)
	ok := ss.Next()
	if !ok {
		return nil, errors.New("could not get series set")
	}
	s := ss.At()
	i := s.Iterator()
	ok = i.Seek(timestamp)
	if !ok {
		return nil, errors.New("could not get series set")
	}
	_, buf := i.At()

	return buf, nil
}

func (p *pprofUI) PprofView(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	series, timestamp, remainingPath := parsePath(r.URL.Path)
	if !strings.HasPrefix(remainingPath, "/") {
		remainingPath = "/" + remainingPath
	}
	level.Debug(p.logger).Log("msg", "parsed path", "series", series, "timestamp", timestamp, "remainingPath", remainingPath)
	decodedSeriesName, err := base64.URLEncoding.DecodeString(series)
	if err != nil {
		msg := fmt.Sprintf("could not decode series name: %s with error %v", series, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	seriesLabelsString := string(decodedSeriesName)
	m, err := parser.ParseMetricSelector(seriesLabelsString)
	if err != nil {
		msg := fmt.Sprintf("failed to parse series labels %v with error %v", seriesLabelsString, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	t, err := stringToInt(timestamp)
	if err != nil {
		msg := fmt.Sprintf("failed to parse timestamp %s with error %v", timestamp, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	server := func(args *driver.HTTPServerArgs) error {
		handler, ok := args.Handlers[remainingPath]
		if !ok {
			return errors.Errorf("unknown endpoint %s", remainingPath)
		}
		handler.ServeHTTP(w, r)
		return nil
	}

	storageFetcher := func(_ string, _, _ time.Duration) (*profile.Profile, string, error) {
		var prof *profile.Profile

		buf, err := p.selectProfile(m, t)
		if err != nil {
			return prof, "", err
		}
		prof, err = profile.Parse(bytes.NewReader(buf))
		return prof, "", err
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
}

func (p *pprofUI) PprofDownload(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	parts := strings.Split(path.Clean(strings.TrimPrefix(r.URL.Path, "/download/")), "/")
	if len(parts) < 2 {
		http.Error(w, "don't have enough parameters", http.StatusBadRequest)
		return
	}
	series, timestamp := parts[0], parts[1]
	level.Debug(p.logger).Log("msg", "parsed path", "series", series, "timestamp", timestamp)
	decodedSeriesName, err := base64.URLEncoding.DecodeString(series)
	if err != nil {
		msg := fmt.Sprintf("could not decode series name: %s", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	seriesLabelsString := string(decodedSeriesName)
	m, err := parser.ParseMetricSelector(seriesLabelsString)
	if err != nil {
		msg := fmt.Sprintf("failed to parse series labels %v with error %v", seriesLabelsString, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	t, err := stringToInt(timestamp)
	if err != nil {
		msg := fmt.Sprintf("failed to parse timestamp %s with error %v", timestamp, err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	buf, err := p.selectProfile(m, t)
	if err != nil {
		msg := fmt.Sprintf("failed to select profile with error %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=profile")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	w.Write(buf)
}

type fetcherFn func(_ string, _, _ time.Duration) (*profile.Profile, string, error)

func (f fetcherFn) Fetch(s string, d, t time.Duration) (*profile.Profile, string, error) {
	return f(s, d, t)
}

func stringToInt(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)
	return i, err
}
