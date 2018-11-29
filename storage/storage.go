// Copyright 2018 The conprof Authors
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
	"encoding/base64"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/pkg/labels"
)

type Storage interface {
	Get(series, timestamp string, read func(io.Reader) error) error
	List() (map[string][]string, error)

	Appender() Appender
}

type diskStorage struct {
	logger log.Logger
	path   string
}

func NewDiskStorage(logger log.Logger, path string) Storage {
	return &diskStorage{
		logger: logger,
		path:   path,
	}
}

func (s *diskStorage) Get(series, timestamp string, read func(io.Reader) error) error {
	f, err := os.Open(path.Join(s.path, base64.URLEncoding.EncodeToString([]byte(series)), timestamp))
	if err != nil {
		return err
	}
	defer f.Close()
	return read(f)
}

func (s *diskStorage) List() (map[string][]string, error) {
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

func (s *diskStorage) Appender() Appender {
	return &storageAppender{path: s.path, logger: s.logger}
}

type Appender interface {
	Add(l labels.Labels, timestamp int64, profile []byte) error
}

type storageAppender struct {
	logger log.Logger
	path   string
}

func (s *storageAppender) Add(labels labels.Labels, timestamp int64, profile []byte) error {
	sort.Sort(labels)
	seriesString := labels.String()
	seriesString = base64.URLEncoding.EncodeToString([]byte(seriesString))

	p := filepath.Join(s.path, seriesString)
	err := os.MkdirAll(p, os.ModePerm)
	if err != nil {
		return err
	}

	fullpath := filepath.Join(p, intToString(timestamp))
	level.Debug(s.logger).Log("msg", "writing profile", "path", fullpath)
	return ioutil.WriteFile(fullpath, profile, 0644)
}

func intToString(i int64) string {
	return strconv.FormatInt(i, 10)
}
