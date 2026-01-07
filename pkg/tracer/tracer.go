// Copyright 2022-2026 The Parca Authors
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

// Copyright (c) The Thanos Authors.
// Licensed under the Apache License 2.0.

package tracer

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type (
	ExporterType string
	SamplerType  string
)

const (
	ExporterTypeGRPC  ExporterType = "grpc"
	ExporterTypeHTTP  ExporterType = "http"
	ExporterTypeStdio ExporterType = "stdout"

	SamplerTypeAlways     SamplerType = "always"
	SamplerTypeNever      SamplerType = "never"
	SamplerTypeRatioBased SamplerType = "ratio_based"
)

type NoopExporter struct{}

func (n NoopExporter) ExportSpans(_ context.Context, _ []sdktrace.ReadOnlySpan) error {
	return nil
}

func (n NoopExporter) MarshalLog() interface{} {
	return nil
}

func (n NoopExporter) Shutdown(_ context.Context) error {
	return nil
}

func (n NoopExporter) Start(_ context.Context) error {
	return nil
}

func NewNoopExporter() *NoopExporter {
	return &NoopExporter{}
}

type Exporter interface {
	sdktrace.SpanExporter

	Start(context.Context) error
}

// NewProvider returns an OTLP exporter based tracer.
func NewProvider(ctx context.Context, version string, exporter sdktrace.SpanExporter, opts ...sdktrace.TracerProviderOption) (trace.TracerProvider, error) {
	if exporter == nil {
		return noop.NewTracerProvider(), nil
	}

	res, err := resources(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	provider := sdktrace.NewTracerProvider(
		// Sampler options:
		// - sdktrace.NeverSample()
		// - sdktrace.TraceIDRatioBased(0.01)
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(sdktrace.NewBatchSpanProcessor(exporter)),
	)
	// Set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(provider)

	return provider, nil
}

func NewExporter(exType, otlpAddress string, otlpInsecure bool) (Exporter, error) {
	switch strings.ToLower(exType) {
	case string(ExporterTypeGRPC):
		return NewGRPCExporter(otlpAddress, otlpInsecure)
	case string(ExporterTypeHTTP):
		return NewHTTPExporter(otlpAddress, otlpInsecure)
	case string(ExporterTypeStdio):
		return NewConsoleExporter(os.Stdout)
	default:
		return NewNoopExporter(), fmt.Errorf("unknown exporter type: %s", exType)
	}
}

type consoleExporter struct {
	*stdouttrace.Exporter
}

func (c *consoleExporter) Start(_ context.Context) error {
	return nil
}

// NewConsoleExporter returns a console exporter.
func NewConsoleExporter(w io.Writer) (Exporter, error) {
	exp, err := stdouttrace.New(
		stdouttrace.WithWriter(w),
		// Use human-readable output.
		stdouttrace.WithPrettyPrint(),
		// Do not print timestamps for the demo.
		stdouttrace.WithoutTimestamps(),
	)
	if err != nil {
		return nil, err
	}
	return &consoleExporter{exp}, nil
}

// NewGRPCExporter returns a gRPC exporter.
func NewGRPCExporter(otlpAddress string, otlpInsecure bool) (Exporter, error) {
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(otlpAddress)}
	if otlpInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	return otlptracegrpc.NewUnstarted(opts...), nil
}

// NewHTTPExporter returns a HTTP exporter.
func NewHTTPExporter(otlpAddress string, otlpInsecure bool) (Exporter, error) {
	opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(otlpAddress)}
	if otlpInsecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}
	return otlptrace.NewUnstarted(otlptracehttp.NewClient(
		opts...,
	)), nil
}

func resources(ctx context.Context, version string) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("parca"),
			semconv.ServiceVersionKey.String(version),
		),
		resource.WithFromEnv(),   // pull attributes from OTEL_RESOURCE_ATTRIBUTES and OTEL_SERVICE_NAME environment variables
		resource.WithProcess(),   // This option configures a set of Detectors that discover process information
		resource.WithOS(),        // This option configures a set of Detectors that discover OS information
		resource.WithContainer(), // This option configures a set of Detectors that discover container information
		resource.WithHost(),      // This option configures a set of Detectors that discover host information
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	return res, nil
}
