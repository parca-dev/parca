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
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dgraph-io/sroar"
	"github.com/parca-dev/parca/pkg/storage/index"
	"github.com/prometheus/prometheus/pkg/labels"
)

// Bitmap used by func isRegexMetaCharacter to check whether a character needs to be escaped.
var regexMetaCharacterBytes [16]byte

// isRegexMetaCharacter reports whether byte b needs to be escaped.
func isRegexMetaCharacter(b byte) bool {
	return b < utf8.RuneSelf && regexMetaCharacterBytes[b%16]&(1<<(b/16)) != 0
}

func init() {
	// The following characters need to be escaped.
	// These characters are used in queries like {foo="(bar|baz*)"}
	for _, b := range []byte(`.+*?()|[]{}^$`) {
		regexMetaCharacterBytes[b%16] |= 1 << (b / 16)
	}
}

// PostingsForMatchers assembles a single postings iterator against the index reader
// based on the given matchers. The resulting postings are not ordered by series.
func PostingsForMatchers(ix IndexReader, ms ...*labels.Matcher) (*sroar.Bitmap, error) {
	bitmap := sroar.NewBitmap()
	noBitmap := sroar.NewBitmap()

	// See which label must be non-empty.
	// Optimization for case like {l=~".", l!="1"}.
	labelMustBeSet := make(map[string]bool, len(ms))
	for _, m := range ms {
		if !m.Matches("") {
			labelMustBeSet[m.Name] = true
		}
	}

	for _, m := range ms {
		if labelMustBeSet[m.Name] {
			// If this matcher must be non-empty, we can be smarter.
			matchesEmpty := m.Matches("")
			isNot := m.Type == labels.MatchNotEqual || m.Type == labels.MatchNotRegexp

			if isNot && matchesEmpty { // l!="foo"
				inverse, err := m.Inverse()
				if err != nil {
					return nil, err
				}
				bm, err := postingsForMatcher(ix, inverse)
				if err != nil {
					return nil, err
				}
				noBitmap.Or(bm)
			} else if isNot && !matchesEmpty { // l!=""
				inverse, err := m.Inverse()
				if err != nil {
					return nil, err
				}
				bm, err := inversePostingsForMatcher(ix, inverse)
				if err != nil {
					return nil, err
				}

				if bitmap.IsEmpty() {
					bitmap = bm
				} else {
					bitmap.And(bm)
				}
			} else { // l="a"
				// Non-Not matcher, use normal postingsForMatcher.
				bm, err := postingsForMatcher(ix, m)
				if err != nil {
					return nil, err
				}

				if bitmap.IsEmpty() {
					bitmap = bm
				} else {
					bitmap.And(bm)
				}
			}
		} else { // l=""
			// If the matchers for a labelname selects an empty value, it selects all
			// the series which don't have the label name set too. See:
			// https://github.com/prometheus/prometheus/issues/3575 and
			// https://github.com/prometheus/prometheus/pull/3578#issuecomment-351653555
			bm, err := inversePostingsForMatcher(ix, m)
			if err != nil {
				return nil, err
			}
			if noBitmap.IsEmpty() {
				noBitmap = bm
			} else {
				noBitmap.Or(bm)
			}
		}
	}

	// If there's nothing to subtract from, add in everything and remove the noBitmap later.
	//if bitmap.IsEmpty() && !noBitmap.IsEmpty() {
	if bitmap.GetCardinality() == 0 && noBitmap.GetCardinality() != 0 {
		allPostings, err := ix.Postings(index.AllPostingsKey())
		if err != nil {
			return nil, err
		}
		bitmap.Or(allPostings)
	}

	// If either of bitmaps contain the special MaxUint64
	// we need to make sure to have it in the other to delete it for good.
	noBitmap.Set(math.MaxUint64)
	bitmap.Set(math.MaxUint64)

	// Intersect to remove the unwanted postings
	bitmap.AndNot(noBitmap)

	return bitmap, nil
}

func postingsForMatcher(ix IndexReader, m *labels.Matcher) (*sroar.Bitmap, error) {
	// This method will not return postings for missing labels.

	// Fast-path for equal matching.
	if m.Type == labels.MatchEqual {
		return ix.Postings(m.Name, m.Value)
	}

	// Fast-path for set matching.
	if m.Type == labels.MatchRegexp {
		setMatches := findSetMatches(m.GetRegexString())
		if len(setMatches) > 0 {
			sort.Strings(setMatches)
			return ix.Postings(m.Name, setMatches...)
		}
	}

	vals, err := ix.LabelValues(m.Name)
	if err != nil {
		return nil, err
	}

	var res []string
	lastVal, isSorted := "", true
	for _, val := range vals {
		if m.Matches(val) {
			res = append(res, val)
			if isSorted && val < lastVal {
				isSorted = false
			}
			lastVal = val
		}
	}

	if len(res) == 0 {
		return sroar.NewBitmap(), nil
	}

	if !isSorted {
		sort.Strings(res)
	}
	return ix.Postings(m.Name, res...)
}

func findSetMatches(pattern string) []string {
	// Return empty matches if the wrapper from Prometheus is missing.
	if len(pattern) < 6 || pattern[:4] != "^(?:" || pattern[len(pattern)-2:] != ")$" {
		return nil
	}
	escaped := false
	sets := []*strings.Builder{{}}
	for i := 4; i < len(pattern)-2; i++ {
		if escaped {
			switch {
			case isRegexMetaCharacter(pattern[i]):
				sets[len(sets)-1].WriteByte(pattern[i])
			case pattern[i] == '\\':
				sets[len(sets)-1].WriteByte('\\')
			default:
				return nil
			}
			escaped = false
		} else {
			switch {
			case isRegexMetaCharacter(pattern[i]):
				if pattern[i] == '|' {
					sets = append(sets, &strings.Builder{})
				} else {
					return nil
				}
			case pattern[i] == '\\':
				escaped = true
			default:
				sets[len(sets)-1].WriteByte(pattern[i])
			}
		}
	}
	matches := make([]string, 0, len(sets))
	for _, s := range sets {
		if s.Len() > 0 {
			matches = append(matches, s.String())
		}
	}
	return matches
}

func labelValuesWithMatchers(r IndexReader, name string, matchers ...*labels.Matcher) ([]string, error) {
	// We're only interested in metrics which have the label <name>.
	requireLabel, err := labels.NewMatcher(labels.MatchNotEqual, name, "")
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate label matcher: %w", err)
	}

	bm, err := PostingsForMatchers(r, append(matchers, requireLabel)...)
	if err != nil {
		return nil, err
	}

	dedupe := map[string]interface{}{}

	it := bm.NewIterator()
	for it.HasNext() {
		v, err := r.LabelValueFor(it.Next(), name)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}
		dedupe[v] = nil
	}

	values := make([]string, 0, len(dedupe))
	for value := range dedupe {
		values = append(values, value)
	}

	return values, nil
}

func labelNamesWithMatchers(r IndexReader, matchers ...*labels.Matcher) ([]string, error) {
	p, err := PostingsForMatchers(r, matchers...)
	if err != nil {
		return nil, err
	}

	return r.LabelNamesFor(p.ToArray()...)
}

// inversePostingsForMatcher returns the postings for the series with the label name set but not matching the matcher.
func inversePostingsForMatcher(ix IndexReader, m *labels.Matcher) (*sroar.Bitmap, error) {
	vals, err := ix.LabelValues(m.Name)
	if err != nil {
		return nil, err
	}

	var res []string
	lastVal, isSorted := "", true
	for _, val := range vals {
		if !m.Matches(val) {
			res = append(res, val)
			if isSorted && val < lastVal {
				isSorted = false
			}
			lastVal = val
		}
	}

	if !isSorted {
		sort.Strings(res)
	}
	return ix.Postings(m.Name, res...)
}
