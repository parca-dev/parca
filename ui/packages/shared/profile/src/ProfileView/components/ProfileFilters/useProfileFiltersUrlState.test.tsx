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

import {type ReactNode} from 'react';

// eslint-disable-next-line import/named
import {act, renderHook, waitFor} from '@testing-library/react';
import {NuqsTestingAdapter, type OnUrlUpdateFunction} from 'nuqs/adapters/testing';
import {describe, expect, it, vi} from 'vitest';

import {type ProfileFilter} from './useProfileFilters';
import {decodeProfileFilters, useProfileFiltersUrlState} from './useProfileFiltersUrlState';

// Helper to create wrapper with NuqsTestingAdapter
const createWrapper = (
  searchParams: string | Record<string, string> = {},
  onUrlUpdate?: OnUrlUpdateFunction
): (({children}: {children: ReactNode}) => JSX.Element) => {
  const Wrapper = ({children}: {children: ReactNode}): JSX.Element => (
    <NuqsTestingAdapter searchParams={searchParams} onUrlUpdate={onUrlUpdate} hasMemory={true}>
      {children}
    </NuqsTestingAdapter>
  );
  Wrapper.displayName = 'NuqsTestingWrapper';
  return Wrapper;
};

describe('useProfileFiltersUrlState', () => {
  describe('decodeProfileFilters', () => {
    it('should return empty array for empty string', () => {
      expect(decodeProfileFilters('')).toEqual([]);
    });

    it('should return empty array for undefined', () => {
      expect(decodeProfileFilters(undefined as unknown as string)).toEqual([]);
    });

    it('should decode stack filter with function_name', () => {
      // Format: type:field:match:value -> s:fn:=:testFunc
      const encoded = 's:fn:=:testFunc';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'stack',
        field: 'function_name',
        matchType: 'equal',
        value: 'testFunc',
      });
    });

    it('should decode frame filter with binary', () => {
      const encoded = 'f:b:!=:libc.so';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'frame',
        field: 'binary',
        matchType: 'not_equal',
        value: 'libc.so',
      });
    });

    it('should decode filter with contains match', () => {
      const encoded = 's:fn:~:runtime';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'stack',
        field: 'function_name',
        matchType: 'contains',
        value: 'runtime',
      });
    });

    it('should decode filter with not_contains match', () => {
      const encoded = 'f:b:!~:node';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'frame',
        field: 'binary',
        matchType: 'not_contains',
        value: 'node',
      });
    });

    it('should decode filter with starts_with match', () => {
      const encoded = 's:fn:^:std::';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'stack',
        field: 'function_name',
        matchType: 'starts_with',
        value: 'std::',
      });
    });

    it('should decode filter with not_starts_with match', () => {
      const encoded = 'f:fn:!^:tokio::';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'frame',
        field: 'function_name',
        matchType: 'not_starts_with',
        value: 'tokio::',
      });
    });

    it('should decode multiple filters', () => {
      const encoded = 's:fn:=:testFunc,f:b:!=:libc.so';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(2);
      expect(result[0]).toMatchObject({
        type: 'stack',
        field: 'function_name',
        matchType: 'equal',
        value: 'testFunc',
      });
      expect(result[1]).toMatchObject({
        type: 'frame',
        field: 'binary',
        matchType: 'not_equal',
        value: 'libc.so',
      });
    });

    it('should decode preset filter', () => {
      const encoded = 'p:hide_libc:enabled';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'hide_libc',
        value: 'enabled',
      });
    });

    it('should handle values with colons', () => {
      const encoded = 'p:some_preset:value:with:colons';
      const result = decodeProfileFilters(encoded);

      expect(result).toHaveLength(1);
      expect(result[0]).toMatchObject({
        type: 'some_preset',
        value: 'value:with:colons',
      });
    });

    it('should decode all field types', () => {
      const testCases = [
        {encoded: 's:fn:=:test', expectedField: 'function_name'},
        {encoded: 's:b:=:test', expectedField: 'binary'},
        {encoded: 's:sn:=:test', expectedField: 'system_name'},
        {encoded: 's:f:=:test', expectedField: 'filename'},
        {encoded: 's:a:=:test', expectedField: 'address'},
        {encoded: 's:ln:=:test', expectedField: 'line_number'},
      ];

      for (const {encoded, expectedField} of testCases) {
        const result = decodeProfileFilters(encoded);
        expect(result[0].field).toBe(expectedField);
      }
    });

    it('should return empty array for malformed input', () => {
      // This should not throw - it returns empty array on error
      expect(() => decodeProfileFilters('malformed')).not.toThrow();
    });

    it('should generate unique IDs for each filter', () => {
      const encoded = 's:fn:=:func1,s:fn:=:func2,s:fn:=:func3';
      const result = decodeProfileFilters(encoded);

      const ids = result.map(f => f.id);
      const uniqueIds = new Set(ids);
      expect(uniqueIds.size).toBe(ids.length);
    });
  });

  describe('Basic functionality', () => {
    it('should initialize with empty filters when no URL params', () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      expect(result.current.appliedFilters).toEqual([]);
    });

    it('should read filters from URL', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:testFunc'}),
      });

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0]).toMatchObject({
          type: 'stack',
          field: 'function_name',
          matchType: 'equal',
          value: 'testFunc',
        });
      });
    });

    it('should update URL when setting filters', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({}, onUrlUpdate),
      });

      const newFilters: ProfileFilter[] = [
        {
          id: 'test-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'libc.so',
        },
      ];

      act(() => {
        result.current.setAppliedFilters(newFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.get('profile_filters')).toBe('f:b:!~:libc.so');
      });
    });

    it('should clear URL param when setting empty filters', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:testFunc'}, onUrlUpdate),
      });

      act(() => {
        result.current.setAppliedFilters([]);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.has('profile_filters')).toBe(false);
      });
    });
  });

  describe('forceApplyFilters', () => {
    it('should provide forceApplyFilters method', () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      expect(typeof result.current.forceApplyFilters).toBe('function');
    });

    it('should force apply filters overwriting existing', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:existingFunc'}, onUrlUpdate),
      });

      // Verify existing filter is loaded
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
      });

      const newFilters: ProfileFilter[] = [
        {
          id: 'forced-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'forcedValue',
        },
      ];

      act(() => {
        result.current.forceApplyFilters(newFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.get('profile_filters')).toBe('f:b:!~:forcedValue');
      });
    });

    it('should clear filters when force applying empty array', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:existingFunc'}, onUrlUpdate),
      });

      act(() => {
        result.current.forceApplyFilters([]);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.has('profile_filters')).toBe(false);
      });
    });
  });

  describe('Preset filter encoding', () => {
    it('should encode preset filters correctly', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({}, onUrlUpdate),
      });

      const presetFilters: ProfileFilter[] = [
        {
          id: 'preset-1',
          type: 'hide_libc',
          value: 'enabled',
        },
      ];

      act(() => {
        result.current.setAppliedFilters(presetFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.get('profile_filters')).toBe('p:hide_libc:enabled');
      });
    });

    it('should handle mixed preset and regular filters', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({}, onUrlUpdate),
      });

      const mixedFilters: ProfileFilter[] = [
        {
          id: 'preset-1',
          type: 'hide_libc',
          value: 'enabled',
        },
        {
          id: 'regular-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'node',
        },
      ];

      act(() => {
        result.current.setAppliedFilters(mixedFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        expect(lastCall.searchParams.get('profile_filters')).toBe(
          'p:hide_libc:enabled,f:b:!~:node'
        );
      });
    });
  });

  describe('URL encoding edge cases', () => {
    it('should handle special characters in filter values', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({}, onUrlUpdate),
      });

      const filtersWithSpecialChars: ProfileFilter[] = [
        {
          id: 'special-1',
          type: 'stack',
          field: 'function_name',
          matchType: 'contains',
          value: 'std::vector<int>',
        },
      ];

      act(() => {
        result.current.setAppliedFilters(filtersWithSpecialChars);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        const filterValue = lastCall.searchParams.get('profile_filters');
        // The value should contain the encoded special characters
        expect(filterValue).toContain('std%3A%3Avector%3Cint%3E');
      });
    });

    it('should filter out incomplete filters when encoding', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({}, onUrlUpdate),
      });

      const incompleteFilters: ProfileFilter[] = [
        {
          id: 'complete-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'valid',
        },
        {
          id: 'incomplete-1',
          type: 'frame',
          // Missing field, matchType
          value: '',
        },
        {
          id: 'incomplete-2',
          type: undefined,
          value: 'value',
        },
      ];

      act(() => {
        result.current.setAppliedFilters(incompleteFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        // Only the complete filter should be encoded
        expect(lastCall.searchParams.get('profile_filters')).toBe('f:b:!~:valid');
      });
    });
  });

  describe('Memoization', () => {
    it('should return empty array with consistent structure when no filters', () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      // Empty filters should be an empty array (not undefined or null)
      expect(Array.isArray(result.current.appliedFilters)).toBe(true);
      expect(result.current.appliedFilters).toHaveLength(0);
    });

    it('should always return array (never undefined)', () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      expect(Array.isArray(result.current.appliedFilters)).toBe(true);
      expect(result.current.appliedFilters).toEqual([]);
    });

    it('should return correctly structured filters from URL', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:testFunc'}),
      });

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
      });

      // Verify the filter structure is correct
      const filter = result.current.appliedFilters[0];
      expect(filter).toHaveProperty('id');
      expect(filter).toHaveProperty('type', 'stack');
      expect(filter).toHaveProperty('field', 'function_name');
      expect(filter).toHaveProperty('matchType', 'equal');
      // eslint-disable-next-line jest-dom/prefer-to-have-value
      expect(filter).toHaveProperty('value', 'testFunc');
    });
  });

  describe('View switching scenarios', () => {
    it('should completely replace filters when switching views using forceApplyFilters', async () => {
      const onUrlUpdate = vi.fn();
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper(
          {profile_filters: 's:fn:=:viewAFunc,f:b:!=:viewABinary'},
          onUrlUpdate
        ),
      });

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(2);
      });

      // Switch to View B (completely different filter)
      const viewBFilters: ProfileFilter[] = [
        {
          id: 'viewB-1',
          type: 'frame',
          field: 'function_name',
          matchType: 'contains',
          value: 'viewBOnly',
        },
      ];

      act(() => {
        result.current.forceApplyFilters(viewBFilters);
      });

      await waitFor(() => {
        expect(onUrlUpdate).toHaveBeenCalled();
        const lastCall = onUrlUpdate.mock.calls[onUrlUpdate.mock.calls.length - 1][0];
        const filterValue = lastCall.searchParams.get('profile_filters');

        // View A's filters should be completely gone
        expect(filterValue).not.toContain('viewAFunc');
        expect(filterValue).not.toContain('viewABinary');

        // Only View B's filter should be present
        expect(filterValue).toBe('f:fn:~:viewBOnly');
      });
    });

    it('should not change filters when clicking the same view tab', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {
        wrapper: createWrapper({profile_filters: 's:fn:=:existingFilter'}),
      });

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
      });

      // Apply the same filters (simulating clicking the same view tab)
      const sameFilters: ProfileFilter[] = [
        {
          id: 'same-1',
          type: 'stack',
          field: 'function_name',
          matchType: 'equal',
          value: 'existingFilter',
        },
      ];

      act(() => {
        result.current.forceApplyFilters(sameFilters);
      });

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('existingFilter');
      });
    });
  });
});
