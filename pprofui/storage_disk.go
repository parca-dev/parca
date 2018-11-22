// Copyright 2018 The conprof Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.


package pprofui

import (
	"encoding/base64"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/oklog/ulid"
	"github.com/pkg/errors"
)

// A DiskStorage is a Storage implementation that reads profiles from disk.
type DiskStorage struct {
	path string
}

var _ Storage = &DiskStorage{}

// NewDiskStorage creates a DiskStorage that reads profiles from disk.
func NewDiskStorage(path string) *DiskStorage {
	return &DiskStorage{
		path: path,
	}
}

// ID implements Storage.
func (s *DiskStorage) ID() string {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}

// Store implements Storage.
func (s *DiskStorage) Store(id string, write func(io.Writer) error) error {
	return errors.New("not implemented")
}

// Get implements Storage.
func (s *DiskStorage) Get(series, timestamp string, read func(io.Reader) error) error {
	f, err := os.Open(path.Join(s.path, base64.URLEncoding.EncodeToString([]byte(series)), timestamp))
	if err != nil {
		return err
	}
	defer f.Close()
	return read(f)
}

// List implements Storage.
func (s *DiskStorage) List() (map[string][]string, error) {
	seriesDirs, err := ioutil.ReadDir(s.path)
	if err != nil {
		return nil, err
	}

	series := make(map[string][]string, len(seriesDirs))

	for _, seriesDir := range seriesDirs {
		seriesDirName := seriesDir.Name()
		if !seriesDir.IsDir() || seriesDirName == "." || seriesDirName == ".." {
			continue
		}
		decodedSeriesName, err := base64.URLEncoding.DecodeString(seriesDirName)
		if err != nil {
			return nil, err
		}
		seriesName := string(decodedSeriesName)

		files, err := ioutil.ReadDir(filepath.Join(s.path, seriesDirName))
		if err != nil {
			return nil, err
		}

		series[seriesName] = make([]string, len(files))
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			series[seriesName] = append(series[seriesName], file.Name())
		}
	}

	return series, nil
}
