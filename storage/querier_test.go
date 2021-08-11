package storage

import (
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestPostingsForMatchers(t *testing.T) {
	h := NewHead()
	h.minTime = *atomic.NewInt64(-1)
	h.maxTime = *atomic.NewInt64(1)
	h.postings.Add(0, labels.Labels{{"n", "1"}})
	h.postings.Add(1, labels.Labels{{"n", "1"}, {"i", "a"}})
	h.postings.Add(2, labels.Labels{{"n", "1"}, {"i", "b"}})
	h.postings.Add(3, labels.Labels{{"n", "2"}})
	h.postings.Add(4, labels.Labels{{"n", "2.5"}})

	ir := &headIndexReader{head: h}

	//var empty []uint64

	testcases := []struct {
		matchers []*labels.Matcher
		exp      []uint64
	}{{
		// Simple equals.
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
		},
		exp: []uint64{0, 1, 2},
	}, {
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "n", Value: "1"},
			{Type: labels.MatchEqual, Name: "i", Value: "a"},
		},
		exp: []uint64{1},
		//}, {
		// TODO: Still returns all
		//	matchers: []*labels.Matcher{
		//		{Type: labels.MatchEqual, Name: "n", Value: "1"},
		//		{Type: labels.MatchEqual, Name: "i", Value: "missing"},
		//	},
		//	exp: empty,
	}, {
		matchers: []*labels.Matcher{
			{Type: labels.MatchEqual, Name: "missing", Value: ""},
		},
		exp: []uint64{0, 1, 2, 3, 4}, // all
	}, {
		// Not equals.
		matchers: []*labels.Matcher{
			{Type: labels.MatchNotEqual, Name: "n", Value: "1"},
		},
		exp: []uint64{3, 4},
	}, {
		matchers: []*labels.Matcher{
			{Type: labels.MatchNotEqual, Name: "i", Value: ""},
		},
		exp: []uint64{1, 2},
	}}
	for i, tc := range testcases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			bm, err := PostingsForMatchers(ir, tc.matchers...)
			require.NoError(t, err)
			require.Equal(t, tc.exp, bm.ToArray())
		})
	}
}
