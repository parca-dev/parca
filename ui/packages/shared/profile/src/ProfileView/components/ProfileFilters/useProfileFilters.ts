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

import {useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {type Filter} from '@parca/client';

import {getPresetByKey, isPresetKey} from './filterPresets';
import {useProfileFiltersUrlState} from './useProfileFiltersUrlState';

export interface ProfileFilter {
  id: string;
  type?: 'stack' | 'frame' | string; // string allows preset keys
  field?: 'function_name' | 'binary' | 'system_name' | 'filename' | 'address' | 'line_number';
  matchType?: 'equal' | 'not_equal' | 'contains' | 'not_contains';
  value: string;
}

// Convert ProfileFilter[] to protobuf Filter[] matching the expected structure
export const convertToProtoFilters = (profileFilters: ProfileFilter[]): Filter[] => {
  // First, expand any preset filters to their constituent filters
  const expandedFilters: ProfileFilter[] = [];

  for (const filter of profileFilters) {
    if (filter.type != null && isPresetKey(filter.type)) {
      // This is a preset filter, expand it
      const preset = getPresetByKey(filter.type);
      if (preset != null) {
        preset.filters.forEach((presetFilter, index) => {
          expandedFilters.push({
            ...presetFilter,
            id: `${filter.id}-expanded-${index}`,
            value: presetFilter.value,
          });
        });
      }
    } else {
      // Regular filter, add as is
      expandedFilters.push(filter);
    }
  }

  return expandedFilters
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
} => {
  const {appliedFilters, setAppliedFilters} = useProfileFiltersUrlState();

  const [localFilters, setLocalFilters] = useState<ProfileFilter[]>(() => {
    // Initialize local filters with applied filters on first render
    return appliedFilters ?? [];
  });

  const lastAppliedFiltersRef = useRef<ProfileFilter[]>([]);

  const localFiltersRef = useRef<ProfileFilter[]>(localFilters);
  localFiltersRef.current = localFilters;

  useEffect(() => {
    const currentApplied = appliedFilters ?? [];
    const lastApplied = lastAppliedFiltersRef.current;

    // Check if appliedFilters actually changed (avoid circular updates)
    const appliedChanged =
      currentApplied.length !== lastApplied.length ||
      currentApplied.some((applied, index) => {
        const last = lastApplied[index];
        return (
          last == null ||
          applied.type !== last.type ||
          applied.field !== last.field ||
          applied.matchType !== last.matchType ||
          applied.value !== last.value
        );
      });

    if (!appliedChanged) {
      return;
    }

    lastAppliedFiltersRef.current = currentApplied;
    setLocalFilters(currentApplied);
  }, [appliedFilters]);

  const hasUnsavedChanges = useMemo(() => {
    const localWithValues = localFilters.filter(f => {
      // For preset filters, only need type and value
      if (f.type != null && isPresetKey(f.type)) {
        return f.value !== '' && f.type != null;
      }
      // For regular filters, need all fields
      return f.value !== '' && f.type != null && f.field != null && f.matchType != null;
    });

    const appliedWithValues = (appliedFilters ?? []).filter(f => {
      // For preset filters, only need type and value
      if (f.type != null && isPresetKey(f.type)) {
        return f.value !== '' && f.type != null;
      }
      // For regular filters, need all fields
      return f.value !== '' && f.type != null && f.field != null && f.matchType != null;
    });

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
    setLocalFilters([...localFiltersRef.current, newFilter]);
  }, []);

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
      setLocalFilters([...localFiltersRef.current, newFilter]);

      // Auto-apply the filter since it has a value
      const filtersToApply = [...(appliedFilters ?? []), newFilter];
      setAppliedFilters(filtersToApply);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [setAppliedFilters]
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
        setLocalFilters(localFiltersRef.current.filter(f => f.id !== filterToRemove.id));
      }
    },
    [appliedFilters, setAppliedFilters]
  );

  const removeFilter = useCallback((id: string) => {
    setLocalFilters(localFiltersRef.current.filter(f => f.id !== id));
  }, []);

  const updateFilter = useCallback((id: string, updates: Partial<ProfileFilter>) => {
    setLocalFilters(localFiltersRef.current.map(f => (f.id === id ? {...f, ...updates} : f)));
  }, []);

  const resetFilters = useCallback(() => {
    setLocalFilters([]);
    setAppliedFilters([]);
  }, [setAppliedFilters]);

  const onApplyFilters = useCallback((): void => {
    const validFilters = localFiltersRef.current.filter(f => {
      // For preset filters, only need type and value
      if (f.type != null && isPresetKey(f.type)) {
        return f.value !== '' && f.type != null;
      }
      // For regular filters, need all fields
      return f.value !== '' && f.type != null && f.field != null && f.matchType != null;
    });

    const filtersToApply = validFilters.map((f, index) => ({
      ...f,
      id: `filter-${Date.now()}-${index}`,
    }));

    setAppliedFilters(filtersToApply);
  }, [setAppliedFilters]);

  const protoFilters = useMemo(() => {
    return convertToProtoFilters(appliedFilters ?? []);
  }, [appliedFilters]);

  return {
    localFilters,
    appliedFilters: appliedFilters ?? [],
    protoFilters,
    hasUnsavedChanges,
    onApplyFilters,
    addFilter,
    excludeBinary,
    removeExcludeBinary,
    removeFilter,
    updateFilter,
    resetFilters,
  };
};
