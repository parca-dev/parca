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

import type {ProfileFilter} from './useProfileFilters';

export interface FilterPreset {
  key: string;
  name: string;
  description: string;
  filters: Array<Omit<ProfileFilter, 'id'>>;
  allowedProfileTypes?: string[];
}

export const filterPresets: FilterPreset[] = [
  {
    key: 'go_runtime_expected_off_cpu',
    name: 'Go Runtime Expected Off-CPU',
    description: 'Excludes expected Go runtime blocking functions',
    allowedProfileTypes: ['parca_agent:wallclock:nanoseconds:samples:count:delta'],
    filters: [
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'runtime.usleep',
      },
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'runtime.futex',
      },
    ],
  },
  {
    key: 'rust_runtime_expected_off_cpu',
    name: 'Rust Expected Off-CPU',
    description: 'Excludes expected Rust runtime blocking functions',
    allowedProfileTypes: ['parca_agent:wallclock:nanoseconds:samples:count:delta'],
    filters: [
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'parking_lot_core::thread_parker::imp::ThreadParker::futex_wait',
      },
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'tokio::runtime::time::Driver::park_internal',
      },
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'futex_wait',
      },
    ],
  },
  {
    key: 'hide_v8_internals',
    name: 'Hide V8 internals',
    description: 'Excludes Node.js and V8 internal functions from the profile',
    filters: [
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'node',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_contains',
        value: 'V8',
      },
    ],
  },
  {
    key: 'hide_cuda_internals',
    name: 'Hide CUDA Internals',
    description: 'Excludes CUDA and NVIDIA GPU driver internal functions from the profile',
    filters: [
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcudnn_engines_precompiled.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcupti.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcudart.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcuda.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcudnn.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libcudnn_graph.so',
      },
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libparcagpucupti.so',
      },
    ],
  },
  {
    key: 'hide_python_internals',
    name: 'Hide Python Internals',
    description: 'Excludes Python interpreter internal functions from the profile',
    filters: [
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'python3',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_equal',
        value: '<interpreter trampoline>',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_equal',
        value: '<module>',
      },
    ],
  },
  {
    key: 'hide_libc',
    name: 'Hide libc',
    description: 'Excludes C standard library functions from the profile',
    filters: [
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'libc.so',
      },
    ],
  },
  {
    key: 'hide_tokio_frames',
    name: 'Hide Tokio Frames',
    description: 'Excludes Tokio runtime frames from the profile',
    filters: [
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'tokio::',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '<tokio::',
      },
    ],
  },
];

const presetKeys = new Set(filterPresets.map(preset => preset.key));

export const isPresetKey = (key: string): boolean => {
  return presetKeys.has(key);
};

export const getPresetByKey = (key: string): FilterPreset | undefined => {
  return filterPresets.find(preset => preset.key === key);
};

export const getPresetsForProfileType = (profileType?: string): FilterPreset[] => {
  if (profileType === undefined || profileType === '') return filterPresets;

  return filterPresets.filter(preset => {
    if (preset.allowedProfileTypes === undefined) return true;
    return preset.allowedProfileTypes.includes(profileType);
  });
};
