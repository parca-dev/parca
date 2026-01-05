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

import {useCallback, useMemo} from 'react';

import {useURLStateBatch, useURLStateCustom, type ParamValueSetterCustom} from '@parca/components';
import {safeDecode} from '@parca/utilities';

import {isPresetKey} from './filterPresets';
import {type ProfileFilter} from './useProfileFilters';

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
  starts_with: '^',
  not_starts_with: '!^',
};

// Reverse mappings for decoding
const TYPE_MAP_REVERSE = Object.fromEntries(Object.entries(TYPE_MAP).map(([k, v]) => [v, k]));
const FIELD_MAP_REVERSE = Object.fromEntries(Object.entries(FIELD_MAP).map(([k, v]) => [v, k]));
const MATCH_MAP_REVERSE = Object.fromEntries(Object.entries(MATCH_MAP).map(([k, v]) => [v, k]));

// Encode filters to compact string format
const encodeProfileFilters = (filters: ProfileFilter[]): string => {
  if (filters.length === 0) return '';

  return filters
    .filter(f => f.value !== '' && f.type != null)
    .map(f => {
      // Handle preset filters differently
      if (isPresetKey(f.type!)) {
        const presetKey = encodeURIComponent(f.type!);
        const value = encodeURIComponent(f.value);
        return `p:${presetKey}:${value}`;
      }

      // Handle regular filters
      const type = TYPE_MAP[f.type!];
      const field = FIELD_MAP[f.field!];
      const match = MATCH_MAP[f.matchType!];
      const value = encodeURIComponent(f.value);
      return `${type}:${field}:${match}:${value}`;
    })
    .join(',');
};

const generateFilterId = (
  filter: {type?: string; field?: string; matchType?: string; value: string},
  index: number
): string => {
  const parts = [
    filter.type ?? '',
    filter.field ?? '',
    filter.matchType ?? '',
    filter.value,
    index.toString(),
  ];
  return `filter-${parts.join('-').replace(/[^a-zA-Z0-9-]/g, '_')}`;
};

// Decode filters from compact string format
export const decodeProfileFilters = (encoded: string): ProfileFilter[] => {
  if (encoded === '' || encoded === undefined) return [];

  try {
    // Decode once since we fixed the double encoding issue
    const decodedString = safeDecode(encoded);

    return decodedString.split(',').map((filter, index) => {
      const parts = filter.split(':');

      // Handle preset filters (format: p:presetKey:value)
      if (parts[0] === 'p' && parts.length >= 3) {
        const presetKey = parts[1];
        const value = parts.slice(2).join(':'); // Handle values with colons

        const filterData = {type: presetKey, value};
        return {
          id: generateFilterId(filterData, index),
          type: presetKey,
          value,
        };
      }

      // Handle regular filters (format: type:field:match:value)
      const [type, field, match, ...valueParts] = parts;
      const value = valueParts.join(':'); // Handle values with colons

      const decodedFilter = {
        id: generateFilterId(
          {
            type: TYPE_MAP_REVERSE[type],
            field: FIELD_MAP_REVERSE[field],
            matchType: MATCH_MAP_REVERSE[match],
            value,
          },
          index
        ),
        type: TYPE_MAP_REVERSE[type] as ProfileFilter['type'],
        field: FIELD_MAP_REVERSE[field] as ProfileFilter['field'],
        matchType: MATCH_MAP_REVERSE[match] as ProfileFilter['matchType'],
        value,
      };

      return decodedFilter;
    });
  } catch {
    return [];
  }
};

interface UseProfileFiltersUrlStateOptions {
  viewDefaults?: ProfileFilter[];
}

export const useProfileFiltersUrlState = (
  options: UseProfileFiltersUrlStateOptions = {}
): {
  appliedFilters: ProfileFilter[];
  setAppliedFilters: ParamValueSetterCustom<ProfileFilter[]>;
  applyViewDefaults: () => void;
  forceApplyFilters: (filters: ProfileFilter[]) => void;
} => {
  const {viewDefaults} = options;

  const batchUpdates = useURLStateBatch();

  // Store applied filters in URL state for persistence using compact encoding
  const [appliedFilters, setAppliedFilters] = useURLStateCustom<ProfileFilter[]>(
    `profile_filters`,
    {
      parse: value => {
        return decodeProfileFilters(value as string);
      },
      stringify: value => {
        return encodeProfileFilters(value);
      },
      defaultValue: [],
    }
  );

  // Setter with preserve-existing strategy for applying view defaults
  const [, setAppliedFiltersWithPreserve] = useURLStateCustom<ProfileFilter[]>(`profile_filters`, {
    parse: value => {
      const result = decodeProfileFilters(value as string);
      return result;
    },
    stringify: value => {
      const result = encodeProfileFilters(value);
      return result;
    },
    mergeStrategy: 'preserve-existing',
  });

  const memoizedAppliedFilters = useMemo(() => {
    return appliedFilters ?? [];
  }, [appliedFilters]);

  // Apply view defaults (only if URL is empty)
  const applyViewDefaults = useCallback(() => {
    if (viewDefaults === undefined || viewDefaults.length === 0) {
      return;
    }

    batchUpdates(() => {
      setAppliedFiltersWithPreserve(viewDefaults);
    });
  }, [viewDefaults, batchUpdates, setAppliedFiltersWithPreserve]);

  // Force apply filters (bypasses preserve-existing strategy)
  // This validates filters before applying, similar to onApplyFilters in useProfileFilters.
  // Use this when switching views to completely replace the current filters.
  const forceApplyFilters = useCallback(
    (filters: ProfileFilter[]) => {
      const validFilters = filters.filter(f => {
        if (f.type != null && isPresetKey(f.type)) {
          return f.value !== '' && f.type != null;
        }
        return f.value !== '' && f.type != null && f.field != null && f.matchType != null;
      });

      batchUpdates(() => {
        setAppliedFilters(validFilters);
      });
    },
    [batchUpdates, setAppliedFilters]
  );

  return {
    appliedFilters: memoizedAppliedFilters,
    setAppliedFilters,
    applyViewDefaults,
    forceApplyFilters,
  };
};
