package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/common/route"
	extpromhttp "github.com/thanos-io/thanos/pkg/extprom/http"
	"github.com/thanos-io/thanos/pkg/server/http/middleware"
)

type status string

const (
	StatusSuccess status = "success"
	StatusError   status = "error"
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

type response struct {
	Status    status      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorType ErrorType   `json:"errorType,omitempty"`
	Error     string      `json:"error,omitempty"`
	Warnings  []string    `json:"warnings,omitempty"`
}

type httpResponseRenderer interface {
	Render(w http.ResponseWriter)
}

type ApiFunc func(r *http.Request) (interface{}, []error, *ApiError)

// TODO: add tracer
// Instr returns a http HandlerFunc with the instrumentation middleware.
func Instr(
	_ log.Logger,
	ins extpromhttp.InstrumentationMiddleware,
) func(name string, f ApiFunc) httprouter.Handle {
	instr := func(name string, f ApiFunc) httprouter.Handle {
		hf := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			data, warnings, err := f(r)
			ren := chooseRenderer(data, warnings, err)
			ren.Render(w)
		})
		return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()

			for _, p := range params {
				ctx = route.WithParam(ctx, p.Key, p.Value)
			}
			ins.NewHandler(name, gziphandler.GzipHandler(middleware.RequestID(hf))).ServeHTTP(w, r.WithContext(ctx))
		}
	}
	return instr
}

func chooseRenderer(data interface{}, warnings []error, err *ApiError) httpResponseRenderer {
	if err != nil {
		return &errorResponse{Data: data, ApiErr: err}
	}
	if data != nil {
		if v, ok := data.(httpResponseRenderer); ok {
			return v
		}

		return &successResponse{Data: data, Warnings: warnings}
	} else {
		return &emptyResponse{}
	}
}

type emptyResponse struct{}

func (r *emptyResponse) Render(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

type successResponse struct {
	Data     interface{}
	Warnings []error
}

func (r *successResponse) Render(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	if len(r.Warnings) > 0 {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(http.StatusOK)

	resp := &response{
		Status: StatusSuccess,
		Data:   r.Data,
	}
	for _, warn := range r.Warnings {
		resp.Warnings = append(resp.Warnings, warn.Error())
	}
	_ = json.NewEncoder(w).Encode(resp)
}

type errorResponse struct {
	Data   interface{}
	ApiErr *ApiError
}

func (r *errorResponse) Render(w http.ResponseWriter) {
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

	_ = json.NewEncoder(w).Encode(&response{
		Status:    StatusError,
		ErrorType: r.ApiErr.Typ,
		Error:     r.ApiErr.Err.Error(),
		Data:      r.Data,
	})
}
