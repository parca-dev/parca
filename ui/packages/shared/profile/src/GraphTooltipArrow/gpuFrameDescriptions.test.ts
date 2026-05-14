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

import {gpuFrameDescription} from './gpuFrameDescriptions';

describe('gpuFrameDescription', () => {
  test.each([
    ['ISETP'],
    ['IMAD'],
    ['MOV'],
    ['FFMA'],
    ['LDG'],
    ['BRA'],
  ])('returns a description for SASS mnemonic %s', mnemonic => {
    const desc = gpuFrameDescription(mnemonic);
    expect(desc).toBeDefined();
    expect(desc?.length).toBeGreaterThan(0);
  });

  test.each([
    ['smsp__pcsamp_warps_issue_stalled_long_scoreboard'],
    ['smsp__pcsamp_warps_issue_stalled_short_scoreboard'],
    ['smsp__pcsamp_warps_issue_stalled_barrier'],
    ['smsp__pcsamp_warps_issue_stalled_drain'],
  ])('returns a description for stall reason %s', reason => {
    const desc = gpuFrameDescription(reason);
    expect(desc).toBeDefined();
    expect(desc?.length).toBeGreaterThan(0);
  });

  test.each([
    ['main'],
    ['at::native::add'],
    ['c10::impl::wrap_kernel_functor_unboxed_'],
    ['<unknown>'],
    [''],
  ])('returns undefined for non-GPU frame name %j', name => {
    expect(gpuFrameDescription(name)).toBeUndefined();
  });
});
