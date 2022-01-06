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

import "github.com/parca-dev/parca/pkg/profile"

type ProfileSeriesIterator interface {
	Next() bool
	At() profile.InstantProfile
	Err() error
}

type ProfileSeries interface {
	Iterator() ProfileSeriesIterator
}

type SliceProfileSeriesIterator struct {
	samples []profile.InstantProfile
	i       int
	err     error
}

func (i *SliceProfileSeriesIterator) Next() bool {
	if i.err != nil {
		return false
	}

	i.i++
	return i.i < len(i.samples)
}

func (i *SliceProfileSeriesIterator) At() profile.InstantProfile {
	return i.samples[i.i]
}

func (i *SliceProfileSeriesIterator) Err() error {
	return i.err
}
