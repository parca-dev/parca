// Copyright 2024-2026 The Parca Authors
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

package badgerlogger

import (
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type BadgerLogger struct {
	Logger log.Logger
}

func (l *BadgerLogger) Errorf(f string, v ...interface{}) {
	level.Error(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Warningf(f string, v ...interface{}) {
	level.Warn(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Infof(f string, v ...interface{}) {
	level.Info(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}

func (l *BadgerLogger) Debugf(f string, v ...interface{}) {
	level.Debug(l.Logger).Log("msg", fmt.Sprintf(f, v...))
}
