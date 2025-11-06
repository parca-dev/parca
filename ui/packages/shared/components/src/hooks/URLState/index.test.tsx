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

import { ReactNode } from 'react';
import { renderHook, act, waitFor } from '@testing-library/react';
import { describe, expect, it, beforeEach, vi } from 'vitest';

import { URLStateProvider, useURLState, useURLStateBatch, useURLStateCustom, JSONParser, JSONSerializer } from './index';

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
      if (!mockLocation.search) return {};
      const params = new URLSearchParams(mockLocation.search);
      const result: Record<string, string | string[]> = {};
      for (const [key, value] of params.entries()) {
        // Handle decoding
        const decodedValue = decodeURIComponent(value);
        if (result[key]) {
          // Convert to array if multiple values
          result[key] = Array.isArray(result[key])
            ? [...(result[key] as string[]), decodedValue]
            : [result[key] as string, decodedValue];
        } else {
          result[key] = decodedValue;
        }
      }
      return result;
    },
  };
});

// Helper to create wrapper with URLStateProvider
const createWrapper = (paramPreferences = {}) => {
  return ({ children }: { children: ReactNode }) => (
    <URLStateProvider navigateTo={mockNavigateTo} paramPreferences={paramPreferences}>
      {children}
    </URLStateProvider>
  );
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
      const { result } = renderHook(
        () => useURLState('testParam', { defaultValue: 'defaultValue' }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('defaultValue');
    });

    it('should update state and trigger URL navigation on setter call', async () => {
      const { result } = renderHook(
        () => useURLState('testParam'),
        { wrapper: createWrapper() }
      );

      const [, setParam] = result.current;

      act(() => {
        setParam('newValue');
      });

      // Wait for the microtask to complete
      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          { testParam: 'newValue' },
          { replace: true }
        );
      });

      // Check that state is updated
      const [value] = result.current;
      expect(value).toBe('newValue');
    });

    it('should handle array values correctly', () => {
      const { result } = renderHook(
        () => useURLState<string[]>('tags', {
          defaultValue: ['tag1', 'tag2'],
          alwaysReturnArray: true
        }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toEqual(['tag1', 'tag2']);
    });

    it('should return single value when array has one item and alwaysReturnArray is false', () => {
      mockLocation.search = '?item=single';

      const { result } = renderHook(
        () => useURLState('item', { alwaysReturnArray: false }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toBe('single');
    });

    it('should always return array when alwaysReturnArray is true', () => {
      // Set up initial state with a single string value
      mockLocation.search = '';

      const { result } = renderHook(
        () => useURLState<string[]>('item', {
          defaultValue: ['single'],
          alwaysReturnArray: true
        }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toEqual(['single']);
    });
  });

  describe('useURLStateBatch', () => {
    it('should batch multiple state updates into a single URL navigation', async () => {
      // Create a test component that uses multiple URL states
      const TestComponent = () => {
        const [color, setColor] = useURLState('color');
        const [size, setSize] = useURLState('size');
        const batchUpdates = useURLStateBatch();

        return {
          color,
          size,
          setColor,
          setSize,
          batchUpdates
        };
      };

      const { result } = renderHook(() => TestComponent(), {
        wrapper: createWrapper()
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
          { color: 'red', size: 'large' },
          { replace: true }
        );
      });

      // Check that both states are updated
      expect(result.current.color).toBe('red');
      expect(result.current.size).toBe('large');
    });

    it('should handle nested batch updates correctly - multiple levels of nesting', async () => {
      // This test simulates real-world scenarios like toggleGroupBy calling resetFlameGraphState,
      // where both functions use batchUpdates, testing 2 levels of nesting
      const TestComponent = () => {
        const [param1, setParam1] = useURLState('param1');
        const [param2, setParam2] = useURLState('param2');
        const [param3, setParam3] = useURLState('param3');
        const [param4, setParam4] = useURLState('param4');
        const [param5, setParam5] = useURLState('param5');
        const [param6, setParam6] = useURLState('param6');
        const batchUpdates = useURLStateBatch();

        // Level 2 nesting - deepest function
        const deeplyNestedFunction = () => {
          batchUpdates(() => {
            setParam5('value5');
            setParam6('value6');
          });
        };

        // Level 1 nesting - calls another batched function
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
          batchUpdates
        };
      };

      const { result } = renderHook(() => TestComponent(), {
        wrapper: createWrapper()
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
            param6: 'value6'
          },
          { replace: true }
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
      const customData = { foo: 'bar', count: 42 };

      const { result } = renderHook(
        () => useURLStateCustom('customData', {
          parse: JSONParser,
          stringify: JSONSerializer,
          defaultValue: JSON.stringify(customData),
        }),
        { wrapper: createWrapper() }
      );

      const [value] = result.current;
      expect(value).toEqual(customData);
    });

    it('should handle custom serialization for complex objects', async () => {
      const { result } = renderHook(
        () => useURLStateCustom<{ items: string[]; enabled: boolean }>('config', {
          parse: JSONParser,
          stringify: JSONSerializer,
        }),
        { wrapper: createWrapper() }
      );

      const [, setConfig] = result.current;

      act(() => {
        setConfig({ items: ['a', 'b', 'c'], enabled: true });
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          { config: '{"items":["a","b","c"],"enabled":true}' },
          { replace: true }
        );
      });
    });
  });

  describe('Real-world use cases', () => {
    it('should handle dashboard panel management', async () => {
      // Simulate ViewSelector component behavior
      const { result: dashboardResult } = renderHook(
        () => useURLState<string[]>('dashboard_items', {
          defaultValue: ['flamegraph'],
          alwaysReturnArray: true
        }),
        { wrapper: createWrapper() }
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
          { dashboard_items: 'flamegraph,table' },
          { replace: true }
        );
      });
    });

    it('should handle complex filter updates with batching', async () => {
      // Simulate ProfileSelector component behavior
      const TestComponent = () => {
        const [colorBy, setColorBy] = useURLState('color_by', { defaultValue: 'function' });
        const [groupBy, setGroupBy] = useURLState<string[]>('group_by', {
          defaultValue: ['function_name'],
          alwaysReturnArray: true
        });
        const batchUpdates = useURLStateBatch();

        return {
          colorBy,
          groupBy,
          setColorBy,
          setGroupBy,
          batchUpdates
        };
      };

      const { result } = renderHook(() => TestComponent(), {
        wrapper: createWrapper()
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
            group_by: 'function_name,filename'
          },
          { replace: true }
        );
      });
    });

    it('should not update URL for default values', async () => {
      const paramPreferences = {
        view: { defaultValue: 'flamegraph' },
        sort: { defaultValue: 'cumulative' },
      };

      const { result } = renderHook(
        () => useURLState('view'),
        { wrapper: createWrapper(paramPreferences) }
      );

      const [, setView] = result.current;

      // Set to default value
      act(() => {
        setView('flamegraph');
      });

      await waitFor(() => {
        // Should still be called but with empty params (sanitized)
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {},
          { replace: true }
        );
      });
    });

    it('should handle rapid successive updates', async () => {
      const { result } = renderHook(
        () => useURLState('rapidParam'),
        { wrapper: createWrapper() }
      );

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
          { rapidParam: 'value3' },
          { replace: true }
        );
      });
    });
  });

  describe('URL Parameter Preservation', () => {
    it('should preserve other query parameters when resetting specific ones', async () => {
      // Simulate existing URL parameters
      mockLocation.search = '?expression_a=process_cpu%7B%7D&from_a=1234567890&to_a=9876543210&time_selection_a=1h&group_by=existing_group&cur_path=/existing/path';

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
          }
        };
      };

      const { result } = renderHook(() => TestComponent(), {
        wrapper: createWrapper()
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
      mockLocation.search = '?managed=old_value&unmanaged=should_persist&another_unmanaged=also_persists';

      const { result } = renderHook(
        () => useURLState('managed'),
        { wrapper: createWrapper() }
      );

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
            another_unmanaged: 'also_persists'
          },
          { replace: true }
        );
      });
    });

    it('should preserve unmanaged parameters when adding new state', async () => {
      // Start with some unmanaged parameters in URL
      mockLocation.search = '?existing_param=value1&another_param=value2';

      const { result } = renderHook(
        () => useURLState('new_param'),
        { wrapper: createWrapper() }
      );

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
            new_param: 'new_value'
          },
          { replace: true }
        );
      });
    });

    it('should handle complex nested objects in unmanaged parameters', async () => {
      // Simulate URL with JSON-encoded objects
      mockLocation.search = '?filter=%7B%22type%22%3A%22cpu%22%2C%22value%22%3A100%7D&managed=test';

      const { result } = renderHook(
        () => useURLState('managed'),
        { wrapper: createWrapper() }
      );

      const [, setManaged] = result.current;

      act(() => {
        setManaged('updated');
      });

      await waitFor(() => {
        expect(mockNavigateTo).toHaveBeenCalledWith(
          '/test',
          {
            filter: '{"type":"cpu","value":100}',
            managed: 'updated'
          },
          { replace: true }
        );
      });
    });

    it('should preserve array parameters not managed by hooks', async () => {
      // URL with array parameters - note that our mock getQueryParamsFromURL processes these
      mockLocation.search = '?tags=tag1&tags=tag2&tags=tag3&managed=value';

      const { result } = renderHook(
        () => useURLState('managed'),
        { wrapper: createWrapper() }
      );

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
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => { });

      expect(() => {
        renderHook(() => useURLState('param'));
      }).toThrow('useURLState must be used within a URLStateProvider');

      consoleSpy.mockRestore();
    });

    it('should throw error for useURLStateBatch when used outside URLStateProvider', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => { });

      expect(() => {
        renderHook(() => useURLStateBatch());
      }).toThrow('useURLStateBatch must be used within a URLStateProvider');

      consoleSpy.mockRestore();
    });
  });
});
