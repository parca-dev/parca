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

package e2econprof

import (
	"os/exec"
	"testing"

	"github.com/cortexproject/cortex/integration/e2e"

	"github.com/thanos-io/thanos/pkg/testutil"
)

func CleanScenario(t *testing.T, s *e2e.Scenario) func() {
	return func() {
		// Make sure Clean can properly delete everything.
		testutil.Ok(t, exec.Command("chmod", "-R", "777", s.SharedDir()).Run())
		s.Close()
	}
}
