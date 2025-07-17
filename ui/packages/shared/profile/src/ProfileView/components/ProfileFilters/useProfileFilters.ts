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

import {useCallback, useEffect, useMemo} from 'react';

import {type Filter} from '@parca/client';
import {
  selectLocalFilters,
  setLocalFilters,
  useAppDispatch,
  useAppSelector,
  type ProfileFilter,
} from '@parca/store';

import {useProfileFiltersUrlState} from './useProfileFiltersUrlState';
import {type FilterPreset} from './filterPresets';

export type {ProfileFilter};

// Convert ProfileFilter[] to protobuf Filter[] matching the expected structure
export const convertToProtoFilters = (profileFilters: ProfileFilter[]): Filter[] => {
  return profileFilters
    .filter(f => f.value !== '' && f.type != null && f.field != null && f.matchType != null) // Only include complete filters with values
    .map(f => {
      // Build the condition based on field type
      const isNumberField = f.field === 'address' || f.field === 'line_number';

      let condition: any;
      if (isNumberField) {
        const numValue = BigInt(f.value);
        condition = {
          condition:
            f.matchType === 'equal'
              ? {oneofKind: 'equal' as const, equal: numValue}
              : {oneofKind: 'notEqual' as const, notEqual: numValue},
        };
      } else {
        condition = {
          condition:
            f.matchType === 'equal'
              ? {oneofKind: 'equal' as const, equal: f.value}
              : f.matchType === 'not_equal'
              ? {oneofKind: 'notEqual' as const, notEqual: f.value}
              : f.matchType === 'contains'
              ? {oneofKind: 'contains' as const, contains: f.value}
              : {oneofKind: 'notContains' as const, notContains: f.value},
        };
      }

      // Create FilterCriteria
      const criteria: any = {};
      switch (f.field) {
        case 'function_name':
          criteria.functionName = condition;
          break;
        case 'binary':
          criteria.binary = condition;
          break;
        case 'system_name':
          criteria.systemName = condition;
          break;
        case 'filename':
          criteria.filename = condition;
          break;
        case 'address':
          criteria.address = condition;
          break;
        case 'line_number':
          criteria.lineNumber = condition;
          break;
      }

      // Create the appropriate filter type with proper oneofKind structure
      if (f.type === 'stack') {
        return {
          filter: {
            oneofKind: 'stackFilter' as const,
            stackFilter: {
              filter: {
                oneofKind: 'criteria' as const,
                criteria,
              },
            },
          },
        };
      } else {
        return {
          filter: {
            oneofKind: 'frameFilter' as const,
            frameFilter: {
              filter: {
                oneofKind: 'criteria' as const,
                criteria,
              },
            },
          },
        };
      }
    });
};

export const useProfileFilters = (): {
  localFilters: ProfileFilter[];
  appliedFilters: ProfileFilter[];
  protoFilters: Filter[];
  hasUnsavedChanges: boolean;
  onApplyFilters: () => void;
  addFilter: () => void;
  excludeBinary: (binaryName: string) => void;
  removeExcludeBinary: (binaryName: string) => void;
  removeFilter: (id: string) => void;
  updateFilter: (id: string, updates: Partial<ProfileFilter>) => void;
  resetFilters: () => void;
  applyPreset: (preset: FilterPreset) => void;
} => {
  const {appliedFilters, setAppliedFilters} = useProfileFiltersUrlState();
  const dispatch = useAppDispatch();
  const localFilters = useAppSelector(selectLocalFilters);

  useEffect(() => {
    if (appliedFilters != null && appliedFilters.length > 0) {
      // Check if they're different to avoid unnecessary updates
      const areFiltersEqual =
        appliedFilters.length === localFilters.length &&
        appliedFilters.every((applied, index) => {
          const local = localFilters[index];
          return (
            local != null &&
            applied.type === local.type &&
            applied.field === local.field &&
            applied.matchType === local.matchType &&
            applied.value === local.value
          );
        });

      if (!areFiltersEqual) {
        dispatch(setLocalFilters(appliedFilters));
      }
    } else if (appliedFilters != null && appliedFilters.length === 0 && localFilters.length > 0) {
      dispatch(setLocalFilters([]));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const hasUnsavedChanges = useMemo(() => {
    const localWithValues = localFilters.filter(f => f.value !== '');
    const appliedWithValues = (appliedFilters ?? []).filter(f => f.value !== '');

    if (localWithValues.length !== appliedWithValues.length) return true;

    return !localWithValues.every((local, index) => {
      const applied = appliedWithValues[index];
      return (
        local.type === applied?.type &&
        local.field === applied?.field &&
        local.matchType === applied?.matchType &&
        local.value === applied?.value
      );
    });
  }, [localFilters, appliedFilters]);

  const addFilter = useCallback(() => {
    const newFilter: ProfileFilter = {
      id: `filter-${Date.now()}-${Math.random()}`,
      value: '',
    };
    dispatch(setLocalFilters([...localFilters, newFilter]));
  }, [dispatch, localFilters]);

  const excludeBinary = useCallback(
    (binaryName: string) => {
      // Check if this binary is already being filtered with not_contains
      const existingFilter = (appliedFilters ?? []).find(
        f =>
          f.type === 'frame' &&
          f.field === 'binary' &&
          f.matchType === 'not_contains' &&
          f.value === binaryName
      );

      if (existingFilter != null) {
        return; // Already exists, don't add duplicate
      }

      const newFilter: ProfileFilter = {
        id: `filter-${Date.now()}-${Math.random()}`,
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: binaryName,
      };
      dispatch(setLocalFilters([...localFilters, newFilter]));

      // Auto-apply the filter since it has a value
      const filtersToApply = [...(appliedFilters ?? []), newFilter];
      setAppliedFilters(filtersToApply);
    },
    [appliedFilters, setAppliedFilters, dispatch, localFilters]
  );

  const removeExcludeBinary = useCallback(
    (binaryName: string) => {
      // Search for the exclude filter (not_contains) for this binary
      const filterToRemove = (appliedFilters ?? []).find(
        f =>
          f.type === 'frame' &&
          f.field === 'binary' &&
          f.matchType === 'not_contains' &&
          f.value === binaryName
      );

      if (filterToRemove != null) {
        // Remove the filter from applied filters
        const updatedAppliedFilters = (appliedFilters ?? []).filter(
          f => f.id !== filterToRemove.id
        );
        setAppliedFilters(updatedAppliedFilters);

        // Also remove from local filters
        const updatedLocalFilters = localFilters.filter(f => f.id !== filterToRemove.id);
        dispatch(setLocalFilters(updatedLocalFilters));
      }
    },
    [appliedFilters, setAppliedFilters, dispatch, localFilters]
  );

  const removeFilter = useCallback(
    (id: string) => {
      dispatch(setLocalFilters(localFilters.filter(f => f.id !== id)));
    },
    [dispatch, localFilters]
  );

  const updateFilter = useCallback(
    (id: string, updates: Partial<ProfileFilter>) => {
      dispatch(setLocalFilters(localFilters.map(f => (f.id === id ? {...f, ...updates} : f))));
    },
    [dispatch, localFilters]
  );

  const resetFilters = useCallback(() => {
    dispatch(setLocalFilters([]));
    setAppliedFilters([]);
  }, [dispatch, setAppliedFilters]);

  const onApplyFilters = useCallback((): void => {
    const validFilters = localFilters.filter(
      f => f.value !== '' && f.type != null && f.field != null && f.matchType != null
    );

    const filtersToApply = validFilters.map((f, index) => ({
      ...f,
      id: `filter-${Date.now()}-${index}`,
    }));

    setAppliedFilters(filtersToApply);
  }, [localFilters, setAppliedFilters]);

  const protoFilters = useMemo(() => {
    return convertToProtoFilters(appliedFilters ?? []);
  }, [appliedFilters]);

  const applyPreset = useCallback(
    (preset: FilterPreset) => {
      // Create new filters from the preset with unique IDs
      const presetFilters: ProfileFilter[] = preset.filters.map((filter, index) => ({
        ...filter,
        id: `filter-preset-${Date.now()}-${index}`,
      }));

      // Set local filters to the preset filters
      dispatch(setLocalFilters(presetFilters));

      // Immediately apply the filters
      setAppliedFilters(presetFilters);
    },
    [dispatch, setAppliedFilters]
  );

  return {
    localFilters,
    appliedFilters,
    protoFilters,
    hasUnsavedChanges,
    onApplyFilters,
    addFilter,
    excludeBinary,
    removeExcludeBinary,
    removeFilter,
    updateFilter,
    resetFilters,
    applyPreset,
  };
};
