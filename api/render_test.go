// Copyright 2021 The conprof Authors
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
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestRenderFlamegraph(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	p, err := profile.ParseData(b)
	require.NoError(t, err)

	v := url.Values{}
	v.Set("report", "flamegraph")
	u := &url.URL{
		Scheme:   "http",
		Host:     "example.com",
		RawQuery: v.Encode(),
	}
	req := httptest.NewRequest("GET", u.String(), nil)

	r := NewProfileResponseRenderer(
		log.NewNopLogger(),
		p,
		nil,
		req,
	)

	w := httptest.NewRecorder()

	err = r.Render(w)
	require.NoError(t, err)

	res := w.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRenderSVG(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	p, err := profile.ParseData(b)
	require.NoError(t, err)

	v := url.Values{}
	v.Set("report", "svg")
	u := &url.URL{
		Scheme:   "http",
		Host:     "example.com",
		RawQuery: v.Encode(),
	}
	req := httptest.NewRequest("GET", u.String(), nil)

	r := NewProfileResponseRenderer(
		log.NewNopLogger(),
		p,
		nil,
		req,
	)

	w := httptest.NewRecorder()
	tryRender(t, r, w)

	res := w.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRenderMeta(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	p, err := profile.ParseData(b)
	require.NoError(t, err)

	v := url.Values{}
	v.Set("report", "meta")
	u := &url.URL{
		Scheme:   "http",
		Host:     "example.com",
		RawQuery: v.Encode(),
	}
	req := httptest.NewRequest("GET", u.String(), nil)

	r := NewProfileResponseRenderer(
		log.NewNopLogger(),
		p,
		nil,
		req,
	)

	w := httptest.NewRecorder()

	err = r.Render(w)
	require.NoError(t, err)

	res := w.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

func TestRenderTop(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	p, err := profile.ParseData(b)
	require.NoError(t, err)

	v := url.Values{}
	v.Set("report", "top")
	u := &url.URL{
		Scheme:   "http",
		Host:     "example.com",
		RawQuery: v.Encode(),
	}
	req := httptest.NewRequest("GET", u.String(), nil)

	r := NewProfileResponseRenderer(
		log.NewNopLogger(),
		p,
		nil,
		req,
	)

	w := httptest.NewRecorder()

	err = r.Render(w)
	require.NoError(t, err)

	res := w.Result()
	require.Equal(t, http.StatusOK, res.StatusCode)
}

// A renderer renders output to an http.ResponseWriter.
type renderer interface {
	Render(w http.ResponseWriter) error
}

// tryRender calls r.Render but skips a test if certain conditions are not met.
func tryRender(t *testing.T, r renderer, w http.ResponseWriter) {
	t.Helper()

	err := r.Render(w)
	if errors.Is(err, exec.ErrNotFound) {
		// SVG renderer requires a graphviz installation.
		t.Skipf("skipping, missing executable: %v", err)
	}

	require.NoError(t, err)
}
