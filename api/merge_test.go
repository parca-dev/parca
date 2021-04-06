package api

import (
	"io/ioutil"
	"testing"

	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb/tsdbutil"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

type sample struct {
	t int64
	v []byte
}

func (s *sample) T() int64 {
	return s.t
}

func (s *sample) V() []byte {
	return s.v
}

type sliceSeriesSet struct {
	s   []storage.Series
	cur int
}

func newSliceSeriesSet(s []storage.Series) *sliceSeriesSet {
	return &sliceSeriesSet{
		s:   s,
		cur: -1,
	}
}

func (s *sliceSeriesSet) Next() bool {
	s.cur++
	return s.cur < len(s.s)
}

func (s *sliceSeriesSet) At() storage.Series {
	return s.s[s.cur]
}

func (s *sliceSeriesSet) Err() error {
	return nil
}

func (s *sliceSeriesSet) Warnings() storage.Warnings {
	return nil
}

func TestBatchIteratorNoSeries(t *testing.T) {
	set := newSliceSeriesSet([]storage.Series{})

	i := newBatchIterator(set, 2)
	require.False(t, i.Next())
}

func TestBatchIteratorNoSamples(t *testing.T) {
	set := newSliceSeriesSet([]storage.Series{
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "a"}}, []tsdbutil.Sample{}),
	})

	i := newBatchIterator(set, 2)
	require.False(t, i.Next())
}

func TestBatchIteratorSingleSeries(t *testing.T) {
	set := newSliceSeriesSet([]storage.Series{
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "a"}}, []tsdbutil.Sample{
			&sample{t: 0, v: []byte("a")},
			&sample{t: 0, v: []byte("b")},
			&sample{t: 0, v: []byte("c")},
			&sample{t: 0, v: []byte("d")},
			&sample{t: 0, v: []byte("e")},
		}),
	})

	i := newBatchIterator(set, 2)
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("a"), []byte("b")}, i.Batch())
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("c"), []byte("d")}, i.Batch())
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("e")}, i.Batch())
	require.False(t, i.Next())
}

func TestBatchIteratorMultipleSeries(t *testing.T) {
	set := newSliceSeriesSet([]storage.Series{
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "a"}}, []tsdbutil.Sample{
			&sample{t: 0, v: []byte("a")},
		}),
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "b"}}, []tsdbutil.Sample{
			&sample{t: 0, v: []byte("b")},
			&sample{t: 0, v: []byte("c")},
			&sample{t: 0, v: []byte("d")},
			&sample{t: 0, v: []byte("e")},
		}),
	})

	i := newBatchIterator(set, 2)
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("a"), []byte("b")}, i.Batch())
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("c"), []byte("d")}, i.Batch())
	require.True(t, i.Next())
	require.EqualValues(t, [][]byte{[]byte("e")}, i.Batch())
	require.False(t, i.Next())
}

func TestMergeSeriesSet(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	set := newSliceSeriesSet([]storage.Series{
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "a"}}, []tsdbutil.Sample{
			&sample{t: 0, v: b},
		}),
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "b"}}, []tsdbutil.Sample{
			&sample{t: 0, v: b},
			&sample{t: 0, v: b},
			&sample{t: 0, v: b},
			&sample{t: 0, v: b},
		}),
	})

	_, _, err = mergeSeriesSet(set, 2)
	require.NoError(t, err)
}

func TestMergeSeriesSetSingleSample(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	set := newSliceSeriesSet([]storage.Series{
		storage.NewListSeries(labels.Labels{{Name: "instance", Value: "a"}}, []tsdbutil.Sample{
			&sample{t: 0, v: b},
		}),
	})

	_, _, err = mergeSeriesSet(set, 2)
	require.NoError(t, err)
}
