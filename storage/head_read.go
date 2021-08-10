package storage

import (
	"math"

	"github.com/prometheus/tsdb/index"
)

// IndexReader provides reading access of serialized index data.
type IndexReader interface {
	// Postings returns the postings list iterator for the label pairs.
	// The Postings here contain the offsets to the series inside the index.
	// Found IDs are not strictly required to point to a valid Series, e.g.
	// during background garbage collections. Input values must be sorted.
	Postings(name string, values ...string) (index.Postings, error)

	// Close releases the underlying resource of the reader.
	Close() error
}

// Index returns an IndexReader against the block.
func (h *Head) Index() (IndexReader, error) {
	return h.indexRange(math.MinInt64, math.MaxInt64), nil
}

func (h *Head) indexRange(mint, maxt int64) *headIndexReader {
	if hmin := h.MinTime(); hmin > mint {
		mint = hmin
	}
	return &headIndexReader{head: h, mint: mint, maxt: maxt}
}

type headIndexReader struct {
	head       *Head
	mint, maxt int64
}

func (h *headIndexReader) Close() error {
	return nil
}

//
//func (h *headIndexReader) Postings(name string, values ...string) (index.Postings, error) {
//	res := make([]Postings, 0, len(values))
//	for _, value := range values {
//		//res = append(res, h.head.postings[name][value].ToArray())
//		panic("continue here")
//	}
//}
