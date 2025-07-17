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

import type {ProfileFilter} from '@parca/store';

export interface FilterPreset {
  key: string;
  name: string;
  description: string;
  filters: Array<Omit<ProfileFilter, 'id'>>;
}

export const filterPresets: FilterPreset[] = [
  {
    key: 'runtime_no_gc_drain',
    name: 'Runtime without GC drain frames',
    description: 'Shows runtime frames but excludes garbage collection drain frames',
    filters: [
      {
        type: 'stack',
        field: 'function_name',
        matchType: 'contains',
        value: 'runtime',
      },
      {
        type: 'frame',
        field: 'function_name',
        matchType: 'not_contains',
        value: 'gcDrain',
      },
    ],
  },
];

export const isPresetKey = (key: string): boolean => {
  return filterPresets.some(preset => preset.key === key);
};

export const getPresetByKey = (key: string): FilterPreset | undefined => {
  return filterPresets.find(preset => preset.key === key);
};
