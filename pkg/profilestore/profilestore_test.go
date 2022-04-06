// Copyright 2022 The Parca Authors
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

package profilestore

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"

	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/storage"
)

func Test_LabelName_Invalid(t *testing.T) {
	mStr := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		mStr.Close()
	})

	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	pStr := NewProfileStore(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		mStr,
	)

	ctx := context.Background()

	req := &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{{
					Name:  "n0:n",
					Value: "v0",
				}},
			},
		}},
	}

	_, err := pStr.WriteRaw(ctx, req)
	st, _ := status.FromError(err)

	require.Equal(t, st.Code(), codes.InvalidArgument)
}
