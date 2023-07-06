// Copyright 2022-2023 The Parca Authors
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

//nolint:nonamedreturn
package objectstore

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/azure"
	"github.com/thanos-io/objstore/providers/bos"
	"github.com/thanos-io/objstore/providers/cos"
	"github.com/thanos-io/objstore/providers/filesystem"
	"github.com/thanos-io/objstore/providers/gcs"
	"github.com/thanos-io/objstore/providers/obs"
	"github.com/thanos-io/objstore/providers/oci"
	"github.com/thanos-io/objstore/providers/oss"
	"github.com/thanos-io/objstore/providers/s3"
	"github.com/thanos-io/objstore/providers/swift"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"
)

func NewBucket(tracer trace.Tracer, logger log.Logger, confContentYaml []byte, reg prometheus.Registerer, component string) (objstore.InstrumentedBucket, error) {
	level.Info(logger).Log("msg", "loading bucket configuration")
	bucketConf := &client.BucketConfig{}
	if err := yaml.UnmarshalStrict(confContentYaml, bucketConf); err != nil {
		return nil, fmt.Errorf("parsing config YAML file: %w", err)
	}

	config, err := yaml.Marshal(bucketConf.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal content of bucket configuration: %w", err)
	}

	var bucket objstore.Bucket
	switch strings.ToUpper(string(bucketConf.Type)) {
	case string(client.GCS):
		bucket, err = gcs.NewBucket(context.Background(), logger, config, component)
	case string(client.S3):
		bucket, err = s3.NewBucket(logger, config, component)
	case string(client.AZURE):
		bucket, err = azure.NewBucket(logger, config, component)
	case string(client.SWIFT):
		bucket, err = swift.NewContainer(logger, config)
	case string(client.COS):
		bucket, err = cos.NewBucket(logger, config, component)
	case string(client.ALIYUNOSS):
		bucket, err = oss.NewBucket(logger, config, component)
	case string(client.FILESYSTEM):
		bucket, err = filesystem.NewBucketFromConfig(config)
	case string(client.BOS):
		bucket, err = bos.NewBucket(logger, config, component)
	case string(client.OCI):
		bucket, err = oci.NewBucket(logger, config)
	case string(client.OBS):
		bucket, err = obs.NewBucket(logger, config)
	default:
		return nil, fmt.Errorf("bucket with type %s is not supported", bucketConf.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("create %s client", bucketConf.Type)
	}

	return NewTracingBucket(tracer, objstore.BucketWithMetrics(bucket.Name(), objstore.NewPrefixedBucket(bucket, bucketConf.Prefix), reg)), nil
}

// TracingBucket is a wrapper around objstore.Bucket that adds tracing to all operations using OpenTelemetry.
type TracingBucket struct {
	tracer trace.Tracer
	bkt    objstore.Bucket
}

func NewTracingBucket(tracer trace.Tracer, bkt objstore.Bucket) objstore.InstrumentedBucket {
	return TracingBucket{tracer: tracer, bkt: bkt}
}

func (t TracingBucket) Iter(ctx context.Context, dir string, f func(string) error, options ...objstore.IterOption) (err error) {
	ctx, span := t.tracer.Start(ctx, "bucket_iter")
	defer span.End()
	span.SetAttributes(attribute.String("dir", dir))

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	return t.bkt.Iter(ctx, dir, f, options...)
}

func (t TracingBucket) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	ctx, span := t.tracer.Start(ctx, "bucket_get")
	defer span.End()
	span.SetAttributes(attribute.String("name", name))

	r, err := t.bkt.Get(ctx, name)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return newTracingReadCloser(r, span), nil
}

func (t TracingBucket) GetRange(ctx context.Context, name string, off, length int64) (io.ReadCloser, error) {
	ctx, span := t.tracer.Start(ctx, "bucket_getrange")
	defer span.End()
	span.SetAttributes(attribute.String("name", name), attribute.Int64("offset", off), attribute.Int64("length", length))

	r, err := t.bkt.GetRange(ctx, name, off, length)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	return newTracingReadCloser(r, span), nil
}

func (t TracingBucket) Exists(ctx context.Context, name string) (_ bool, err error) {
	ctx, span := t.tracer.Start(ctx, "bucket_exists")
	defer span.End()
	span.SetAttributes(attribute.String("name", name))

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	return t.bkt.Exists(ctx, name)
}

func (t TracingBucket) Attributes(ctx context.Context, name string) (_ objstore.ObjectAttributes, err error) {
	ctx, span := t.tracer.Start(ctx, "bucket_attributes")
	defer span.End()
	span.SetAttributes(attribute.String("name", name))

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	return t.bkt.Attributes(ctx, name)
}

func (t TracingBucket) Upload(ctx context.Context, name string, r io.Reader) (err error) {
	ctx, span := t.tracer.Start(ctx, "bucket_upload")
	defer span.End()
	span.SetAttributes(attribute.String("name", name))

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	return t.bkt.Upload(ctx, name, r)
}

func (t TracingBucket) Delete(ctx context.Context, name string) (err error) {
	ctx, span := t.tracer.Start(ctx, "bucket_delete")
	defer span.End()
	span.SetAttributes(attribute.String("name", name))

	defer func() {
		if err != nil {
			span.RecordError(err)
		}
	}()
	return t.bkt.Delete(ctx, name)
}

func (t TracingBucket) Name() string {
	return "tracing: " + t.bkt.Name()
}

func (t TracingBucket) Close() error {
	return t.bkt.Close()
}

func (t TracingBucket) IsObjNotFoundErr(err error) bool {
	return t.bkt.IsObjNotFoundErr(err)
}

func (t TracingBucket) WithExpectedErrs(expectedFunc objstore.IsOpFailureExpectedFunc) objstore.Bucket {
	if ib, ok := t.bkt.(objstore.InstrumentedBucket); ok {
		return TracingBucket{tracer: t.tracer, bkt: ib.WithExpectedErrs(expectedFunc)}
	}
	return t
}

func (t TracingBucket) ReaderWithExpectedErrs(expectedFunc objstore.IsOpFailureExpectedFunc) objstore.BucketReader {
	return t.WithExpectedErrs(expectedFunc)
}

type tracingReadCloser struct {
	r io.ReadCloser
	s trace.Span

	objSize    int64
	objSizeErr error

	read int
}

func newTracingReadCloser(r io.ReadCloser, span trace.Span) io.ReadCloser {
	// Since TryToGetSize can only reliably return size before doing any read calls,
	// we call during "construction" and remember the results.
	objSize, objSizeErr := objstore.TryToGetSize(r)

	return &tracingReadCloser{r: r, s: span, objSize: objSize, objSizeErr: objSizeErr}
}

func (t *tracingReadCloser) ObjectSize() (int64, error) {
	return t.objSize, t.objSizeErr
}

func (t *tracingReadCloser) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if n > 0 {
		t.read += n
	}
	if err != nil && err != io.EOF && t.s != nil {
		t.s.RecordError(err)
	}
	return n, err
}

func (t *tracingReadCloser) Close() error {
	err := t.r.Close()
	if t.s != nil {
		t.s.SetAttributes(attribute.Int64("read", int64(t.read)))
		if err != nil {
			t.s.SetAttributes(attribute.String("close_err", err.Error()))
		}
		t.s.End()
		t.s = nil
	}
	return err
}
