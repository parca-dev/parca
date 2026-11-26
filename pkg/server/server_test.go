// Copyright 2026 The Parca Authors
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

package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestUIHandlerRendersDisableAutoQuery(t *testing.T) {
	uiFS := fstest.MapFS{
		"index.html": &fstest.MapFile{
			Data: []byte("window.DISABLE_AUTO_QUERY = '{{.DisableAutoQuery}}';"),
		},
	}

	for _, tc := range []struct {
		disableAutoQuery bool
		want             string
	}{
		{disableAutoQuery: false, want: "window.DISABLE_AUTO_QUERY = 'false';"},
		{disableAutoQuery: true, want: "window.DISABLE_AUTO_QUERY = 'true';"},
	} {
		s := NewServer(prometheus.NewRegistry(), "test")
		handler, err := s.uiHandler(uiFS, "", tc.disableAutoQuery)
		require.NoError(t, err)

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		require.Equal(t, http.StatusOK, rec.Code)

		body, err := io.ReadAll(rec.Body)
		require.NoError(t, err)
		require.Equal(t, tc.want, string(body))
	}
}
