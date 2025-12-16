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

import {ReactNode} from 'react';

// eslint-disable-next-line import/named
import {act, renderHook, waitFor} from '@testing-library/react';
import {beforeEach, describe, expect, it, vi} from 'vitest';

import {
  JSONParser,
  JSONSerializer,
  URLStateProvider,
  hasQueryParams,
  useURLState,
  useURLStateBatch,
  useURLStateCustom,
  useURLStateReset,
} from './index';

// Mock the navigate function
const mockNavigateTo = vi.fn();

// Mock window.location
const mockLocation = {
  pathname: '/test',
  search: '',
};

// Mock the getQueryParamsFromURL function to parse our mock search string
vi.mock('./utils', async () => {
  const actual = await vi.importActual('./utils');
  return {
    ...actual,
    getQueryParamsFromURL: () => {
      if (mockLocation.search === '') return {};
      const params = new URLSearchParams(mockLocation.search);
      const result: Record<string, string | string[]> = {};
      for (const [key, value] of params.entries()) {
        // Handle decoding
        const decodedValue = decodeURIComponent(value);
        const existing = result[key];
        if (existing !== undefined) {
          // Convert to array if multiple values
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
const createWrapper = (
  paramPreferences = {}
): (({children}: {children: ReactNode}) => JSX.Element) => {
  const Wrapper = ({children}: {children: ReactNode}): JSX.Element => (
    <URLStateProvider navigateTo={mockNavigateTo} paramPreferences={paramPreferences}>
      {children}
    </URLStateProvider>
  );
  Wrapper.displayName = 'URLStateProviderWrapper';
  return Wrapper;
};

describe('URLState Hooks', () => {
  beforeEach(() => {
    // Reset mocks before each test
    mockNavigateTo.mockClear();

    // Mock window.location
    Object.defineProperty(window, 'location', {
      value: mockLocation,
      writable: true,
    });

    // Reset search params
    mockLocation.search = '';
  });

  describe('useURLState', () => {
    it('should initialize with default value when no URL param exists', () => {
      const {result} = renderHook(() => useURLState('testParam', {defaultValue: 'defaultValue'}), {
        wrapper: createWrapper(),
      });

      const [value] = result.current;
      expect(value).toBe('defaultValue');
    });

    it('should update state and trigger URL navigation on setter call', async () => {
      const {result} = renderHook(() => useURLState('testParam'), {wrapper: createWrapper()});

      const [, setParam] = result.current;

      act(() => {
        setParam('newValue');
      });

      // Wait for the microtask to complete
      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {testParam: 'newValue'},
          {replace: true}
        );
      });

      // Check that state is updated
      const [value] = result.current;
      expect(value).toBe('newValue');
    });

    it('should handle array values correctly', () => {
      const {result} = renderHook(
        () =>
          useURLState<string[]>('tags', {
            defaultValue: ['tag1', 'tag2'],
            alwaysReturnArray: true,
          }),
        {wrapper: createWrapper()}
      );

      const [value] = result.current;
      expect(value).toEqual(['tag1', 'tag2']);
    });

    it('should return single value when array has one item and alwaysReturnArray is false', () => {
      mockLocation.search = '?item=single';

      const {result} = renderHook(() => useURLState('item', {alwaysReturnArray: false}), {
        wrapper: createWrapper(),
      });

      const [value] = result.current;
      expect(value).toBe('single');
    });

    it('should always return array when alwaysReturnArray is true', () => {
      // Set up initial state with a single string value
      mockLocation.search = '';

      const {result} = renderHook(
        () =>
          useURLState<string[]>('item', {
            defaultValue: ['single'],
            alwaysReturnArray: true,
          }),
        {wrapper: createWrapper()}
      );

      const [value] = result.current;
      expect(value).toEqual(['single']);
    });
  });

  describe('useURLStateBatch', () => {
    it('should batch multiple state updates into a single URL navigation', async () => {
      // Create a test component that uses multiple URL states
      const TestComponent = (): {
        color: string | string[] | undefined;
        size: string | string[] | undefined;
        setColor: (val: string | string[] | undefined) => void;
        setSize: (val: string | string[] | undefined) => void;
        batchUpdates: (callback: () => void) => void;
      } => {
        const [color, setColor] = useURLState('color');
        const [size, setSize] = useURLState('size');
        const batchUpdates = useURLStateBatch();

        return {
          color,
          size,
          setColor,
          setSize,
          batchUpdates,
        };
      };

      const {result} = renderHook(() => TestComponent(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.batchUpdates(() => {
          result.current.setColor('red');
          result.current.setSize('large');
        });
      });

      // Wait for the batch to complete
      await waitFor(() => {
        // Should only navigate once with both parameters
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {color: 'red', size: 'large'},
          {replace: true}
        );
      });

      // Check that both states are updated
      expect(result.current.color).toBe('red');
      expect(result.current.size).toBe('large');
    });

    it('should handle nested batch updates correctly - multiple levels of nesting', async () => {
      // This test simulates real-world scenarios like toggleGroupBy calling resetFlameGraphState,
      // where both functions use batchUpdates, testing 2 levels of nesting
      // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
      const TestComponent = () => {
        const [param1, setParam1] = useURLState('param1');
        const [param2, setParam2] = useURLState('param2');
        const [param3, setParam3] = useURLState('param3');
        const [param4, setParam4] = useURLState('param4');
        const [param5, setParam5] = useURLState('param5');
        const [param6, setParam6] = useURLState('param6');
        const batchUpdates = useURLStateBatch();

        // Level 2 nesting - deepest function
        // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
        const deeplyNestedFunction = () => {
          batchUpdates(() => {
            setParam5('value5');
            setParam6('value6');
          });
        };

        // Level 1 nesting - calls another batched function
        // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
        const innerBatchedFunction = () => {
          batchUpdates(() => {
            setParam3('value3');
            setParam4('value4');
            // Call another batched function
            deeplyNestedFunction();
          });
        };

        return {
          param1,
          param2,
          param3,
          param4,
          param5,
          param6,
          setParam1,
          setParam2,
          innerBatchedFunction,
          batchUpdates,
        };
      };

      const {result} = renderHook(() => TestComponent(), {
        wrapper: createWrapper(),
      });

      // Outer batchUpdates that calls nested functions which also use batchUpdates
      act(() => {
        result.current.batchUpdates(() => {
          result.current.setParam1('value1');
          result.current.setParam2('value2');
          // This calls another function that internally uses batchUpdates
          // which in turn calls another function that also uses batchUpdates
          result.current.innerBatchedFunction();
        });
      });

      await waitFor(() => {
        // Critical: Should only navigate ONCE even with 2 levels of nested batchUpdates
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        // All parameters from outer, inner, and deeply nested batches should be in single navigation
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            param1: 'value1',
            param2: 'value2',
            param3: 'value3',
            param4: 'value4',
            param5: 'value5',
            param6: 'value6',
          },
          {replace: true}
        );
      });

      // Verify all state is updated correctly
      expect(result.current.param1).toBe('value1');
      expect(result.current.param2).toBe('value2');
      expect(result.current.param3).toBe('value3');
      expect(result.current.param4).toBe('value4');
      expect(result.current.param5).toBe('value5');
      expect(result.current.param6).toBe('value6');
    });
  });

  describe('useURLStateCustom', () => {
    it('should parse and stringify custom data types', () => {
      const customData = {foo: 'bar', count: 42};

      const {result} = renderHook(
        () =>
          useURLStateCustom('customData', {
            parse: JSONParser,
            stringify: JSONSerializer,
            defaultValue: JSON.stringify(customData),
          }),
        {wrapper: createWrapper()}
      );

      const [value] = result.current;
      expect(value).toEqual(customData);
    });

    it('should handle custom serialization for complex objects', async () => {
      const {result} = renderHook(
        () =>
          useURLStateCustom<{items: string[]; enabled: boolean}>('config', {
            parse: JSONParser,
            stringify: JSONSerializer,
          }),
        {wrapper: createWrapper()}
      );

      const [, setConfig] = result.current;

      act(() => {
        setConfig({items: ['a', 'b', 'c'], enabled: true});
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {config: '{"items":["a","b","c"],"enabled":true}'},
          {replace: true}
        );
      });
    });
  });

  describe('Real-world use cases', () => {
    it('should handle dashboard panel management', async () => {
      // Simulate ViewSelector component behavior
      const {result: dashboardResult} = renderHook(
        () =>
          useURLState<string[]>('dashboard_items', {
            defaultValue: ['flamegraph'],
            alwaysReturnArray: true,
          }),
        {wrapper: createWrapper()}
      );

      const [dashboardItems, setDashboardItems] = dashboardResult.current;
      expect(dashboardItems).toEqual(['flamegraph']);

      // Add a new panel
      act(() => {
        setDashboardItems([...dashboardItems, 'table']);
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {dashboard_items: 'flamegraph,table'},
          {replace: true}
        );
      });
    });

    it('should handle complex filter updates with batching', async () => {
      // Simulate ProfileSelector component behavior
      // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
      const TestComponent = () => {
        const [colorBy, setColorBy] = useURLState('color_by', {defaultValue: 'function'});
        const [groupBy, setGroupBy] = useURLState<string[]>('group_by', {
          defaultValue: ['function_name'],
          alwaysReturnArray: true,
        });
        const batchUpdates = useURLStateBatch();

        return {
          colorBy,
          groupBy,
          setColorBy,
          setGroupBy,
          batchUpdates,
        };
      };

      const {result} = renderHook(() => TestComponent(), {
        wrapper: createWrapper(),
      });

      // Simulate a complex filter change that updates multiple params
      act(() => {
        result.current.batchUpdates(() => {
          result.current.setColorBy('filename');
          result.current.setGroupBy(['function_name', 'filename']);
        });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            color_by: 'filename',
            group_by: 'function_name,filename',
          },
          {replace: true}
        );
      });
    });

    it('should not update URL for default values', async () => {
      const paramPreferences = {
        view: {defaultValue: 'flamegraph'},
        sort: {defaultValue: 'cumulative'},
      };

      const {result} = renderHook(() => useURLState('view'), {
        wrapper: createWrapper(paramPreferences),
      });

      const [, setView] = result.current;

      // Set to default value
      act(() => {
        setView('flamegraph');
      });

      await waitFor(() => {
        // Should still be called but with empty params (sanitized)
        expect(mockNavigateTo).toHaveBeenCalledWith('/test', {}, {replace: true});
      });
    });

    it('should handle rapid successive updates', async () => {
      const {result} = renderHook(() => useURLState('rapidParam'), {wrapper: createWrapper()});

      const [, setParam] = result.current;

      // Rapid successive updates
      act(() => {
        setParam('value1');
        setParam('value2');
        setParam('value3');
      });

      await waitFor(() => {
        // Due to the setTimeout(0) debouncing, we expect the last value
        expect(mockNavigateTo).toHaveBeenLastCalledWith(
          '/test',
          {rapidParam: 'value3'},
          {replace: true}
        );
      });
    });
  });

  describe('URL Parameter Preservation', () => {
    it('should preserve other query parameters when resetting specific ones', async () => {
      // Simulate existing URL parameters
      mockLocation.search =
        '?expression_a=process_cpu%7B%7D&from_a=1234567890&to_a=9876543210&time_selection_a=1h&group_by=existing_group&cur_path=/existing/path';

      // eslint-disable-next-line @typescript-eslint/explicit-function-return-type
      const TestComponent = () => {
        const [groupBy, setGroupBy] = useURLState('group_by');
        const [curPath, setCurPath] = useURLState('cur_path');
        const [expression] = useURLState('expression_a');
        const [from] = useURLState('from_a');
        const batchUpdates = useURLStateBatch();

        return {
          groupBy,
          curPath,
          expression,
          from,
          resetProfileTypeState: () => {
            batchUpdates(() => {
              setGroupBy(undefined);
              setCurPath(undefined);
            });
          },
        };
      };

      const {result} = renderHook(() => TestComponent(), {
        wrapper: createWrapper(),
      });

      // Verify initial values
      expect(result.current.expression).toBe('process_cpu{}');
      expect(result.current.from).toBe('1234567890');

      // Perform the reset
      act(() => {
        result.current.resetProfileTypeState();
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledTimes(1);
        const [, params] = mockNavigateTo.mock.calls[0];

        // Critical: Check that query parameters are preserved
        expect(params).toHaveProperty('expression_a', 'process_cpu{}');
        expect(params).toHaveProperty('from_a', '1234567890');
        expect(params).toHaveProperty('to_a', '9876543210');
        expect(params).toHaveProperty('time_selection_a', '1h');

        // These should be removed
        expect(params).not.toHaveProperty('group_by');
        expect(params).not.toHaveProperty('cur_path');
      });
    });

    it('should preserve unmanaged parameters during single state updates', async () => {
      // Set up URL with both managed and unmanaged parameters
      mockLocation.search =
        '?managed=old_value&unmanaged=should_persist&another_unmanaged=also_persists';

      const {result} = renderHook(() => useURLState('managed'), {wrapper: createWrapper()});

      const [, setManaged] = result.current;

      // Update only the managed parameter
      act(() => {
        setManaged('new_value');
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            managed: 'new_value',
            unmanaged: 'should_persist',
            another_unmanaged: 'also_persists',
          },
          {replace: true}
        );
      });
    });

    it('should preserve unmanaged parameters when adding new state', async () => {
      // Start with some unmanaged parameters in URL
      mockLocation.search = '?existing_param=value1&another_param=value2';

      const {result} = renderHook(() => useURLState('new_param'), {wrapper: createWrapper()});

      const [, setNewParam] = result.current;

      // Add a new parameter
      act(() => {
        setNewParam('new_value');
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            existing_param: 'value1',
            another_param: 'value2',
            new_param: 'new_value',
          },
          {replace: true}
        );
      });
    });

    it('should handle complex nested objects in unmanaged parameters', async () => {
      // Simulate URL with JSON-encoded objects
      mockLocation.search = '?filter=%7B%22type%22%3A%22cpu%22%2C%22value%22%3A100%7D&managed=test';

      const {result} = renderHook(() => useURLState('managed'), {wrapper: createWrapper()});

      const [, setManaged] = result.current;

      act(() => {
        setManaged('updated');
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            filter: '{"type":"cpu","value":100}',
            managed: 'updated',
          },
          {replace: true}
        );
      });
    });

    it('should preserve array parameters not managed by hooks', async () => {
      // URL with array parameters - note that our mock getQueryParamsFromURL processes these
      mockLocation.search = '?tags=tag1&tags=tag2&tags=tag3&managed=value';

      const {result} = renderHook(() => useURLState('managed'), {wrapper: createWrapper()});

      const [, setManaged] = result.current;

      act(() => {
        setManaged('new_value');
      });

      await waitFor(() => {
        const [, params] = mockNavigateTo.mock.calls[0];
        // Tags are preserved (the sanitize function converts arrays to comma-separated strings)
        expect(params.tags).toBe('tag1,tag2,tag3');
        expect(params.managed).toBe('new_value');
      });
    });
  });

  describe('Error handling', () => {
    it('should throw error when used outside URLStateProvider', () => {
      // Suppress console.error for this test
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        renderHook(() => useURLState('param'));
      }).toThrow('useURLState must be used within a URLStateProvider');

      consoleSpy.mockRestore();
    });

    it('should throw error for useURLStateBatch when used outside URLStateProvider', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        renderHook(() => useURLStateBatch());
      }).toThrow('useURLStateBatch must be used within a URLStateProvider');

      consoleSpy.mockRestore();
    });
  });

  describe('mergeStrategy option', () => {
    describe('replace strategy (default)', () => {
      it('should replace existing value when no mergeStrategy is specified', async () => {
        const {result} = renderHook(() => useURLState('param'), {wrapper: createWrapper()});

        const [, setParam] = result.current;

        // Set initial value
        act(() => {
          setParam('initial');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('initial');
        });

        // Replace with new value
        act(() => {
          setParam('replaced');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('replaced');
          expect(mockNavigateTo).toHaveBeenLastCalledWith(
            '/test',
            {param: 'replaced'},
            {replace: true}
          );
        });
      });

      it('should replace existing value when mergeStrategy is "replace"', async () => {
        const {result} = renderHook(() => useURLState('param', {mergeStrategy: 'replace'}), {
          wrapper: createWrapper(),
        });

        const [, setParam] = result.current;

        act(() => {
          setParam('initial');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('initial');
        });

        act(() => {
          setParam('replaced');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('replaced');
        });
      });

      it('should replace array with string using replace strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'replace'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        act(() => {
          setParam(['one', 'two']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['one', 'two']);
        });

        act(() => {
          setParam('single');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('single');
        });
      });
    });

    describe('preserve-existing strategy', () => {
      it('should not overwrite existing value with preserve-existing strategy', async () => {
        const {result} = renderHook(
          () => useURLState('param', {mergeStrategy: 'preserve-existing'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial value
        act(() => {
          setParam('existing');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('existing');
        });

        // Try to set new value - should be ignored
        act(() => {
          setParam('should-be-ignored');
        });

        await waitFor(() => {
          // Value should remain unchanged
          expect(result.current[0]).toBe('existing');
        });
      });

      it('should set value when current is undefined with preserve-existing', async () => {
        const {result} = renderHook(
          () => useURLState('param', {mergeStrategy: 'preserve-existing'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set value when undefined
        act(() => {
          setParam('new-value');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('new-value');
          expect(mockNavigateTo).toHaveBeenCalledWith(
            '/test',
            {param: 'new-value'},
            {replace: true}
          );
        });
      });

      it('should set value when current is empty string with preserve-existing', async () => {
        const {result} = renderHook(
          () => useURLState('param', {mergeStrategy: 'preserve-existing'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set to empty string first
        act(() => {
          setParam('');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('');
        });

        // Should overwrite empty string
        act(() => {
          setParam('new-value');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('new-value');
        });
      });

      it('should set value when current is empty array with preserve-existing', async () => {
        const {result} = renderHook(
          () =>
            useURLState<string[]>('param', {
              mergeStrategy: 'preserve-existing',
              alwaysReturnArray: true,
            }),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set to empty array first
        act(() => {
          setParam([]);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual([]);
        });

        // Should overwrite empty array
        act(() => {
          setParam(['value']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['value']);
        });
      });

      it('should preserve existing array with preserve-existing strategy', async () => {
        const {result} = renderHook(
          () =>
            useURLState<string[]>('param', {
              mergeStrategy: 'preserve-existing',
              alwaysReturnArray: true,
            }),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial array
        act(() => {
          setParam(['existing']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['existing']);
        });

        // Try to set new array - should be ignored
        act(() => {
          setParam(['new']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['existing']);
        });
      });
    });

    describe('append strategy', () => {
      it('should ignore undefined/null values with append strategy', async () => {
        const {result} = renderHook(() => useURLState('param', {mergeStrategy: 'append'}), {
          wrapper: createWrapper(),
        });

        const [, setParam] = result.current;

        // Set initial value
        act(() => {
          setParam('existing');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('existing');
        });

        // Try to append undefined - should be ignored
        act(() => {
          setParam(undefined);
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('existing');
        });
      });

      it('should merge two arrays and deduplicate with append strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial array
        act(() => {
          setParam(['a', 'b']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b']);
        });

        // Append array with overlap
        act(() => {
          setParam(['b', 'c', 'd']);
        });

        await waitFor(() => {
          // Should deduplicate 'b'
          expect(result.current[0]).toEqual(['a', 'b', 'c', 'd']);
        });
      });

      it('should add string to array with append strategy (no duplicates)', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial array
        act(() => {
          setParam(['a', 'b']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b']);
        });

        // Append new string
        act(() => {
          setParam('c');
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b', 'c']);
        });
      });

      it('should not add duplicate string to array with append strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial array
        act(() => {
          setParam(['a', 'b']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b']);
        });

        // Try to append existing string
        act(() => {
          setParam('b');
        });

        await waitFor(() => {
          // Should remain unchanged (no duplicate)
          expect(result.current[0]).toEqual(['a', 'b']);
        });
      });

      it('should merge string with array with append strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial string
        act(() => {
          setParam('a');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('a');
        });

        // Append array
        act(() => {
          setParam(['b', 'c']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b', 'c']);
        });
      });

      it('should create array from two different strings with append strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial string
        act(() => {
          setParam('first');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('first');
        });

        // Append different string
        act(() => {
          setParam('second');
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['first', 'second']);
        });
      });

      it('should not create array when appending same string with append strategy', async () => {
        const {result} = renderHook(
          () => useURLState<string | string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial string
        act(() => {
          setParam('same');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('same');
        });

        // Append same string (should deduplicate)
        act(() => {
          setParam('same');
        });

        await waitFor(() => {
          // Should remain a single string, not create array
          expect(result.current[0]).toBe('same');
        });
      });

      it('should set value when current is empty with append strategy', async () => {
        const {result} = renderHook(() => useURLState('param', {mergeStrategy: 'append'}), {
          wrapper: createWrapper(),
        });

        const [, setParam] = result.current;

        // Append to undefined (should just set)
        act(() => {
          setParam('new-value');
        });

        await waitFor(() => {
          expect(result.current[0]).toBe('new-value');
        });
      });

      it('should deduplicate when merging string array with overlapping values', async () => {
        const {result} = renderHook(
          () => useURLState<string[]>('param', {mergeStrategy: 'append'}),
          {wrapper: createWrapper()}
        );

        const [, setParam] = result.current;

        // Set initial array
        act(() => {
          setParam(['a', 'b', 'c']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['a', 'b', 'c']);
        });

        // Append array with all duplicates and one new value
        act(() => {
          setParam(['a', 'b', 'c', 'd']);
        });

        await waitFor(() => {
          // Should only add 'd'
          expect(result.current[0]).toEqual(['a', 'b', 'c', 'd']);
        });
      });
    });

    describe('Real-world view defaults use case', () => {
      it('should apply view defaults only when URL params are empty (preserve-existing)', async () => {
        // Simulate view defaults being applied
        const {result} = renderHook(
          () =>
            useURLState('group_by', {
              defaultValue: ['function_name'],
              mergeStrategy: 'preserve-existing',
              alwaysReturnArray: true,
            }),
          {wrapper: createWrapper()}
        );

        // Initial render - should use default
        expect(result.current[0]).toEqual(['function_name']);

        const [, setGroupBy] = result.current;

        // User modifies the value
        act(() => {
          setGroupBy(['custom_label']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['custom_label']);
        });

        // Simulate view switching trying to apply defaults again (should be ignored)
        act(() => {
          setGroupBy(['function_name']);
        });

        await waitFor(() => {
          // Should keep user's custom value
          expect(result.current[0]).toEqual(['custom_label']);
        });
      });

      it('should accumulate filter values with append strategy', async () => {
        // Simulate adding multiple filters
        const {result} = renderHook(
          () =>
            useURLState<string[]>('filters', {mergeStrategy: 'append', alwaysReturnArray: true}),
          {wrapper: createWrapper()}
        );

        const [, setFilters] = result.current;

        // Add first filter
        act(() => {
          setFilters(['cpu>50']);
        });

        await waitFor(() => {
          expect(result.current[0]).toEqual(['cpu>50']);
        });

        // Add second filter
        act(() => {
          setFilters(['memory<1000']);
        });

        await waitFor(() => {
          // Should append, not replace
          expect(result.current[0]).toEqual(['cpu>50', 'memory<1000']);
        });

        // Try to add duplicate filter
        act(() => {
          setFilters(['cpu>50']);
        });

        await waitFor(() => {
          // Should not add duplicate
          expect(result.current[0]).toEqual(['cpu>50', 'memory<1000']);
        });
      });
    });

    describe('enabled option', () => {
      it('should return undefined and no-op setter when enabled is false', async () => {
        const {result} = renderHook(() => useURLState('param', {enabled: false}), {
          wrapper: createWrapper(),
        });

        const [value, setParam] = result.current;
        expect(value).toBeUndefined();

        // Try to set value - should be no-op
        act(() => {
          setParam('should-not-work');
        });

        await waitFor(() => {
          expect(mockNavigateTo).not.toHaveBeenCalled();
        });
      });

      it('should handle compare mode group_by use case', async () => {
        const TestComponent = (): {
          groupByA: string | string[] | undefined;
          groupByB: string | string[] | undefined;
        } => {
          const [groupByA] = useURLState('group_by', {enabled: true, defaultValue: ['node']});
          const [groupByB] = useURLState('group_by', {enabled: false});
          return {groupByA, groupByB};
        };

        const {result} = renderHook(() => TestComponent(), {wrapper: createWrapper()});

        expect(result.current.groupByA).toEqual(['node']);
        expect(result.current.groupByB).toBeUndefined();
      });
    });

    describe('namespace option', () => {
      it('should prefix param name with namespace', async () => {
        const {result} = renderHook(
          () => useURLState('setting', {namespace: 'view', defaultValue: 'default'}),
          {wrapper: createWrapper()}
        );

        const [, setSetting] = result.current;

        act(() => {
          setSetting('new-value');
        });

        await waitFor(() => {
          expect(mockNavigateTo).toHaveBeenCalledWith(
            '/test',
            {'view.setting': 'new-value'},
            {replace: true}
          );
        });
      });

      it('should allow multiple namespaces without conflict', async () => {
        const TestComponent = (): {
          setViewColor: (val: string | string[] | undefined) => void;
          setAppColor: (val: string | string[] | undefined) => void;
        } => {
          const [, setViewColor] = useURLState('color', {namespace: 'view'});
          const [, setAppColor] = useURLState('color', {namespace: 'app'});
          return {setViewColor, setAppColor};
        };

        const {result} = renderHook(() => TestComponent(), {wrapper: createWrapper()});

        act(() => {
          result.current.setViewColor('red');
          result.current.setAppColor('blue');
        });

        await waitFor(() => {
          expect(mockNavigateTo).toHaveBeenLastCalledWith(
            '/test',
            {'view.color': 'red', 'app.color': 'blue'},
            {replace: true}
          );
        });
      });
    });

    describe('useURLStateReset', () => {
      it('should clear specified keys and preserve others', async () => {
        mockLocation.search = '?param1=value1&param2=value2&param3=value3';

        const TestComponent = (): {
          reset: (keys: string[]) => void;
        } => {
          const reset = useURLStateReset();
          return {reset};
        };

        const {result} = renderHook(() => TestComponent(), {wrapper: createWrapper()});

        act(() => {
          result.current.reset(['param1', 'param2']);
        });

        await waitFor(() => {
          expect(mockNavigateTo).toHaveBeenCalledWith(
            '/test',
            {param1: undefined, param2: undefined, param3: 'value3'},
            {replace: true}
          );
        });
      });

      it('should throw error when used outside URLStateProvider', () => {
        const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

        expect(() => {
          renderHook(() => useURLStateReset());
        }).toThrow('useURLStateReset must be used within a URLStateProvider');

        consoleSpy.mockRestore();
      });
    });

    describe('hasQueryParams helper', () => {
      it('should return true/false based on params existence', () => {
        expect(hasQueryParams({param1: 'value1'})).toBe(true);
        expect(hasQueryParams({})).toBe(false);
        expect(hasQueryParams({param1: undefined})).toBe(false);
        expect(hasQueryParams({param1: ''})).toBe(false);
      });

      it('should exclude specified keys', () => {
        const state = {routeParam: 'value1', queryParam: 'value2'};
        expect(hasQueryParams(state, ['routeParam'])).toBe(true); // queryParam exists
        expect(hasQueryParams(state, ['routeParam', 'queryParam'])).toBe(false); // all excluded
      });

      it('should handle view switching scenario', () => {
        const stateWithoutQuery = {'project-id': 'abc', 'view-slug': 'my-view'};
        expect(hasQueryParams(stateWithoutQuery, ['project-id', 'view-slug'])).toBe(false);

        const stateWithQuery = {...stateWithoutQuery, group_by: ['node']};
        expect(hasQueryParams(stateWithQuery, ['project-id', 'view-slug'])).toBe(true);
      });
    });
  });
});
