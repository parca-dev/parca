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

package index

import (
	"math"
	"sync"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/pkg/labels"
)

var allPostingsKey = labels.Label{}

// AllPostingsKey returns the label key that is used to store the postings list of all existing IDs.
func AllPostingsKey() (name, value string) {
	return allPostingsKey.Name, allPostingsKey.Value
}

type MemPostings struct {
	mtx sync.RWMutex
	m   map[string]map[string]*sroar.Bitmap
}

func NewMemPostings() *MemPostings {
	p := &MemPostings{
		mtx: sync.RWMutex{},
		m:   map[string]map[string]*sroar.Bitmap{},
	}
	p.m[allPostingsKey.Name] = map[string]*sroar.Bitmap{}
	p.m[allPostingsKey.Name][allPostingsKey.Value] = sroar.NewBitmap()
	return p
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

	p.m[allPostingsKey.Name][allPostingsKey.Value].Set(id)

	p.mtx.Unlock()
}

func (p *MemPostings) Get(name, value string) *sroar.Bitmap {
	bm := sroar.NewBitmap()

	p.mtx.RLock()
	l := p.m[name]
	if l != nil {
		if l[value] != nil {
			bm = l[value].Clone()
		}
	}
	p.mtx.RUnlock()

	if bm.IsEmpty() {
		bm.Set(math.MaxUint64) // This is an errPostings bitmap
	}

	return bm
}

// LabelNames returns all the unique label names.
func (p *MemPostings) LabelNames() []string {
	p.mtx.RLock()
	defer p.mtx.RUnlock()
	n := len(p.m)
	if n == 0 {
		return nil
	}

	names := make([]string, 0, n-1)
	for name := range p.m {
		if name != allPostingsKey.Name {
			names = append(names, name)
		}
	}
	return names
}

// LabelValues returns label values for the given name.
func (p *MemPostings) LabelValues(name string) []string {
	p.mtx.RLock()
	defer p.mtx.RUnlock()

	values := make([]string, 0, len(p.m[name]))
	for v := range p.m[name] {
		values = append(values, v)
	}
	return values
}
