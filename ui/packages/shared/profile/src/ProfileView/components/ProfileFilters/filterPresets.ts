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
  detect?: (mappingFiles: string[], functionNames: string[]) => boolean;
}

export const filterPresets: FilterPreset[] = [
  {
    key: 'go_runtime_expected_off_cpu',
    name: 'Go Runtime Expected Off-CPU',
    description: 'Excludes expected Go runtime blocking functions',
    allowedProfileTypes: ['parca_agent:wallclock:nanoseconds:samples:count:delta'],
    detect: (_mappingFiles, functionNames) =>
      functionNames.some(fn => fn === 'runtime.usleep' || fn === 'runtime.futex'),
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
    detect: (_mappingFiles, functionNames) =>
      functionNames.some(fn => fn.includes('parking_lot_core::') || fn.includes('tokio::runtime')),
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
    detect: mappingFiles => mappingFiles.some(f => f.includes('node')),
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
    detect: mappingFiles =>
      mappingFiles.some(f =>
        ['libcudnn', 'libcupti', 'libcudart', 'libcuda', 'libcublas', 'libparcagpucupti'].some(c =>
          f.includes(c)
        )
      ),
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
      {
        type: 'frame',
        field: 'binary',
        matchType: 'not_starts_with',
        value: 'libcublas',
      },
    ],
  },
  {
    key: 'hide_python_internals',
    name: 'Hide Python Internals',
    description: 'Excludes Python interpreter internal functions from the profile',
    detect: mappingFiles => mappingFiles.some(f => f.includes('python3')),
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
    detect: mappingFiles => mappingFiles.some(f => f.includes('libc.so')),
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
    detect: (_mappingFiles, functionNames) =>
      functionNames.some(fn => fn.startsWith('tokio::') || fn.startsWith('<tokio::')),
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
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_contains',
        value: 'tokio::runtime',
      },
    ],
  },
  {
    key: 'hide_rust_futures',
    name: 'Hide Rust Futures Infrastructure',
    description: 'Excludes Rust futures infrastructure frames from the profile',
    detect: (_mappingFiles, functionNames) =>
      functionNames.some(
        fn => fn.includes('futures_core') || fn.includes('core::future::future::Future')
      ),
    filters: [
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'future',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '<future',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_contains',
        value: 'futures_core',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_contains',
        value: 'core::future::future::Future',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'poll',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'try_poll',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '{closure#',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '{async_block#',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'invoke',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_equal',
        value: 'run',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '{async_fn#',
      },
    ],
  },
  {
    key: 'hide_rust_panic_backtrace',
    name: 'Hide Rust Panic Backtrace Infrastructure',
    description: 'Excludes Rust panic and backtrace infrastructure frames from the profile',
    detect: (_mappingFiles, functionNames) =>
      functionNames.some(
        fn =>
          fn.startsWith('std::panic') ||
          fn.startsWith('<core::panic') ||
          fn.startsWith('std::sys::backtrace')
      ),
    filters: [
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'std::panic',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: '<core::panic',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'std::sys::backtrace',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'catch_unwind<',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'call_once<',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'do_call<',
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

export const getPresetsForProfileType = (
  profileType?: string,
  mappingFiles?: string[],
  functionNames?: string[]
): FilterPreset[] => {
  return filterPresets.filter(preset => {
    if (
      preset.allowedProfileTypes !== undefined &&
      profileType !== undefined &&
      profileType !== '' &&
      !preset.allowedProfileTypes.includes(profileType)
    ) {
      return false;
    }
    if (preset.detect === undefined) return true;
    if (mappingFiles === undefined || functionNames === undefined) return true;
    return preset.detect(mappingFiles, functionNames);
  });
};
