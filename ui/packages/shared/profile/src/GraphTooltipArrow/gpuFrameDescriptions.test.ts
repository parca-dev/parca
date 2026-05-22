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

import {describe, expect, test} from 'vitest';

import {SASS_SOURCE_URL, STALL_SOURCE_URL, gpuFrameInfo} from './gpuFrameDescriptions';

describe('gpuFrameInfo', () => {
  test.each([
    ['STS', 'Store to Shared Memory'],
    ['ISETP', 'Integer Compare And Set Predicate'],
    ['IMAD', 'Integer Multiply And Add'],
    ['MOV', 'Move'],
    ['FFMA', 'FP32 Fused Multiply and Add'],
    ['LDG', 'Load from Global Memory'],
    ['LDCU', 'Load a Value from Constant Memory into a Uniform Register'],
    ['HGMMA', 'Matrix Multiply and Accumulate Across a Warpgroup'],
    ['UTMALDG', 'Tensor Load from Global to Shared Memory'],
    ['LDT', 'Load Matrix from Tensor Memory to Register File'],
  ])('returns SASS info for %s with verbatim description %j', (mnemonic, description) => {
    const info = gpuFrameInfo(mnemonic);
    expect(info?.kind).toBe('sass');
    expect(info?.entry.description).toBe(description);
    expect(info?.entry.reasonLabel.length).toBeGreaterThan(0);
    expect(info?.sourceUrl).toBe(SASS_SOURCE_URL);
  });

  test.each([
    ['smsp__pcsamp_warps_issue_stalled_long_scoreboard', 'Long Scoreboard'],
    ['smsp__pcsamp_warps_issue_stalled_short_scoreboard', 'Short Scoreboard'],
    ['smsp__pcsamp_warps_issue_stalled_barrier', 'Barrier'],
    ['smsp__pcsamp_warps_issue_stalled_drain', 'Drain'],
  ])('returns stall info for %s with reasonLabel %j and per-frame deep link', (reason, label) => {
    const info = gpuFrameInfo(reason);
    expect(info?.kind).toBe('stall');
    expect(info?.entry.description.length).toBeGreaterThan(0);
    expect(info?.entry.reasonLabel).toBe(label);
    expect(info?.sourceUrl).toBe(`${STALL_SOURCE_URL}:~:text=${reason}`);
  });

  test.each([['main'], ['at::native::add'], ['<unknown>'], ['']])(
    'returns undefined for non-GPU frame name %j',
    name => {
      expect(gpuFrameInfo(name)).toBeUndefined();
    }
  );
});
