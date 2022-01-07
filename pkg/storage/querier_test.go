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
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

func TestPostingsForMatchers(t *testing.T) {
	h := NewHead(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	h.minTime = *atomic.NewInt64(-1)
	h.maxTime = *atomic.NewInt64(1)
	h.postings.Add(0, labels.Labels{{Name: "n", Value: "1"}})
	h.postings.Add(1, labels.Labels{{Name: "n", Value: "1"}, {Name: "i", Value: "a"}})
	h.postings.Add(2, labels.Labels{{Name: "n", Value: "1"}, {Name: "i", Value: "b"}})
	h.postings.Add(3, labels.Labels{{Name: "n", Value: "2"}})
	h.postings.Add(4, labels.Labels{{Name: "n", Value: "2.5"}})

	ir := &headIndexReader{head: h}

	testcases := []struct {
		name     string
		matchers []*labels.Matcher
		exp      []uint64
	}{{
		// Simple equals.
		name: `n="1"`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
		},
		exp: []uint64{0, 1, 2},
	}, {
		name: `n="1",i="a"`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
			{Type: labels.MatchEqual, Name: "i", Value: "a"},
		},
		exp: []uint64{1},
	}, {
		name: `n="1",i="missing"`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
			{Type: labels.MatchEqual, Name: "i", Value: "missing"},
		},
		exp: []uint64{},
	}, {
		name: `missing=""`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "missing", Value: ""},
		},
		exp: []uint64{0, 1, 2, 3, 4}, // all
	}, {
		// Not equals.
		name: `n!="1"`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchNotEqual, Name: "n", Value: "1"},
		},
		exp: []uint64{3, 4},
	}, {
		name: `n!=""`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchNotEqual, Name: "i", Value: ""},
		},
		exp: []uint64{1, 2},
	}, {
		name: `missing!=""`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchNotEqual, Name: "missing", Value: ""},
		},
		exp: []uint64{},
	}, {
		name: `n="1",n!="2"`,
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
			{Type: labels.MatchNotEqual, Name: "n", Value: "2"},
		},
		exp: []uint64{0, 1, 2},
	}, {
		name: `n="1",i!="a"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotEqual, "i", "a"),
		},
		exp: []uint64{0, 2},
	}, {
		name: `n="1",i!=""`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotEqual, "i", ""),
		},
		exp: []uint64{1, 2},
	}, {
		// Regex
		name: `n=~"^1$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchRegexp, "n", "^1$"),
		},
		exp: []uint64{0, 1, 2},
	}, {
		name: `n="1",i=~"^a$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^a$"),
		},
		exp: []uint64{1},
	}, {
		name: `n="1",i="^a?$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^a?$"),
		},
		exp: []uint64{0, 1},
	}, {
		name: `i=~"^$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^$"),
		},
		exp: []uint64{0, 3, 4},
	}, {
		name: `n="1",i=~"^$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^$"),
		},
		exp: []uint64{0},
	}, {
		name: `n="1",i=~"^.*$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^.*$"),
		},
		exp: []uint64{0, 1, 2},
	}, {
		name: `n="1",i="^.+$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^.+$"),
		},
		exp: []uint64{1, 2},
	}, {
		name: `n!~"^1$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchNotRegexp, "n", "^1$"),
		},
		exp: []uint64{3, 4},
	}, {
		name: `n="1",i!~"^a$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotRegexp, "i", "^a$"),
		},
		exp: []uint64{0, 2},
	}, {
		name: `n="1",i!~"^a?$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotRegexp, "i", "^a?$"),
		},
		exp: []uint64{2},
	}, {
		name: `n="1",i!~"^$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotRegexp, "i", "^$"),
		},
		exp: []uint64{1, 2},
	}, {
		name: `n="1",i!~"^.*$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotRegexp, "i", "^.*$"),
		},
		exp: []uint64{},
	}, {
		name: `n="1",i!~"^.+$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotRegexp, "i", "^.+$"),
		},
		exp: []uint64{0},
	}, {
		// Combinations.
		name: `n="1",i!="",i="a"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotEqual, "i", ""),
			labels.MustNewMatcher(labels.MatchEqual, "i", "a"),
		},
		exp: []uint64{1},
	}, {
		name: `n="1",i!="b",i=~"^(b|a).*$"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchEqual, "n", "1"),
			labels.MustNewMatcher(labels.MatchNotEqual, "i", "b"),
			labels.MustNewMatcher(labels.MatchRegexp, "i", "^(b|a).*$"),
		},
		exp: []uint64{1},
	}, {
		// Set optimization for Regex.
		// Refer to https://github.com/prometheus/prometheus/issues/2651.
		name: `n=~"1|2"`,
		matchers: []*labels.Matcher{
			labels.MustNewMatcher(labels.MatchRegexp, "n", "1|2"),
		},
		exp: []uint64{0, 1, 2, 3},
	}, {
		name:     `i=~"a|b"`,
		matchers: []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "i", "a|b")},
		exp:      []uint64{1, 2},
	}, {
		name:     `n=~"x1|2"`,
		matchers: []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "n", "x1|2")},
		exp:      []uint64{3},
	}, {
		name:     `n=~"2|2\\.5"`,
		matchers: []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "n", "2|2\\.5")},
		exp:      []uint64{3, 4},
	}, {
		// Empty value.
		name:     `i=~"c||d"`,
		matchers: []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "i", "c||d")},
		exp:      []uint64{0, 3, 4},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bm, err := PostingsForMatchers(ir, tc.matchers...)
			require.NoError(t, err)
			require.Equal(t, tc.exp, bm.ToArray())
		})
	}
}
