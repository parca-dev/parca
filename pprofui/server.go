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
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/alecthomas/template"
	"github.com/google/pprof/driver"
	"github.com/google/pprof/profile"
	"github.com/pkg/errors"
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
	storage Storage
}

// NewServer creates a new Server backed by the supplied Storage.
func NewServer(storage Storage) *Server {
	s := &Server{
		storage: storage,
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
			log.Fatal(err)
		}

		series, err := s.storage.List()
		if err != nil {
			log.Fatal(err)
		}

		escapedSeriesNames := make(map[string]string, len(series))
		for k, _ := range series {
			escapedSeriesNames[k] = base64.URLEncoding.EncodeToString([]byte(k))
		}

		data := struct {
			Series             map[string][]string
			EscapedSeriesNames map[string]string
		}{
			Series:             series,
			EscapedSeriesNames: escapedSeriesNames,
		}

		err = t.Execute(w, data)
		if err != nil {
			log.Print(err)
		}
		return
	}

	series, timestamp, remainingPath := s.parsePath(r.URL.Path)
	fmt.Println("path:       ", r.URL.Path)
	fmt.Println("series:    ", series)
	fmt.Println("timestamp: ", timestamp)
	decodedSeriesName, err := base64.URLEncoding.DecodeString(series)
	if err != nil {
		msg := fmt.Sprintf("could not decode series name", err)
		http.Error(w, msg, http.StatusNotFound)
		return
	}
	series = string(decodedSeriesName)
	// Catch nonexistent IDs early or pprof will do a worse job at
	// giving an informative error.
	if err := s.storage.Get(series, timestamp, func(io.Reader) error { return nil }); err != nil {
		msg := fmt.Sprintf("profile for series %s at timestamp %s not found: %s", series, timestamp, err)
		http.Error(w, msg, http.StatusNotFound)
		return
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
		if err := s.storage.Get(series, timestamp, func(reader io.Reader) error {
			var err error
			p, err = profile.Parse(reader)
			return err
		}); err != nil {
			return nil, "", err
		}
		return p, "", nil
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
