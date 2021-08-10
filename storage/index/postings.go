package index

import (
	"sync"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/pkg/labels"
)

type MemPostings struct {
	mtx sync.RWMutex
	m   map[string]map[string]*sroar.Bitmap
}

func NewMemPostings() *MemPostings {
	return &MemPostings{
		mtx: sync.RWMutex{},
		m:   map[string]map[string]*sroar.Bitmap{},
	}
}

func (p *MemPostings) Add(id uint64, lset labels.Labels) {
	p.mtx.Lock()

	if p.m == nil {
		p.m = map[string]map[string]*sroar.Bitmap{}
	}
	for _, l := range lset {
		if p.m[l.Name] == nil {
			p.m[l.Name] = map[string]*sroar.Bitmap{}
		}
		if p.m[l.Name][l.Value] == nil {
			p.m[l.Name][l.Value] = sroar.NewBitmap()
		}
		p.m[l.Name][l.Value].Set(id)
	}

	p.mtx.Unlock()
}

// Postings provides iterative access over a postings list.
type Postings interface {
	Next() bool
	Seek(v uint64) bool
	At() uint64
	Err() error
}

func (p *MemPostings) Get(name, value string) Postings {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	if p.m[name] == nil || p.m[name][value] == nil {
		return nil // TODO: Return nopPostingIterator
	}

	return &bitmapIterator{
		it: p.m[name][value].NewIterator(),
	}
}

type bitmapIterator struct {
	it *sroar.Iterator
}

func (it *bitmapIterator) Next() bool {
	return it.it.HasNext()
}

func (it *bitmapIterator) Seek(v uint64) bool {
	panic("implement me")
}

func (it *bitmapIterator) At() uint64 {
	return it.it.Next()
}

func (it *bitmapIterator) Err() error {
	return nil
}
