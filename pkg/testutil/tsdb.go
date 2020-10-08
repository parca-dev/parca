// Copyright 2020 The conprof Authors
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

package testutil

import (
	"io/ioutil"
	"math"

	"github.com/conprof/db/tsdb"
)

func NewTSDB() (*tsdb.DB, error) {
	dir, err := ioutil.TempDir("", "conprof-test")
	if err != nil {
		return nil, err
	}
	opts := tsdb.DefaultOptions()
	opts.RetentionDuration = math.MaxInt64
	return tsdb.Open(dir, nil, nil, opts)
}
