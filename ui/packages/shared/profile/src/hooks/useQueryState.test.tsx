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

import {ReactNode, act} from 'react';

// eslint-disable-next-line import/named
import {QueryClient, QueryClientProvider} from '@tanstack/react-query';
// eslint-disable-next-line import/named
import {renderHook, waitFor} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';

import {URLStateProvider} from '@parca/components';

import {useQueryState} from './useQueryState';

// Mock window.location
const mockLocation = {
  pathname: '/test',
  search: '',
};

// Mock the navigate function that actually updates the mock location
const mockNavigateTo = vi.fn((path: string, params: Record<string, string | string[]>) => {
  // Convert params object to query string
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      if (Array.isArray(value)) {
        // For arrays, join with commas
        searchParams.set(key, value.join(','));
      } else {
        searchParams.set(key, String(value));
      }
    }
  });
  mockLocation.search = `?${searchParams.toString()}`;
});

// Mock the getQueryParamsFromURL function
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

// Mock useSumBy with stateful behavior using React's useState
vi.mock('../useSumBy', async () => {
  const actual = await vi.importActual('../useSumBy');
  const react = await import('react');

  return {
    ...actual,
    useSumBy: (
      _queryClient: any,
      _profileType: any,
      _timeRange: any,
      _draftProfileType: any,
      _draftTimeRange: any,
      defaultValue: any
    ) => {
      const [draftSumBy, setDraftSumBy] = react.useState<string[] | undefined>(defaultValue);
      const [sumBy, setSumBy] = react.useState<string[] | undefined>(defaultValue);

      return {
        sumBy,
        setSumBy,
        isLoading: false,
        draftSumBy,
        setDraftSumBy,
        isDraftSumByLoading: false,
      };
    },
  };
});

// Track profile types loading state for tests
let mockProfileTypesLoading = false;
let mockProfileTypesData:
  | {
      types: Array<{
        name: string;
        sampleType: string;
        sampleUnit: string;
        periodType: string;
        periodUnit: string;
        delta: boolean;
      }>;
    }
  | undefined;

// Mock useProfileTypes to control loading state in tests
vi.mock('../ProfileSelector', async () => {
  const actual = await vi.importActual('../ProfileSelector');
  return {
    ...actual,
    useProfileTypes: () => ({
      loading: mockProfileTypesLoading,
      data: mockProfileTypesData,
      error: null,
    }),
  };
});

// Helper to set profile types loading state for tests
const setProfileTypesLoading = (loading: boolean): void => {
  mockProfileTypesLoading = loading;
};

const setProfileTypesData = (data: typeof mockProfileTypesData): void => {
  mockProfileTypesData = data;
};

// Helper to create wrapper with URLStateProvider
const createWrapper = (
  paramPreferences = {}
): (({children}: {children: ReactNode}) => JSX.Element) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  const Wrapper = ({children}: {children: ReactNode}): JSX.Element => (
    <QueryClientProvider client={queryClient}>
      <URLStateProvider navigateTo={mockNavigateTo} paramPreferences={paramPreferences}>
        {children}
      </URLStateProvider>
    </QueryClientProvider>
  );
  Wrapper.displayName = 'URLStateProviderWrapper';
  return Wrapper;
};

describe('useQueryState', () => {
  beforeEach(() => {
    mockNavigateTo.mockClear();
    Object.defineProperty(window, 'location', {
      value: mockLocation,
      writable: true,
    });
    mockLocation.search = '';
    // Reset profile types mock state
    setProfileTypesLoading(false);
    setProfileTypesData(undefined);
  });

  describe('Basic functionality', () => {
    it('should initialize with default values', () => {
      const {result} = renderHook(
        () =>
          useQueryState({
            defaultExpression: 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{}',
            defaultTimeSelection: 'relative:hour|1',
            defaultFrom: 1000,
            defaultTo: 2000,
          }),
        {wrapper: createWrapper()}
      );

      const {querySelection} = result.current;
      expect(querySelection.expression).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{}');
      expect(querySelection.timeSelection).toBe('relative:hour|1');
      // From/to should be calculated from the range
      expect(querySelection.from).toBeDefined();
      expect(querySelection.to).toBeDefined();
    });

    it('should handle suffix for comparison mode', () => {
      mockLocation.search =
        '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from_a=1000&to_a=2000';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const {querySelection} = result.current;
      expect(querySelection.expression).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}');
      expect(querySelection.from).toBe(1000);
      expect(querySelection.to).toBe(2000);
    });
  });

  describe('Individual setters', () => {
    it('should update expression and handle delta profiles', async () => {
      const {result} = renderHook(
        () =>
          useQueryState({
            defaultFrom: 1000,
            defaultTo: 2000,
          }),
        {wrapper: createWrapper()}
      );

      act(() => {
        result.current.setDraftExpression('memory:alloc_objects:count:space:bytes:delta{}');
      });

      // Draft should be updated but not committed
      expect(result.current.draftSelection.expression).toBe(
        'memory:alloc_objects:count:space:bytes:delta{}'
      );

      // Delta profile should auto-calculate merge params in draft
      expect(result.current.draftSelection.mergeFrom).toBe('1000000000');
      expect(result.current.draftSelection.mergeTo).toBe('2000000000');

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:alloc_objects:count:space:bytes:delta{}');
        // Should set merge parameters for delta profile
        expect(params).toHaveProperty('merge_from');
        expect(params).toHaveProperty('merge_to');
        expect(params.merge_from).toBe('1000000000');
        expect(params.merge_to).toBe('2000000000');
      });
    });

    it('should update time range', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        result.current.setDraftTimeRange(3000, 4000, 'relative:minute|5');
      });

      // Draft should be updated
      expect(result.current.draftSelection.from).toBe(3000);
      expect(result.current.draftSelection.to).toBe(4000);
      expect(result.current.draftSelection.timeSelection).toBe('relative:minute|5');

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.from).toBe('3000');
        expect(params.to).toBe('4000');
        expect(params.time_selection).toBe('relative:minute|5');
      });
    });

    it('should update sumBy', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // sumBy only applies to delta profiles, so we need to set one first
      act(() => {
        result.current.setDraftExpression('process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}');
        result.current.setDraftSumBy(['namespace', 'container']);
      });

      // Draft should be updated
      expect(result.current.draftSelection.sumBy).toEqual(['namespace', 'container']);

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.sum_by).toBe('namespace,container');
      });
    });

    it('should auto-calculate merge range for delta profiles', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Set a delta profile expression
      act(() => {
        result.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result.current.setDraftTimeRange(5000, 6000, 'relative:minute|5');
      });

      // Merge range should be auto-calculated in draft
      expect(result.current.draftSelection.mergeFrom).toBe('5000000000');
      expect(result.current.draftSelection.mergeTo).toBe('6000000000');

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.merge_from).toBe('5000000000');
        expect(params.merge_to).toBe('6000000000');
      });
    });
  });

  describe('Batch updates', () => {
    it('should batch multiple updates into single navigation', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        // Update multiple draft values (using delta profile since sumBy only applies to delta)
        result.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result.current.setDraftTimeRange(7000, 8000, 'relative:minute|30');
        result.current.setDraftSumBy(['pod', 'node']);
      });

      // All drafts should be updated
      expect(result.current.draftSelection.expression).toBe(
        'memory:alloc_space:bytes:space:bytes:delta{}'
      );
      expect(result.current.draftSelection.from).toBe(7000);
      expect(result.current.draftSelection.to).toBe(8000);
      expect(result.current.draftSelection.sumBy).toEqual(['pod', 'node']);

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        // Should only navigate once for all updates
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        const [, params] = mockNavigateTo.mock.calls[0];
        expect(params.expression).toBe('memory:alloc_space:bytes:space:bytes:delta{}');
        expect(params.from).toBe('7000');
        expect(params.to).toBe('8000');
        expect(params.time_selection).toBe('relative:minute|30');
        expect(params.sum_by).toBe('pod,node');
      });
    });

    it('should handle partial updates', async () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000&time_selection=relative:hour|1';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        // Only update expression, other values should remain
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      expect(result.current.draftSelection.expression).toBe(
        'memory:inuse_space:bytes:space:bytes{}'
      );
      // Other values should be from URL
      expect(result.current.draftSelection.from).toBe(1000);
      expect(result.current.draftSelection.to).toBe(2000);

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:inuse_space:bytes:space:bytes{}');
        expect(params.from).toBe('1000');
        expect(params.to).toBe('2000');
        expect(params.time_selection).toBe('relative:hour|1');
      });
    });

    it('should auto-calculate merge params for delta profiles in batch update', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        result.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result.current.setDraftTimeRange(9000, 10000, 'relative:minute|5');
      });

      // Merge params should be auto-calculated in draft
      expect(result.current.draftSelection.mergeFrom).toBe('9000000000');
      expect(result.current.draftSelection.mergeTo).toBe('10000000000');

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:alloc_space:bytes:space:bytes:delta{}');
        expect(params.merge_from).toBe('9000000000');
        expect(params.merge_to).toBe('10000000000');
      });
    });
  });

  describe('Helper functions', () => {
    it('should set profile name correctly', async () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{job="parca"}';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        result.current.setDraftProfileName('memory:inuse_space:bytes:space:bytes');
      });

      // Draft should be updated
      expect(result.current.draftSelection.expression).toBe(
        'memory:inuse_space:bytes:space:bytes{job="parca"}'
      );

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:inuse_space:bytes:space:bytes{job="parca"}');
      });
    });

    it('should set matchers correctly using draft', async () => {
      mockLocation.search = '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      act(() => {
        result.current.setDraftMatchers('namespace="default",pod="my-pod"');
      });

      // Draft should be updated but not URL yet
      expect(result.current.draftSelection.expression).toBe(
        'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{namespace="default",pod="my-pod"}'
      );
      expect(mockNavigateTo).not.toHaveBeenCalled();

      // Commit the draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{namespace="default",pod="my-pod"}'
        );
      });
    });
  });

  describe('Comparison mode', () => {
    it('should handle _a suffix correctly', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      // Update draft state
      act(() => {
        result.current.setDraftExpression('process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}');
        result.current.setDraftTimeRange(1111, 2222, 'relative:hour|1');
        result.current.setDraftSumBy(['label_a']);
      });

      // Commit draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression_a).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}');
        expect(params.from_a).toBe('1111');
        expect(params.to_a).toBe('2222');
        expect(params.sum_by_a).toBe('label_a');
      });
    });

    it('should handle _b suffix correctly', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_b'}), {wrapper: createWrapper()});

      // Update draft state (using delta profile since sumBy only applies to delta)
      act(() => {
        result.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result.current.setDraftTimeRange(3333, 4444, 'relative:hour|2');
        result.current.setDraftSumBy(['label_b']);
      });

      // Commit draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression_b).toBe('memory:alloc_space:bytes:space:bytes:delta{}');
        expect(params.from_b).toBe('3333');
        expect(params.to_b).toBe('4444');
        expect(params.sum_by_b).toBe('label_b');
      });
    });
  });

  describe('Draft state pattern', () => {
    it('should not update URL until commit', async () => {
      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Make multiple draft changes (using delta profile since sumBy only applies to delta)
      act(() => {
        result.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result.current.setDraftTimeRange(5000, 6000, 'relative:hour|3');
        result.current.setDraftSumBy(['namespace', 'pod']);
      });

      // URL should not be updated yet
      expect(mockNavigateTo).not.toHaveBeenCalled();

      // Commit all changes at once
      act(() => {
        result.current.commitDraft();
      });

      // Now URL should be updated exactly once with all changes
      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        const [, params] = mockNavigateTo.mock.calls[0];
        expect(params.expression).toBe('memory:alloc_space:bytes:space:bytes:delta{}');
        expect(params.from).toBe('5000');
        expect(params.to).toBe('6000');
        expect(params.sum_by).toBe('namespace,pod');
      });
    });

    it('should handle draft profile name changes', () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{job="test"}';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Change profile name in draft
      act(() => {
        result.current.setDraftProfileName('memory:inuse_space:bytes:space:bytes');
      });

      // Draft should be updated
      expect(result.current.draftSelection.expression).toBe(
        'memory:inuse_space:bytes:space:bytes{job="test"}'
      );

      // URL should not be updated yet
      expect(mockNavigateTo).not.toHaveBeenCalled();
    });
  });

  describe('Edge cases', () => {
    it('should handle invalid expression gracefully and log warning', () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});

      const {result} = renderHook(
        () =>
          useQueryState({
            defaultExpression: 'invalid{{}expression',
          }),
        {wrapper: createWrapper()}
      );

      // Should not throw error - invalid expressions are caught and logged
      expect(() => result.current.querySelection).not.toThrow();
      // Should fall back to empty expression
      expect(result.current.querySelection.expression).toBe('invalid{{}expression');
      // Should log a warning about the parse failure
      expect(consoleSpy).toHaveBeenCalledWith(
        'Failed to parse expression',
        expect.objectContaining({
          expression: 'invalid{{}expression',
        })
      );

      consoleSpy.mockRestore();
    });

    it('should handle empty expression gracefully', () => {
      const {result} = renderHook(
        () =>
          useQueryState({
            defaultExpression: '',
          }),
        {wrapper: createWrapper()}
      );

      // Should not throw error with empty expression
      expect(() => result.current.querySelection).not.toThrow();
      expect(result.current.querySelection.expression).toBe('');
    });

    it('should clear merge params for non-delta profiles', async () => {
      mockLocation.search =
        '?expression=memory:alloc_objects:count:space:bytes:delta{}&merge_from=1000000000&merge_to=2000000000';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Switch to non-delta profile (without :delta suffix) using draft
      act(() => {
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      // Commit the draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:inuse_space:bytes:space:bytes{}');
        expect(params).not.toHaveProperty('merge_from');
        expect(params).not.toHaveProperty('merge_to');
      });
    });

    it('should preserve other URL parameters when updating', async () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&other_param=value&unrelated=test';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Update draft and commit
      act(() => {
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.expression).toBe('memory:inuse_space:bytes:space:bytes{}');
        expect(params.other_param).toBe('value');
        expect(params.unrelated).toBe('test');
      });
    });
  });

  describe('Commit with refreshed time range (time range re-evaluation)', () => {
    it('should use refreshed time range values instead of draft state when provided', async () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&from=1000&to=2000&time_selection=relative:minute|15';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Draft state has original values
      expect(result.current.draftSelection.from).toBe(1000);
      expect(result.current.draftSelection.to).toBe(2000);
      expect(result.current.draftSelection.timeSelection).toBe('relative:minute|15');

      // Commit with refreshed time range (simulating re-evaluated time range)
      act(() => {
        result.current.commitDraft({
          from: 5000,
          to: 6000,
          timeSelection: 'relative:minute|15',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        // Should use refreshed time range values, not draft values
        expect(params.from).toBe('5000');
        expect(params.to).toBe('6000');
        expect(params.time_selection).toBe('relative:minute|15');
      });
    });

    it('should update draft state with refreshed time range after commit', async () => {
      const {result} = renderHook(
        () =>
          useQueryState({
            defaultExpression: 'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}',
            defaultFrom: 1000,
            defaultTo: 2000,
            defaultTimeSelection: 'relative:minute|5',
          }),
        {wrapper: createWrapper()}
      );

      // Commit with refreshed time values
      act(() => {
        result.current.commitDraft({
          from: 3000,
          to: 4000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
      });

      // Draft state should be updated with the refreshed time range
      expect(result.current.draftSelection.from).toBe(3000);
      expect(result.current.draftSelection.to).toBe(4000);
    });

    it('should trigger navigation even when expression unchanged (time re-evaluation)', async () => {
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&from=1000&to=2000&time_selection=relative:minute|5';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      mockNavigateTo.mockClear();

      // First commit with new time values
      act(() => {
        result.current.commitDraft({
          from: 5000,
          to: 6000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
      });

      const firstCallParams = mockNavigateTo.mock.calls[0][1];
      expect(firstCallParams.from).toBe('5000');
      expect(firstCallParams.to).toBe('6000');

      mockNavigateTo.mockClear();

      // Second commit with different time values (simulating clicking Search again)
      act(() => {
        result.current.commitDraft({
          from: 7000,
          to: 8000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
      });

      const secondCallParams = mockNavigateTo.mock.calls[0][1];
      expect(secondCallParams.from).toBe('7000');
      expect(secondCallParams.to).toBe('8000');

      // Verify that navigation was called both times despite expression being unchanged
      expect(firstCallParams.from).not.toBe(secondCallParams.from);
    });

    it('should auto-calculate merge params for delta profiles when using refreshed time range', async () => {
      const {result} = renderHook(
        () =>
          useQueryState({
            defaultExpression: 'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}',
            defaultFrom: 1000,
            defaultTo: 2000,
          }),
        {wrapper: createWrapper()}
      );

      // Commit with refreshed time range for delta profile
      act(() => {
        result.current.commitDraft({
          from: 5000,
          to: 6000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // Verify merge params are calculated from refreshed time range
        expect(params.merge_from).toBe('5000000000'); // 5000ms * 1_000_000
        expect(params.merge_to).toBe('6000000000'); // 6000ms * 1_000_000
      });
    });

    it('should use draft values when refreshedTimeRange is not provided', async () => {
      mockLocation.search =
        '?expression=memory:inuse_space:bytes:space:bytes{}&from=1000&to=2000&time_selection=relative:hour|1';

      const {result} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Change draft values
      act(() => {
        result.current.setDraftTimeRange(3000, 4000, 'relative:minute|30');
      });

      // Commit without refreshedTimeRange - should use draft values
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // Should use updated draft values
        expect(params.from).toBe('3000');
        expect(params.to).toBe('4000');
        expect(params.time_selection).toBe('relative:minute|30');
      });
    });
  });

  describe('State persistence after page reload', () => {
    it('should retain committed values after page reload simulation', async () => {
      // Initial state (using delta profile since sumBy only applies to delta)
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000';

      const {result: result1, unmount} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(),
      });

      // User makes changes to draft (using delta profile since sumBy only applies to delta)
      act(() => {
        result1.current.setDraftExpression('memory:alloc_space:bytes:space:bytes:delta{}');
        result1.current.setDraftTimeRange(5000, 6000, 'relative:minute|15');
        result1.current.setDraftSumBy(['namespace', 'pod']);
      });

      // User clicks Search to commit
      act(() => {
        result1.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
      });

      // Get the params that were committed to URL
      const committedParams = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1][1];

      // Simulate page reload by updating mockLocation.search with committed values
      const queryString = new URLSearchParams({
        expression: committedParams.expression as string,
        from: committedParams.from as string,
        to: committedParams.to as string,
        time_selection: committedParams.time_selection as string,
        sum_by: committedParams.sum_by as string,
      }).toString();

      mockLocation.search = `?${queryString}`;

      // Unmount the old hook instance
      unmount();

      // Clear navigation mock to verify no new navigation on reload
      mockNavigateTo.mockClear();

      // Create new hook instance (simulating page reload)
      const {result: result2} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Verify state is loaded from URL after "reload"
      expect(result2.current.querySelection.expression).toBe(
        'memory:alloc_space:bytes:space:bytes:delta{}'
      );
      expect(result2.current.querySelection.from).toBe(5000);
      expect(result2.current.querySelection.to).toBe(6000);
      expect(result2.current.querySelection.timeSelection).toBe('relative:minute|15');
      expect(result2.current.querySelection.sumBy).toEqual(['namespace', 'pod']);

      // Draft should be synced with URL state on page load
      expect(result2.current.draftSelection.expression).toBe(
        'memory:alloc_space:bytes:space:bytes:delta{}'
      );
      expect(result2.current.draftSelection.from).toBe(5000);
      expect(result2.current.draftSelection.to).toBe(6000);
      expect(result2.current.draftSelection.sumBy).toEqual(['namespace', 'pod']);

      // No navigation should occur on page load
      expect(mockNavigateTo).not.toHaveBeenCalled();
    });

    it('should preserve delta profile merge params after reload', async () => {
      // Initial state with delta profile
      mockLocation.search =
        '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000';

      const {result: result1, unmount} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(),
      });

      // Commit with time override
      act(() => {
        result1.current.commitDraft({
          from: 5000,
          to: 6000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
      });

      const committedParams = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1][1];

      // Verify merge params were set
      expect(committedParams.merge_from).toBe('5000000000');
      expect(committedParams.merge_to).toBe('6000000000');

      // Simulate page reload with all params including merge params
      const queryString = new URLSearchParams({
        expression: committedParams.expression as string,
        from: committedParams.from as string,
        to: committedParams.to as string,
        time_selection: committedParams.time_selection as string,
        merge_from: committedParams.merge_from as string,
        merge_to: committedParams.merge_to as string,
      }).toString();

      mockLocation.search = `?${queryString}`;
      unmount();
      mockNavigateTo.mockClear();

      // Create new hook instance
      const {result: result2} = renderHook(() => useQueryState(), {wrapper: createWrapper()});

      // Verify merge params are preserved
      expect(result2.current.querySelection.mergeFrom).toBe('5000000000');
      expect(result2.current.querySelection.mergeTo).toBe('6000000000');

      // Draft should also have merge params
      expect(result2.current.draftSelection.mergeFrom).toBe('5000000000');
      expect(result2.current.draftSelection.mergeTo).toBe('6000000000');
    });
  });

  describe('ProfileSelection state management', () => {
    it('should initialize with null ProfileSelection when no URL params exist', () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      expect(result.current.profileSelection).toBeNull();
    });

    it('should compute ProfileSelection from URL params', () => {
      // Set URL with ProfileSelection params - using valid profile type
      mockLocation.search =
        '?merge_from_a=1234567890&merge_to_a=9876543210&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{pod="test"}';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const {profileSelection} = result.current;
      expect(profileSelection).not.toBeNull();

      // Test using the interface methods
      expect(profileSelection?.Type()).toBe('merge');
      expect(profileSelection?.ProfileName()).toBe(
        'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta'
      );

      // Test HistoryParams which should return merge params
      const historyParams = profileSelection?.HistoryParams();
      expect(historyParams?.merge_from).toBe('1234567890');
      expect(historyParams?.merge_to).toBe('9876543210');
      expect(historyParams?.selection).toBe(
        'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{pod="test"}'
      );
    });

    it('should auto-commit ProfileSelection to URL when setProfileSelection called', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const mergeFrom = BigInt(5000000000);
      const mergeTo = BigInt(6000000000);

      // Create a mock Query object - in real code, this would be Query.parse()
      const mockQuery = {
        toString: () => 'memory:inuse_space:bytes:space:bytes{namespace="default"}',
        profileType: () => ({delta: false}),
      } as any;

      act(() => {
        result.current.setProfileSelection(mergeFrom, mergeTo, mockQuery);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.selection_a).toBe(
          'memory:inuse_space:bytes:space:bytes{namespace="default"}'
        );
        expect(params.merge_from_a).toBe('5000000000');
        expect(params.merge_to_a).toBe('6000000000');
      });
    });

    it('should use correct suffix for ProfileSelection in comparison mode', async () => {
      const {result: resultB} = renderHook(() => useQueryState({suffix: '_b'}), {
        wrapper: createWrapper(),
      });

      const mergeFrom = BigInt(7000000000);
      const mergeTo = BigInt(8000000000);

      const mockQuery = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}',
        profileType: () => ({delta: false}),
      } as any;

      act(() => {
        resultB.current.setProfileSelection(mergeFrom, mergeTo, mockQuery);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.selection_b).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}');
        expect(params.merge_from_b).toBe('7000000000');
        expect(params.merge_to_b).toBe('8000000000');
      });
    });

    it('should clear ProfileSelection when commitDraft is called', async () => {
      // Start with a ProfileSelection in URL - using valid profile type
      mockLocation.search =
        '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&merge_from_a=1000000000&merge_to_a=2000000000&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      // Verify ProfileSelection exists
      expect(result.current.profileSelection).not.toBeNull();

      // Make a change to trigger commit
      act(() => {
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      // Commit the draft (this should clear ProfileSelection as per design decision 4.B)
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // ProfileSelection params should be cleared
        expect(params).not.toHaveProperty('selection_a');

        // But QuerySelection params should still be present
        expect(params.expression_a).toBe('memory:inuse_space:bytes:space:bytes{}');
      });
    });

    it('should handle ProfileSelection with delta profiles correctly', () => {
      mockLocation.search =
        '?merge_from_a=1000000000&merge_to_a=2000000000&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{node="worker"}';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const {profileSelection} = result.current;
      expect(profileSelection).not.toBeNull();

      // Test that ProfileSelection recognizes delta profile type
      expect(profileSelection?.ProfileName()).toBe(
        'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta'
      );

      // Test HistoryParams
      const historyParams = profileSelection?.HistoryParams();
      expect(historyParams?.merge_from).toBe('1000000000');
      expect(historyParams?.merge_to).toBe('2000000000');
    });

    it('should persist ProfileSelection across page reloads', async () => {
      // Initial state - user clicks on metrics graph point
      const {result: result1, unmount} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(),
      });

      const mergeFrom = BigInt(3000000000);
      const mergeTo = BigInt(4000000000);
      const mockQuery = {
        toString: () => 'memory:alloc_objects:count:space:bytes{pod="test"}',
        profileType: () => ({delta: false}),
      } as any;

      // Set ProfileSelection
      act(() => {
        result1.current.setProfileSelection(mergeFrom, mergeTo, mockQuery);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
      });

      const committedParams = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1][1];

      // Simulate page reload by updating mockLocation.search
      const selectionA = String(committedParams.selection_a ?? '');
      const mergeFromA = String(committedParams.merge_from_a ?? '');
      const mergeToA = String(committedParams.merge_to_a ?? '');
      mockLocation.search = `?selection_a=${encodeURIComponent(
        selectionA
      )}&merge_from_a=${mergeFromA}&merge_to_a=${mergeToA}`;
      unmount();
      mockNavigateTo.mockClear();

      // Create new hook instance (simulating page reload)
      const {result: result2} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(),
      });

      // Verify ProfileSelection is loaded from URL after reload
      const profileSelection = result2.current.profileSelection;
      expect(profileSelection).not.toBeNull();

      // Use interface methods to test
      expect(profileSelection?.Type()).toBe('merge');
      const historyParams = profileSelection?.HistoryParams();
      expect(historyParams?.merge_from).toBe('3000000000');
      expect(historyParams?.merge_to).toBe('4000000000');
      expect(historyParams?.selection).toBe('memory:alloc_objects:count:space:bytes{pod="test"}');

      // No navigation should occur on page load
      expect(mockNavigateTo).not.toHaveBeenCalled();
    });

    it('should handle independent ProfileSelection for both sides in comparison mode', async () => {
      // Test component using both hooks with the same URLStateProvider (real-world scenario)
      // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
      const TestComponent = () => {
        const stateA = useQueryState({suffix: '_a'});
        const stateB = useQueryState({suffix: '_b'});
        return {stateA, stateB};
      };

      const {result} = renderHook(() => TestComponent(), {
        wrapper: createWrapper(),
      });

      const mockQueryA = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="app-a"}',
        profileType: () => ({delta: false}),
      } as any;

      const mockQueryB = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="app-b"}',
        profileType: () => ({delta: false}),
      } as any;

      // Set ProfileSelection for side A
      act(() => {
        result.current.stateA.setProfileSelection(
          BigInt(1000000000),
          BigInt(2000000000),
          mockQueryA
        );
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
      });

      mockNavigateTo.mockClear();

      // Set ProfileSelection for side B
      act(() => {
        result.current.stateB.setProfileSelection(
          BigInt(3000000000),
          BigInt(4000000000),
          mockQueryB
        );
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // Both selections should be in URL with different suffixes
        expect(params.selection_a).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="app-a"}');
        expect(params.selection_b).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="app-b"}');
        expect(params.merge_from_a).toBe('1000000000');
        expect(params.merge_from_b).toBe('3000000000');
      });

      // The mockNavigateTo automatically updates mockLocation.search, so the URL change
      // should propagate to the hooks automatically. Verify both ProfileSelections exist.
      await waitFor(() => {
        expect(result.current.stateA.profileSelection).not.toBeNull();
        expect(result.current.stateB.profileSelection).not.toBeNull();
      });
    });

    it('should return null ProfileSelection when only partial params exist', () => {
      // Missing selection param
      mockLocation.search = '?merge_from_a=1000000000&merge_to_a=2000000000';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      expect(result.current.profileSelection).toBeNull();
    });

    it('should handle ProfileSelection with complex query expressions', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const mockQuery = {
        toString: () =>
          'memory:alloc_objects:count:space:bytes:delta{namespace="default",pod="app-1",container="main"}',
        profileType: () => ({delta: true}),
      } as any;

      act(() => {
        result.current.setProfileSelection(BigInt(5000000000), BigInt(6000000000), mockQuery);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];
        expect(params.selection_a).toBe(
          'memory:alloc_objects:count:space:bytes:delta{namespace="default",pod="app-1",container="main"}'
        );
      });
    });

    it('should batch ProfileSelection update with other URL state changes', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const mockQuery = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}',
        profileType: () => ({delta: false}),
      } as any;

      // The batchUpdates is used internally by setProfileSelection
      act(() => {
        result.current.setProfileSelection(BigInt(1000000000), BigInt(2000000000), mockQuery);
      });

      await waitFor(() => {
        // Should only navigate once despite setting 3 params (selection, merge_from, merge_to)
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        const [, params] = mockNavigateTo.mock.calls[0];
        expect(params.selection_a).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}');
        expect(params.merge_from_a).toBe('1000000000');
        expect(params.merge_to_a).toBe('2000000000');
      });
    });

    it('should preserve other URL params when setting ProfileSelection', async () => {
      mockLocation.search =
        '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&other_param=value&unrelated=test';

      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {wrapper: createWrapper()});

      const mockQuery = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}',
        profileType: () => ({delta: false}),
      } as any;

      act(() => {
        result.current.setProfileSelection(BigInt(1000000000), BigInt(2000000000), mockQuery);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalled();
        const [, params] = mockNavigateTo.mock.calls[mockNavigateTo.mock.calls.length - 1];

        // ProfileSelection params should be set
        expect(params.selection_a).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}');

        // Other params should be preserved
        expect(params.expression_a).toBe('process_cpu:cpu:nanoseconds:cpu:nanoseconds{}');
        expect(params.other_param).toBe('value');
        expect(params.unrelated).toBe('test');
      });
    });
  });
});
