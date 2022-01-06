// Copyright 2021 The Parca Authors
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

package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestHeadIndexReader_Postings(t *testing.T) {
	ir := headIndexReader{head: NewHead(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)}
	ir.head.postings.Add(1, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test1"}})
	ir.head.postings.Add(2, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test2"}})
	ir.head.postings.Add(3, labels.Labels{{Name: "foo", Value: "baz"}, {Name: "container", Value: "test3"}})

	bm, err := ir.Postings("foo", "bar")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2}, bm.ToArray())

	bm, err = ir.Postings("foo", "bar", "baz")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, bm.ToArray())
}

func TestHeadIndexReader_LabelValues(t *testing.T) {
	ctx := context.Background()
	h := NewHead(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)

	for i := 0; i < 100; i++ {
		app, err := h.Appender(context.Background(), labels.Labels{
			{Name: "unique", Value: fmt.Sprintf("value%d", i)},
			{Name: "tens", Value: fmt.Sprintf("value%d", i/10)},
		})
		require.NoError(t, err)
		err = app.AppendFlat(ctx, &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{
				Timestamp: int64(100 + i),
			},
		})
		require.NoError(t, err)
	}

	ir, err := h.Index()
	require.NoError(t, err)

	testcases := []struct {
		name      string
		labelName string
		matchers  []*labels.Matcher
		exp       []string
	}{{
		name:      "get tens based on unique id",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchEqual, "unique", "value35")},
		exp:       []string{"value3"},
	}, {
		name:      "get unique ids based on a ten",
		labelName: "unique",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchEqual, "tens", "value1")},
		exp:       []string{"value10", "value11", "value12", "value13", "value14", "value15", "value16", "value17", "value18", "value19"},
	}, {
		name:      "get tens by pattern matching on unique id",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "unique", "value[5-7]5")},
		exp:       []string{"value5", "value6", "value7"},
	}, {
		name:      "get tens by matching for absence of unique label",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchNotEqual, "unique", "")},
		exp:       []string{"value0", "value1", "value2", "value3", "value4", "value5", "value6", "value7", "value8", "value9"},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := ir.LabelValues(tc.labelName, tc.matchers...)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.exp, values)
		})
	}
}
