// Copyright 2020 The conprof Authors
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
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/common/route"
	extpromhttp "github.com/thanos-io/thanos/pkg/extprom/http"
	"github.com/thanos-io/thanos/pkg/server/http/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Status string

const (
	StatusSuccess Status = "success"
	StatusError   Status = "error"
)

type ErrorType string

const (
	ErrorNone     ErrorType = ""
	ErrorTimeout  ErrorType = "timeout"
	ErrorCanceled ErrorType = "canceled"
	ErrorExec     ErrorType = "execution"
	ErrorBadData  ErrorType = "bad_data"
	ErrorInternal ErrorType = "internal"
	ErrorNotFound ErrorType = "not_found"
)

type ApiError struct {
	Typ ErrorType
	Err error
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("%s: %s", e.Typ, e.Err)
}

type Response struct {
	Status    Status      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorType ErrorType   `json:"errorType,omitempty"`
	Error     string      `json:"error,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
}

type HttpResponseRenderer interface {
	Render(w http.ResponseWriter) error
}

type ApiFunc func(r *http.Request) (interface{}, []error, *ApiError)

// TODO: add tracer
// Instr returns a http HandlerFunc with the instrumentation middleware.
func Instr(
	logger log.Logger,
	ins extpromhttp.InstrumentationMiddleware,
) func(name string, f ApiFunc) httprouter.Handle {
	instr := func(name string, f ApiFunc) httprouter.Handle {
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, warnings, apiErr := f(r)
			ren := chooseRenderer(data, warnings, apiErr)
			err := ren.Render(w)
			if err != nil {
				// Attempt to show the user the error.
				ren = chooseRenderer(nil, nil, &ApiError{Typ: ErrorInternal, Err: err})
				renErr := ren.Render(w)
				level.Error(logger).Log("msg", "failed to render error", "err", err, "render_error", renErr)
			}
		})
		return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			for _, p := range params {
				ctx = route.WithParam(ctx, p.Key, p.Value)
			}
			otelhttp.NewHandler(ins.NewHandler(name, gziphandler.GzipHandler(middleware.RequestID(hf))), name).ServeHTTP(w, r.WithContext(ctx))
		}
	}
	return instr
}

func chooseRenderer(data interface{}, warnings []error, err *ApiError) HttpResponseRenderer {
	if err != nil {
		return &ErrorResponse{Data: data, ApiErr: err}
	}
	if data != nil {
		if v, ok := data.(HttpResponseRenderer); ok {
			return v
		}

		return &SuccessResponse{Data: data, Warnings: warnings}
	} else {
		return &EmptyResponse{}
	}
}

type EmptyResponse struct{}

func (r *EmptyResponse) Render(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type SuccessResponse struct {
	Data     interface{}
	Warnings []error
}

func NewSuccessResponse(data interface{}, warnings []error) *SuccessResponse {
	return &SuccessResponse{Data: data, Warnings: warnings}
}

func (r *SuccessResponse) Render(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	if len(r.Warnings) > 0 {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(http.StatusOK)

	resp := &Response{
		Status: StatusSuccess,
		Data:   r.Data,
	}
	for _, warn := range r.Warnings {
		resp.Warnings = append(resp.Warnings, warn.Error())
	}
	return json.NewEncoder(w).Encode(resp)
}

type ErrorResponse struct {
	Data   interface{}
	ApiErr *ApiError
}

func (r *ErrorResponse) Render(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")

	var code int
	switch r.ApiErr.Typ {
	case ErrorBadData:
		code = http.StatusBadRequest
	case ErrorExec:
		code = 422
	case ErrorCanceled, ErrorTimeout:
		code = http.StatusServiceUnavailable
	case ErrorInternal:
		code = http.StatusInternalServerError
	case ErrorNotFound:
		code = http.StatusNotFound
	default:
		code = http.StatusInternalServerError
	}
	w.WriteHeader(code)

	return json.NewEncoder(w).Encode(&Response{
		Status:    StatusError,
		ErrorType: r.ApiErr.Typ,
		Error:     r.ApiErr.Err.Error(),
		Data:      r.Data,
	})
}
