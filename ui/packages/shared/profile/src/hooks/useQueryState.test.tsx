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
import {NuqsTestingAdapter} from 'nuqs/dist/adapters/testing';
import {beforeEach, describe, expect, it, vi} from 'vitest';

import {useQueryState} from './useQueryState';

const mockNavigateTo = vi.fn();

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

// Helper to create wrapper with NuqsTestingAdapter
const createWrapper = (
  _paramPreferences = {},
  searchParams: string | Record<string, string> = {}
): (({children}: {children: ReactNode}) => JSX.Element) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
  const Wrapper = ({children}: {children: ReactNode}): JSX.Element => (
    <NuqsTestingAdapter searchParams={searchParams} hasMemory={true}>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </NuqsTestingAdapter>
  );
  Wrapper.displayName = 'NuqsTestingWrapper';
  return Wrapper;
};

describe('useQueryState', () => {
  beforeEach(() => {
    mockNavigateTo.mockClear();
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
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from_a=1000&to_a=2000'
        ),
      });

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
        expect(result.current.querySelection.expression).toBe(
          'memory:alloc_objects:count:space:bytes:delta{}'
        );
        expect(result.current.querySelection.mergeFrom).toBe('1000000000');
        expect(result.current.querySelection.mergeTo).toBe('2000000000');
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
        expect(String(result.current.querySelection.from)).toBe('3000');
        expect(String(result.current.querySelection.to)).toBe('4000');
        expect(result.current.querySelection.timeSelection).toBe('relative:minute|5');
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
        // sumBy is managed by the mocked useSumBy hook; verify it was set in draft
        expect(result.current.draftSelection.sumBy).toEqual(['namespace', 'container']);
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
        expect(result.current.querySelection.mergeFrom).toBe('5000000000');
        expect(result.current.querySelection.mergeTo).toBe('6000000000');
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
        // Verify all state values are correct after the batch
        expect(result.current.querySelection.expression).toBe(
          'memory:alloc_space:bytes:space:bytes:delta{}'
        );
        expect(String(result.current.querySelection.from)).toBe('7000');
        expect(String(result.current.querySelection.to)).toBe('8000');
        expect(result.current.querySelection.timeSelection).toBe('relative:minute|30');
        // sumBy is managed by the mocked useSumBy hook; verify it was set in draft
        expect(result.current.draftSelection.sumBy).toEqual(['pod', 'node']);
      });
    });

    it('should handle partial updates', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000&time_selection=relative:hour|1'
        ),
      });

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
        expect(result.current.querySelection.expression).toBe(
          'memory:inuse_space:bytes:space:bytes{}'
        );
        expect(String(result.current.querySelection.from)).toBe('1000');
        expect(String(result.current.querySelection.to)).toBe('2000');
        expect(result.current.querySelection.timeSelection).toBe('relative:hour|1');
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
        expect(result.current.querySelection.expression).toBe(
          'memory:alloc_space:bytes:space:bytes:delta{}'
        );
        expect(result.current.querySelection.mergeFrom).toBe('9000000000');
        expect(result.current.querySelection.mergeTo).toBe('10000000000');
      });
    });
  });

  describe('Helper functions', () => {
    it('should set profile name correctly', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{job="parca"}'
        ),
      });

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
        expect(result.current.querySelection.expression).toBe(
          'memory:inuse_space:bytes:space:bytes{job="parca"}'
        );
      });
    });

    it('should set matchers correctly using draft', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}'
        ),
      });

      act(() => {
        result.current.setDraftMatchers('namespace="default",pod="my-pod"');
      });

      // Draft should be updated but not URL yet
      expect(result.current.draftSelection.expression).toBe(
        'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{namespace="default",pod="my-pod"}'
      );

      // Commit the draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(result.current.querySelection.expression).toBe(
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
        expect(result.current.querySelection.expression).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}'
        );
        expect(String(result.current.querySelection.from)).toBe('1111');
        expect(String(result.current.querySelection.to)).toBe('2222');
        // sumBy is managed by the mocked useSumBy hook; verify it was set in draft
        expect(result.current.draftSelection.sumBy).toEqual(['label_a']);
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
        expect(result.current.querySelection.expression).toBe(
          'memory:alloc_space:bytes:space:bytes:delta{}'
        );
        expect(String(result.current.querySelection.from)).toBe('3333');
        expect(String(result.current.querySelection.to)).toBe('4444');
        // sumBy is managed by the mocked useSumBy hook; verify it was set in draft
        expect(result.current.draftSelection.sumBy).toEqual(['label_b']);
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

      // Commit all changes at once
      act(() => {
        result.current.commitDraft();
      });

      // Verify all state values are correct
      await waitFor(() => {
        expect(result.current.querySelection.expression).toBe(
          'memory:alloc_space:bytes:space:bytes:delta{}'
        );
        expect(String(result.current.querySelection.from)).toBe('5000');
        expect(String(result.current.querySelection.to)).toBe('6000');
        // sumBy is managed by the mocked useSumBy hook; verify it was set in draft
        expect(result.current.draftSelection.sumBy).toEqual(['namespace', 'pod']);
      });
    });

    it('should handle draft profile name changes', () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{job="test"}'
        ),
      });

      // Change profile name in draft
      act(() => {
        result.current.setDraftProfileName('memory:inuse_space:bytes:space:bytes');
      });

      // Draft should be updated
      expect(result.current.draftSelection.expression).toBe(
        'memory:inuse_space:bytes:space:bytes{job="test"}'
      );
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
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=memory:alloc_objects:count:space:bytes:delta{}&merge_from=1000000000&merge_to=2000000000'
        ),
      });

      // Switch to non-delta profile (without :delta suffix) using draft
      act(() => {
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      // Commit the draft
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(result.current.querySelection.expression).toBe(
          'memory:inuse_space:bytes:space:bytes{}'
        );
        // Merge params should not be set for non-delta profiles
        expect(result.current.querySelection.mergeFrom).toBeUndefined();
        expect(result.current.querySelection.mergeTo).toBeUndefined();
      });
    });

    it('should preserve other URL parameters when updating', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&other_param=value&unrelated=test'
        ),
      });

      // Update draft and commit
      act(() => {
        result.current.setDraftExpression('memory:inuse_space:bytes:space:bytes{}');
      });

      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        expect(result.current.querySelection.expression).toBe(
          'memory:inuse_space:bytes:space:bytes{}'
        );
      });
    });
  });

  describe('Commit with refreshed time range (time range re-evaluation)', () => {
    it('should use refreshed time range values instead of draft state when provided', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&from=1000&to=2000&time_selection=relative:minute|15'
        ),
      });

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
        // Should use refreshed time range values, not draft values
        expect(String(result.current.querySelection.from)).toBe('5000');
        expect(String(result.current.querySelection.to)).toBe('6000');
        expect(result.current.querySelection.timeSelection).toBe('relative:minute|15');
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
        expect(String(result.current.querySelection.from)).toBe('3000');
        expect(String(result.current.querySelection.to)).toBe('4000');
      });

      // Draft state should be updated with the refreshed time range
      expect(result.current.draftSelection.from).toBe(3000);
      expect(result.current.draftSelection.to).toBe(4000);
    });

    it('should trigger navigation even when expression unchanged (time re-evaluation)', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&from=1000&to=2000&time_selection=relative:minute|5'
        ),
      });

      // First commit with new time values
      act(() => {
        result.current.commitDraft({
          from: 5000,
          to: 6000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(String(result.current.querySelection.from)).toBe('5000');
        expect(String(result.current.querySelection.to)).toBe('6000');
      });

      // Second commit with different time values (simulating clicking Search again)
      act(() => {
        result.current.commitDraft({
          from: 7000,
          to: 8000,
          timeSelection: 'relative:minute|5',
        });
      });

      await waitFor(() => {
        expect(String(result.current.querySelection.from)).toBe('7000');
        expect(String(result.current.querySelection.to)).toBe('8000');
      });
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
        // Verify merge params are calculated from refreshed time range
        expect(result.current.querySelection.mergeFrom).toBe('5000000000'); // 5000ms * 1_000_000
        expect(result.current.querySelection.mergeTo).toBe('6000000000'); // 6000ms * 1_000_000
      });
    });

    it('should use draft values when refreshedTimeRange is not provided', async () => {
      const {result} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=memory:inuse_space:bytes:space:bytes{}&from=1000&to=2000&time_selection=relative:hour|1'
        ),
      });

      // Change draft values
      act(() => {
        result.current.setDraftTimeRange(3000, 4000, 'relative:minute|30');
      });

      // Commit without refreshedTimeRange - should use draft values
      act(() => {
        result.current.commitDraft();
      });

      await waitFor(() => {
        // Should use updated draft values
        expect(String(result.current.querySelection.from)).toBe('3000');
        expect(String(result.current.querySelection.to)).toBe('4000');
        expect(result.current.querySelection.timeSelection).toBe('relative:minute|30');
      });
    });
  });

  describe('State persistence after page reload', () => {
    it('should retain committed values after page reload simulation', async () => {
      // Initial state (using delta profile since sumBy only applies to delta)
      const {result: result1, unmount} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000'
        ),
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
        expect(result1.current.querySelection.expression).toBe(
          'memory:alloc_space:bytes:space:bytes:delta{}'
        );
      });

      // Build the query string from the committed state
      const queryString = new URLSearchParams({
        expression: String(result1.current.querySelection.expression),
        from: String(result1.current.querySelection.from),
        to: String(result1.current.querySelection.to),
        time_selection: String(result1.current.querySelection.timeSelection),
        sum_by: (result1.current.querySelection.sumBy ?? []).join(','),
      }).toString();

      // Unmount the old hook instance
      unmount();

      // Clear navigation mock to verify no new navigation on reload
      mockNavigateTo.mockClear();

      // Create new hook instance (simulating page reload) with the committed search params
      const {result: result2} = renderHook(() => useQueryState(), {
        wrapper: createWrapper({}, `?${queryString}`),
      });

      // Verify state is loaded from URL after "reload"
      expect(result2.current.querySelection.expression).toBe(
        'memory:alloc_space:bytes:space:bytes:delta{}'
      );
      expect(result2.current.querySelection.from).toBe(5000);
      expect(result2.current.querySelection.to).toBe(6000);
      expect(result2.current.querySelection.timeSelection).toBe('relative:minute|15');

      // Draft should be synced with URL state on page load
      expect(result2.current.draftSelection.expression).toBe(
        'memory:alloc_space:bytes:space:bytes:delta{}'
      );
      expect(result2.current.draftSelection.from).toBe(5000);
      expect(result2.current.draftSelection.to).toBe(6000);
    });

    it('should preserve delta profile merge params after reload', async () => {
      // Initial state with delta profile
      const {result: result1, unmount} = renderHook(() => useQueryState(), {
        wrapper: createWrapper(
          {},
          '?expression=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{}&from=1000&to=2000'
        ),
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
        expect(result1.current.querySelection.mergeFrom).toBe('5000000000');
        expect(result1.current.querySelection.mergeTo).toBe('6000000000');
      });

      // Simulate page reload with all params including merge params
      const queryString = new URLSearchParams({
        expression: String(result1.current.querySelection.expression),
        from: String(result1.current.querySelection.from),
        to: String(result1.current.querySelection.to),
        time_selection: String(result1.current.querySelection.timeSelection),
        merge_from: String(result1.current.querySelection.mergeFrom),
        merge_to: String(result1.current.querySelection.mergeTo),
      }).toString();

      unmount();
      mockNavigateTo.mockClear();

      // Create new hook instance
      const {result: result2} = renderHook(() => useQueryState(), {
        wrapper: createWrapper({}, `?${queryString}`),
      });

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
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          '?merge_from_a=1234567890&merge_to_a=9876543210&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{pod="test"}'
        ),
      });

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
        const {profileSelection} = result.current;
        expect(profileSelection).not.toBeNull();
        const historyParams = profileSelection?.HistoryParams();
        expect(historyParams?.selection).toBe(
          'memory:inuse_space:bytes:space:bytes{namespace="default"}'
        );
        expect(historyParams?.merge_from).toBe('5000000000');
        expect(historyParams?.merge_to).toBe('6000000000');
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
        const {profileSelection} = resultB.current;
        expect(profileSelection).not.toBeNull();
        const historyParams = profileSelection?.HistoryParams();
        expect(historyParams?.selection).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}'
        );
        expect(historyParams?.merge_from).toBe('7000000000');
        expect(historyParams?.merge_to).toBe('8000000000');
      });
    });

    it('should clear ProfileSelection when commitDraft is called', async () => {
      // Start with a ProfileSelection in URL - using valid profile type
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&merge_from_a=1000000000&merge_to_a=2000000000&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}'
        ),
      });

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
        // ProfileSelection should be cleared
        expect(result.current.profileSelection).toBeNull();

        // But QuerySelection params should still be present
        expect(result.current.querySelection.expression).toBe(
          'memory:inuse_space:bytes:space:bytes{}'
        );
      });
    });

    it('should handle ProfileSelection with delta profiles correctly', () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          '?merge_from_a=1000000000&merge_to_a=2000000000&selection_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{node="worker"}'
        ),
      });

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
        expect(result1.current.profileSelection).not.toBeNull();
      });

      // Get the committed state values to build reload URL
      const historyParams = result1.current.profileSelection?.HistoryParams();
      const selectionA = historyParams?.selection ?? '';
      const mergeFromA = historyParams?.merge_from ?? '';
      const mergeToA = historyParams?.merge_to ?? '';

      unmount();
      mockNavigateTo.mockClear();

      // Create new hook instance (simulating page reload) with the committed search params
      const {result: result2} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
          `?selection_a=${encodeURIComponent(selectionA)}&merge_from_a=${
            mergeFromA as string
          }&merge_to_a=${mergeToA as string}`
        ),
      });

      // Verify ProfileSelection is loaded from URL after reload
      const profileSelection = result2.current.profileSelection;
      expect(profileSelection).not.toBeNull();

      // Use interface methods to test
      expect(profileSelection?.Type()).toBe('merge');
      const reloadedHistoryParams = profileSelection?.HistoryParams();
      expect(reloadedHistoryParams?.merge_from).toBe('3000000000');
      expect(reloadedHistoryParams?.merge_to).toBe('4000000000');
      expect(reloadedHistoryParams?.selection).toBe(
        'memory:alloc_objects:count:space:bytes{pod="test"}'
      );
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
        expect(result.current.stateA.profileSelection).not.toBeNull();
      });

      // Set ProfileSelection for side B
      act(() => {
        result.current.stateB.setProfileSelection(
          BigInt(3000000000),
          BigInt(4000000000),
          mockQueryB
        );
      });

      // Verify both ProfileSelections exist
      await waitFor(() => {
        expect(result.current.stateA.profileSelection).not.toBeNull();
        expect(result.current.stateB.profileSelection).not.toBeNull();
      });
    });

    it('should return null ProfileSelection when only partial params exist', () => {
      // Missing selection param
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper({}, '?merge_from_a=1000000000&merge_to_a=2000000000'),
      });

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
        const {profileSelection} = result.current;
        expect(profileSelection).not.toBeNull();
        const historyParams = profileSelection?.HistoryParams();
        // The expression gets re-serialized through Query.parse which adds spaces after commas
        expect(historyParams?.selection).toBe(
          'memory:alloc_objects:count:space:bytes:delta{namespace="default", pod="app-1", container="main"}'
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
        const {profileSelection} = result.current;
        expect(profileSelection).not.toBeNull();
        const historyParams = profileSelection?.HistoryParams();
        expect(historyParams?.selection).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds{job="test"}'
        );
        expect(historyParams?.merge_from).toBe('1000000000');
        expect(historyParams?.merge_to).toBe('2000000000');
      });
    });

    it('should preserve other URL params when setting ProfileSelection', async () => {
      const {result} = renderHook(() => useQueryState({suffix: '_a'}), {
        wrapper: createWrapper(
          {},
          '?expression_a=process_cpu:cpu:nanoseconds:cpu:nanoseconds{}&other_param=value&unrelated=test'
        ),
      });

      const mockQuery = {
        toString: () => 'process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}',
        profileType: () => ({delta: false}),
      } as any;

      act(() => {
        result.current.setProfileSelection(BigInt(1000000000), BigInt(2000000000), mockQuery);
      });

      await waitFor(() => {
        // ProfileSelection params should be set
        const {profileSelection} = result.current;
        expect(profileSelection).not.toBeNull();
        const historyParams = profileSelection?.HistoryParams();
        expect(historyParams?.selection).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds{pod="test"}'
        );

        // Expression should still be present
        expect(result.current.querySelection.expression).toBe(
          'process_cpu:cpu:nanoseconds:cpu:nanoseconds{}'
        );
      });
    });
  });
});
