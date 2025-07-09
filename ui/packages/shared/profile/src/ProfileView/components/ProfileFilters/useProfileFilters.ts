// Copyright 2025 The Parca Authors
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

import { useCallback, useMemo, useState } from 'react';
import { USER_PREFERENCES, useUserPreference } from '@parca/hooks';
import { useProfileFiltersUrlState, type ProfileFilter } from './useProfileFiltersUrlState';

// Re-export the ProfileFilter type for convenience
export type { ProfileFilter };

// Convert ProfileFilter[] to protobuf Filter[] matching the expected structure
const convertToProtoFilters = (profileFilters: ProfileFilter[]) => {
  return profileFilters
    .filter(f => f.value !== '') // Only include filters with values
    .map(f => {
      // Build the condition based on field type
      const isNumberField = f.field === 'address' || f.field === 'line_number';

      let condition: any;
      if (isNumberField) {
        const numValue = BigInt(f.value);
        condition = {
          condition: f.matchType === 'equal'
            ? { oneofKind: 'equal' as const, equal: numValue }
            : { oneofKind: 'notEqual' as const, notEqual: numValue }
        };
      } else {
        condition = {
          condition:
            f.matchType === 'equal' ? { oneofKind: 'equal' as const, equal: f.value } :
            f.matchType === 'not_equal' ? { oneofKind: 'notEqual' as const, notEqual: f.value } :
            f.matchType === 'contains' ? { oneofKind: 'contains' as const, contains: f.value } :
            { oneofKind: 'notContains' as const, notContains: f.value }
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
            stackFilter: { criteria }
          }
        };
      } else {
        return {
          filter: {
            oneofKind: 'frameFilter' as const,
            frameFilter: { criteria }
          }
        };
      }
    });
};

interface UseProfileFiltersProps {
  onFiltersChange?: (filters: ProfileFilter[]) => void;
}

export const useProfileFilters = ({ onFiltersChange }: UseProfileFiltersProps = {}) => {

  const { appliedFilters, setAppliedFilters } = useProfileFiltersUrlState();

  const [localFilters, setLocalFilters] = useState<ProfileFilter[]>(() => {
    return appliedFilters || [];
  });


  const hasUnsavedChanges = useMemo(() => {
    const localWithValues = localFilters.filter(f => f.value !== '');
    const appliedWithValues = (appliedFilters || []).filter(f => f.value !== '');

    if (localWithValues.length !== appliedWithValues.length) return true;

    return !localWithValues.every((local, index) => {
      const applied = appliedWithValues[index];
      return local.type === applied?.type &&
             local.field === applied?.field &&
             local.matchType === applied?.matchType &&
             local.value === applied?.value;
    });
  }, [localFilters, appliedFilters]);

  const isClearAction = useMemo(() => {
    const hasAppliedFilters = appliedFilters && appliedFilters.length > 0 && appliedFilters.some(f => f.value !== '');
    return hasAppliedFilters && !hasUnsavedChanges;
  }, [appliedFilters, hasUnsavedChanges]);

  const addFilter = useCallback(() => {
    const newFilter: ProfileFilter = {
      id: `filter-${Date.now()}-${Math.random()}`,
      type: 'stack',
      field: 'function_name',
      matchType: 'contains',
      value: '',
    };
    setLocalFilters(prev => [...prev, newFilter]);
  }, []);

  const removeFilter = useCallback((id: string) => {
    setLocalFilters(prev => prev.filter(f => f.id !== id));
  }, []);

  const updateFilter = useCallback((id: string, updates: Partial<ProfileFilter>) => {
    setLocalFilters(prev => prev.map(f => f.id === id ? { ...f, ...updates } : f));
  }, []);

  const resetFilters = useCallback(() => {
    setLocalFilters([]);
    setAppliedFilters([]);
    onFiltersChange?.([]);
  }, [setAppliedFilters, onFiltersChange]);

  const onApplyFilters = useCallback((): void => {
    if (isClearAction) {
      resetFilters();
    } else {
      const validFilters = localFilters.filter(f => f.value !== '');

      const filtersToApply = validFilters.map((f, index) => ({
        ...f,
        id: `filter-${Date.now()}-${index}`
      }));

      setAppliedFilters(filtersToApply);

      onFiltersChange?.(filtersToApply);
    }
  }, [
    localFilters,
    isClearAction,
    resetFilters,
    setAppliedFilters,
    onFiltersChange,
  ]);

  const protoFilters = useMemo(() => {
    return convertToProtoFilters(appliedFilters || []);
  }, [appliedFilters]);

  return {
    localFilters,
    appliedFilters,
    protoFilters,
    hasUnsavedChanges,
    isClearAction,
    onApplyFilters,
    addFilter,
    removeFilter,
    updateFilter,
    resetFilters,
  };
};
