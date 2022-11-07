// Copyright 2022 The Parca Authors
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

import {runBenchmark} from './util';
import parca1mProps from './data/parca-1m-profile-props';
import parca1hProps from './data/parca-1h-profile-props';
import ProfileIcicleGraph from '../../src/ProfileIcicleGraph';

describe('Benchmark 1h', () => {
  test('Flamegraph for 1h parca profile', async () => {
    const results1h = await runBenchmark({
      component: ProfileIcicleGraph,
      props: parca1hProps,
    });
    console.table({flamegraph1h: results1h}, ['min', 'max', 'mean', 'sampleCount', 'p70', 'p95']);
  });
});

describe('Benchmark 1m', () => {
  test('Flamegraph for 1m parca profile', async () => {
    const results1m = await runBenchmark({
      component: ProfileIcicleGraph,
      props: parca1mProps,
    });
    console.table({flamegraph1m: results1m}, ['min', 'max', 'mean', 'sampleCount', 'p70', 'p95']);
  });
});
