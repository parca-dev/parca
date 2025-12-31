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
import {beforeEach, describe, expect, it, vi} from 'vitest';

import {URLStateProvider} from '@parca/components';

import {type ProfileFilter} from './useProfileFilters';
import {decodeProfileFilters, useProfileFiltersUrlState} from './useProfileFiltersUrlState';

// Mock window.location
const mockLocation = {
  pathname: '/test',
  search: '',
};

// Mock the navigate function
const mockNavigateTo = vi.fn((path: string, params: Record<string, string | string[]>) => {
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      if (Array.isArray(value)) {
        searchParams.set(key, value.join(','));
      } else {
        searchParams.set(key, String(value));
      }
    }
  });
  mockLocation.search = `?${searchParams.toString()}`;
});

// Mock getQueryParamsFromURL
vi.mock('@parca/components/src/hooks/URLState/utils', async () => {
  const actual = await vi.importActual('@parca/components/src/hooks/URLState/utils');
  return {
    ...actual,
    getQueryParamsFromURL: () => {
      if (mockLocation.search === '') return {};
      const params = new URLSearchParams(mockLocation.search);
      const result: Record<string, string | string[]> = {};
      for (const [key, value] of params.entries()) {
        const decodedValue = decodeURIComponent(value);
        const existing = result[key];
        if (existing !== undefined) {
          result[key] = Array.isArray(existing)
            ? [...existing, decodedValue]
            : [existing, decodedValue];
        } else {
          result[key] = decodedValue;
        }
      }
      return result;
    },
  };
});

// Helper to create wrapper with URLStateProvider
const createWrapper = (): (({children}: {children: ReactNode}) => JSX.Element) => {
  const Wrapper = ({children}: {children: ReactNode}): JSX.Element => (
    <URLStateProvider navigateTo={mockNavigateTo}>{children}</URLStateProvider>
  );
  Wrapper.displayName = 'URLStateProviderWrapper';
  return Wrapper;
};

describe('useProfileFiltersUrlState', () => {
  beforeEach(() => {
    mockNavigateTo.mockClear();
    Object.defineProperty(window, 'location', {
      value: mockLocation,
      writable: true,
    });
    mockLocation.search = '';
  });

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
      mockLocation.search = '?profile_filters=s:fn:=:testFunc';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('f:b:!~:libc.so');
      });
    });

    it('should clear URL param when setting empty filters', async () => {
      mockLocation.search = '?profile_filters=s:fn:=:testFunc';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      act(() => {
        result.current.setAppliedFilters([]);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // When filters are empty, the param is either empty string or undefined (removed)
        expect(params.profile_filters === '' || params.profile_filters === undefined).toBe(true);
      });
    });
  });

  describe('View defaults', () => {
    it('should provide applyViewDefaults method', () => {
      const viewDefaults: ProfileFilter[] = [
        {
          id: 'default-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'libc.so',
        },
      ];

      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.applyViewDefaults).toBe('function');
    });

    it('should apply view defaults to empty URL', async () => {
      const viewDefaults: ProfileFilter[] = [
        {
          id: 'default-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'libc.so',
        },
      ];

      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.applyViewDefaults();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('f:b:!~:libc.so');
      });
    });

    it('should not overwrite existing filters when applying view defaults (preserve-existing)', async () => {
      mockLocation.search = '?profile_filters=s:fn:=:existingFunc';

      const viewDefaults: ProfileFilter[] = [
        {
          id: 'default-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'libc.so',
        },
      ];

      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      // Verify existing filter is loaded
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('existingFunc');
      });

      mockNavigateTo.mockClear();

      act(() => {
        result.current.applyViewDefaults();
      });

      // With preserve-existing strategy, the existing value should be preserved
      await waitFor(() => {
        // Either no navigation (because value already exists) or value is preserved
        if (mockNavigateTo.mock.calls.length > 0) {
          const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
          // The existing filter should be preserved
          expect(params.profile_filters).toBe('s:fn:=:existingFunc');
        }
      });
    });

    it('should do nothing when viewDefaults is undefined', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      mockNavigateTo.mockClear();

      act(() => {
        result.current.applyViewDefaults();
      });

      // Should not navigate since there are no defaults to apply
      expect(mockNavigateTo).not.toHaveBeenCalled();
    });

    it('should do nothing when viewDefaults is empty array', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults: []}), {
        wrapper: createWrapper(),
      });

      mockNavigateTo.mockClear();

      act(() => {
        result.current.applyViewDefaults();
      });

      // Should not navigate since defaults array is empty
      expect(mockNavigateTo).not.toHaveBeenCalled();
    });
  });

  describe('forceApplyFilters', () => {
    it('should provide forceApplyFilters method', () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      expect(typeof result.current.forceApplyFilters).toBe('function');
    });

    it('should force apply filters overwriting existing', async () => {
      mockLocation.search = '?profile_filters=s:fn:=:existingFunc';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('f:b:!~:forcedValue');
      });
    });

    it('should clear filters when force applying empty array', async () => {
      mockLocation.search = '?profile_filters=s:fn:=:existingFunc';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      act(() => {
        result.current.forceApplyFilters([]);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // When filters are empty, the param is either empty string or undefined (removed)
        expect(params.profile_filters === '' || params.profile_filters === undefined).toBe(true);
      });
    });
  });

  describe('Preset filter encoding', () => {
    it('should encode preset filters correctly', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('p:hide_libc:enabled');
      });
    });

    it('should handle mixed preset and regular filters', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('p:hide_libc:enabled,f:b:!~:node');
      });
    });
  });

  describe('URL encoding edge cases', () => {
    it('should handle special characters in filter values', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // Value should be URL encoded
        expect(params.profile_filters).toContain('std%3A%3Avector%3Cint%3E');
      });
    });

    it('should filter out incomplete filters when encoding', async () => {
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // Only the complete filter should be encoded
        expect(params.profile_filters).toBe('f:b:!~:valid');
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
      mockLocation.search = '?profile_filters=s:fn:=:testFunc';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
      });

      // Verify the filter structure is correct
      const filter = result.current.appliedFilters[0];
      expect(filter).toHaveProperty('id');
      expect(filter).toHaveProperty('type', 'stack');
      expect(filter).toHaveProperty('field', 'function_name');
      expect(filter).toHaveProperty('matchType', 'equal');
      expect(filter).toHaveProperty('value', 'testFunc');
    });
  });

  describe('View switching scenarios', () => {
    it('should completely replace filters when switching views using forceApplyFilters', async () => {
      // Start with View A's filters (2 filters)
      mockLocation.search = '?profile_filters=s:fn:=:viewAFunc,f:b:!=:viewABinary';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(2);
        expect(result.current.appliedFilters[0].value).toBe('viewAFunc');
        expect(result.current.appliedFilters[1].value).toBe('viewABinary');
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
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // View A's filters should be completely gone
        expect(params.profile_filters).not.toContain('viewAFunc');
        expect(params.profile_filters).not.toContain('viewABinary');

        // Only View B's filter should be present
        expect(params.profile_filters).toBe('f:fn:~:viewBOnly');
      });
    });

    it('should handle sequential view switches correctly', async () => {
      // Simulate: [default] -> [storage] -> [testing-view]
      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      // View 1: default view (1 filter)
      const defaultFilters: ProfileFilter[] = [{id: 'd-1', type: 'hide_libc', value: 'enabled'}];

      act(() => {
        result.current.forceApplyFilters(defaultFilters);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('p:hide_libc:enabled');
      });

      mockNavigateTo.mockClear();

      // View 2: storage view (3 filters)
      const storageFilters: ProfileFilter[] = [
        {id: 's-1', type: 'stack', field: 'function_name', matchType: 'not_contains', value: 'io'},
        {id: 's-2', type: 'frame', field: 'binary', matchType: 'not_contains', value: 'disk'},
        {id: 's-3', type: 'frame', field: 'function_name', matchType: 'contains', value: 'storage'},
      ];

      act(() => {
        result.current.forceApplyFilters(storageFilters);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // Default view's filter should be gone
        expect(params.profile_filters).not.toContain('hide_libc');
        // Storage view should have 3 filters
        expect(params.profile_filters).toContain('io');
        expect(params.profile_filters).toContain('disk');
        expect(params.profile_filters).toContain('storage');
      });

      mockNavigateTo.mockClear();

      // View 3: testing-view (2 filters)
      const testingFilters: ProfileFilter[] = [
        {id: 't-1', type: 'stack', field: 'function_name', matchType: 'equal', value: 'test_main'},
        {id: 't-2', type: 'frame', field: 'binary', matchType: 'contains', value: 'test'},
      ];

      act(() => {
        result.current.forceApplyFilters(testingFilters);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // Storage view's filters should be gone
        expect(params.profile_filters).not.toContain('io');
        expect(params.profile_filters).not.toContain('disk');
        expect(params.profile_filters).not.toContain('storage');
        // Testing view should have its 2 filters
        expect(params.profile_filters).toContain('test_main');
        expect(params.profile_filters).toContain('test');
      });
    });

    it('should not change filters when clicking the same view tab', async () => {
      // Start with existing filters
      mockLocation.search = '?profile_filters=s:fn:=:existingFilter';

      const {result} = renderHook(() => useProfileFiltersUrlState(), {wrapper: createWrapper()});

      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
      });

      mockNavigateTo.mockClear();

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

  describe('Page refresh persistence', () => {
    it('should persist user customizations in URL after page refresh simulation', async () => {
      const viewDefaults: ProfileFilter[] = [
        {id: 'default-1', type: 'hide_libc', value: 'enabled'},
      ];

      // User has customized filters (different from defaults)
      mockLocation.search = '?profile_filters=s:fn:=:userCustomFilter';

      const {result, unmount} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      // Verify user's filter is loaded
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('userCustomFilter');
      });

      // Apply view defaults - should NOT overwrite user's URL params (preserve-existing)
      act(() => {
        result.current.applyViewDefaults();
      });

      // User's filter should still be preserved
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('userCustomFilter');
      });

      // Simulate page refresh
      unmount();
      mockNavigateTo.mockClear();

      // Re-render hook (simulating page reload)
      const {result: result2} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      // After "refresh", filter should still be from URL
      await waitFor(() => {
        expect(result2.current.appliedFilters).toHaveLength(1);
        expect(result2.current.appliedFilters[0].value).toBe('userCustomFilter');
      });
    });

    it('should apply view defaults when URL is empty on page load', async () => {
      const viewDefaults: ProfileFilter[] = [
        {
          id: 'default-1',
          type: 'frame',
          field: 'binary',
          matchType: 'not_contains',
          value: 'libc.so',
        },
      ];

      // Empty URL (fresh page load)
      mockLocation.search = '';

      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      // Apply view defaults
      act(() => {
        result.current.applyViewDefaults();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.profile_filters).toBe('f:b:!~:libc.so');
      });
    });

    it('should handle shared/bookmarked URL with custom params', async () => {
      const viewDefaults: ProfileFilter[] = [
        {id: 'default-1', type: 'hide_libc', value: 'enabled'},
        {id: 'default-2', type: 'hide_python_internals', value: 'enabled'},
      ];

      // Shared URL with custom params (not matching view defaults)
      mockLocation.search = '?profile_filters=s:fn:~:customSharedFilter';

      const {result} = renderHook(() => useProfileFiltersUrlState({viewDefaults}), {
        wrapper: createWrapper(),
      });

      // Verify custom params are loaded
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('customSharedFilter');
      });

      // Apply view defaults - should NOT overwrite
      act(() => {
        result.current.applyViewDefaults();
      });

      // Custom params should be honored over view defaults
      await waitFor(() => {
        expect(result.current.appliedFilters).toHaveLength(1);
        expect(result.current.appliedFilters[0].value).toBe('customSharedFilter');
      });
    });
  });
});
