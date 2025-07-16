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

import {useURLStateCustom, type ParamValueSetterCustom} from '@parca/components';
import {type ProfileFilter} from '@parca/store';

// Compact encoding mappings
const TYPE_MAP: Record<string, string> = {
  stack: 's',
  frame: 'f',
};

const FIELD_MAP: Record<string, string> = {
  function_name: 'fn',
  binary: 'b',
  system_name: 'sn',
  filename: 'f',
  address: 'a',
  line_number: 'ln',
};

const MATCH_MAP: Record<string, string> = {
  equal: '=',
  not_equal: '!=',
  contains: '~',
  not_contains: '!~',
};

// Reverse mappings for decoding
const TYPE_MAP_REVERSE = Object.fromEntries(Object.entries(TYPE_MAP).map(([k, v]) => [v, k]));
const FIELD_MAP_REVERSE = Object.fromEntries(Object.entries(FIELD_MAP).map(([k, v]) => [v, k]));
const MATCH_MAP_REVERSE = Object.fromEntries(Object.entries(MATCH_MAP).map(([k, v]) => [v, k]));

// Encode filters to compact string format
const encodeProfileFilters = (filters: ProfileFilter[]): string => {
  if (filters.length === 0) return '';

  return filters
    .filter(f => f.value !== '' && f.type != null && f.field != null && f.matchType != null)
    .map(f => {
      const type = TYPE_MAP[f.type!];
      const field = FIELD_MAP[f.field!];
      const match = MATCH_MAP[f.matchType!];
      const value = encodeURIComponent(f.value);
      return `${type}:${field}:${match}:${value}`;
    })
    .join(',');
};

// Decode filters from compact string format
export const decodeProfileFilters = (encoded: string): ProfileFilter[] => {
  if (encoded === '' || encoded === undefined) return [];

  try {
    return encoded.split(',').map((filter, index) => {
      const [type, field, match, ...valueParts] = filter.split(':');
      const value = decodeURIComponent(valueParts.join(':')); // Handle values with colons

      return {
        id: `filter-${Date.now()}-${index}`,
        type: TYPE_MAP_REVERSE[type] as ProfileFilter['type'],
        field: FIELD_MAP_REVERSE[field] as ProfileFilter['field'],
        matchType: MATCH_MAP_REVERSE[match] as ProfileFilter['matchType'],
        value,
      };
    });
  } catch {
    return [];
  }
};

export const useProfileFiltersUrlState = (): {
  appliedFilters: ProfileFilter[];
  setAppliedFilters: ParamValueSetterCustom<ProfileFilter[]>;
} => {
  // Store applied filters in URL state for persistence using compact encoding
  const [appliedFilters, setAppliedFilters] = useURLStateCustom<ProfileFilter[]>(
    'profile_filters',
    {
      parse: value => {
        return decodeProfileFilters(value as string);
      },
      stringify: value => {
        return encodeProfileFilters(value);
      },
    }
  );

  return {
    appliedFilters,
    setAppliedFilters,
  };
};
