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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/pkg/labels"
)

type Storage interface {
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
	s.logger.Log("msg", "writing profile", "path", fullpath)
	return ioutil.WriteFile(fullpath, profile, 0644)
}

func intToString(i int64) string {
	return strconv.FormatInt(i, 10)
}
