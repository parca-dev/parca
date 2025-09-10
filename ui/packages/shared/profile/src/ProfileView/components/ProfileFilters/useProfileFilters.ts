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

import {type Filter, type NumberCondition, type StringCondition} from '@parca/client';

import {
  detectPresets,
  filterPresets,
  getPresetByKey,
  getPresetsForProfileType,
  isPresetKey,
} from './filterPresets';
import {useProfileFiltersUrlState} from './useProfileFiltersUrlState';

export interface ProfileFilter {
  id: string;
  type?: 'stack' | 'frame' | string; // string allows preset keys
  field?: 'function_name' | 'binary' | 'system_name' | 'filename' | 'address' | 'line_number';
  matchType?: 'equal' | 'not_equal' | 'contains' | 'not_contains';
  value: string;
}

const createStringCondition = (matchType: string, value: string): StringCondition => ({
  condition:
    matchType === 'equal'
      ? {oneofKind: 'equal' as const, equal: value}
      : matchType === 'not_equal'
      ? {oneofKind: 'notEqual' as const, notEqual: value}
      : matchType === 'contains'
      ? {oneofKind: 'contains' as const, contains: value}
      : {oneofKind: 'notContains' as const, notContains: value},
});

const createNumberCondition = (matchType: string, value: bigint): NumberCondition => ({
  condition:
    matchType === 'equal'
      ? {oneofKind: 'equal' as const, equal: value}
      : {oneofKind: 'notEqual' as const, notEqual: value},
});

// Convert protobuf Filter[] back to ProfileFilter[] format for editing
export const convertFromProtoFilters = (
  protoFilters: Filter[],
  profileType?: string
): ProfileFilter[] => {
  const rawProfileFilters: ProfileFilter[] = [];

  for (const [index, protoFilter] of protoFilters.entries()) {
    if (protoFilter?.filter == null) continue;

    const filter = protoFilter.filter;
    let type: string;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let criteria: Record<string, any>;

    if (
      filter.oneofKind === 'stackFilter' &&
      filter.stackFilter?.filter?.oneofKind === 'criteria'
    ) {
      type = 'stack';
      criteria = filter.stackFilter.filter.criteria;
    } else if (
      filter.oneofKind === 'frameFilter' &&
      filter.frameFilter?.filter?.oneofKind === 'criteria'
    ) {
      type = 'frame';
      criteria = filter.frameFilter.filter.criteria;
    } else {
      continue;
    }

    for (const [fieldName, condition] of Object.entries(criteria)) {
      if (condition === undefined || typeof condition !== 'object') continue;

      const conditionObj = condition;
      if (conditionObj.condition?.oneofKind === undefined) continue;

      let matchType: string;
      let value: string;

      switch (conditionObj.condition.oneofKind) {
        case 'equal':
          matchType = 'equal';
          value = String(conditionObj.condition.equal);
          break;
        case 'notEqual':
          matchType = 'not_equal';
          value = String(conditionObj.condition.notEqual);
          break;
        case 'contains':
          matchType = 'contains';
          value = conditionObj.condition.contains ?? '';
          break;
        case 'notContains':
          matchType = 'not_contains';
          value = conditionObj.condition.notContains ?? '';
          break;
        default:
          continue;
      }

      const fieldMap: Record<string, string> = {
        functionName: 'function_name',
        binary: 'binary',
        systemName: 'system_name',
        filename: 'filename',
        address: 'address',
        lineNumber: 'line_number',
      };

      const field = fieldMap[fieldName] ?? fieldName;

      rawProfileFilters.push({
        id: `parsed-${index}-${fieldName}`,
        type: type as ProfileFilter['type'],
        field: field as ProfileFilter['field'],
        matchType: matchType as ProfileFilter['matchType'],
        value,
      });
    }
  }

  const filtersWithoutId = rawProfileFilters.map(filter => {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const {id, ...rest} = filter;
    return rest;
  });

  const {detectedPresets, remainingFilters} = detectPresets(filtersWithoutId);

  const finalFilters: ProfileFilter[] = [];

  const allowedPresets =
    profileType !== undefined ? getPresetsForProfileType(profileType) : filterPresets;
  const allowedPresetKeys = new Set(allowedPresets.map(p => p.key));

  detectedPresets.forEach((preset, index) => {
    if (allowedPresetKeys.has(preset.key)) {
      finalFilters.push({
        id: `preset-${preset.key}-${index}`,
        type: preset.key,
        value: preset.name,
      });
    } else {
      preset.filters.forEach((presetFilter, filterIndex) => {
        finalFilters.push({
          ...presetFilter,
          id: `fallback-${preset.key}-${filterIndex}`,
        });
      });
    }
  });

  remainingFilters.forEach((filter, index) => {
    finalFilters.push({
      ...filter,
      id: `remaining-${index}`,
    });
  });

  return finalFilters;
};

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

      const condition: StringCondition | NumberCondition = isNumberField
        ? createNumberCondition(f.matchType as string, BigInt(f.value))
        : createStringCondition(f.matchType as string, f.value);

      // Create FilterCriteria
      const criteria: Record<string, StringCondition | NumberCondition> = {};
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

  const [localFilters, setLocalFilters] = useState<ProfileFilter[]>(appliedFilters ?? []);

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
