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

import {
  Dispatch,
  ReactNode,
  SetStateAction,
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import {type NavigateFunction} from '@parca/utilities';

import {
  getQueryParamsFromURL,
  sanitize,
  type ParamPreference,
  type ParamPreferences,
  type ParamValue,
} from './utils';

export type ParamValueSetter = (val: ParamValue) => void;
export type {ParamPreferences, ParamPreference};

interface URLState {
  navigateTo: NavigateFunction;
  state: Record<string, string | string[] | undefined>;
  setState: Dispatch<SetStateAction<Record<string, ParamValue>>>;
  paramPreferences: ParamPreferences;
  batchUpdates: (callback: () => void) => void;
}

const URLStateContext = createContext<URLState | undefined>(undefined);

export const URLStateProvider = ({
  children,
  navigateTo,
  paramPreferences = {},
}: {
  children: ReactNode;
  navigateTo: NavigateFunction;
  paramPreferences?: ParamPreferences;
}): JSX.Element => {
  // Extract default values from preferences for backward compatibility
  // TODO(manoj): Check if this backward compatibility support is needed
  const defaultValues = useMemo(() => {
    const defaults: Record<string, ParamValue> = {};
    Object.entries(paramPreferences).forEach(([key, prefs]) => {
      if (prefs.defaultValue !== undefined) {
        defaults[key] = prefs.defaultValue;
      }
    });
    return defaults;
  }, [paramPreferences]);

  const [state, setState] = useState<Record<string, ParamValue>>({
    ...defaultValues,
    ...getQueryParamsFromURL(paramPreferences),
  });

  const isInitialMount = useRef(true);
  const isBatchingRef = useRef(false);
  const batchTimeoutRef = useRef<NodeJS.Timeout>();
  const urlUpdateTimeoutRef = useRef<NodeJS.Timeout>();
  const lastSyncedURLRef = useRef(window.location.search);

  // Sync state from URL when it changes externally (e.g., clicking nav links)
  // Runs on every render of the provider to catch URL changes
  useEffect(() => {
    const currentURL = window.location.search;

    if (currentURL === lastSyncedURLRef.current) {
      return;
    }

    lastSyncedURLRef.current = currentURL;

    const urlParams = getQueryParamsFromURL(paramPreferences);
    const newState = {
      ...defaultValues,
      ...urlParams,
    };
    setState(newState);
  });

  // Track state changes and sync to URL
  useEffect(() => {
    // Skip initial mount to avoid unnecessary navigation as the state was just initialized from URL
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    // If we're batching, don't navigate yet - we'll do it at the end of the batch
    if (isBatchingRef.current) {
      return;
    }

    // Clear any existing timeout
    if (urlUpdateTimeoutRef.current !== undefined) {
      clearTimeout(urlUpdateTimeoutRef.current);
    }

    // Debounce URL updates with a microtask
    urlUpdateTimeoutRef.current = setTimeout(() => {
      // ALWAYS merge with existing URL params to preserve them
      const currentParams = getQueryParamsFromURL(paramPreferences);
      const mergedParams = {...currentParams, ...state};

      const sanitizedParams = sanitize(mergedParams, paramPreferences);
      navigateTo(window.location.pathname, sanitizedParams, {replace: true});

      // Update ref to match the URL we just set (to avoid re-syncing)
      const queryString = new URLSearchParams(sanitizedParams as Record<string, string>).toString();
      lastSyncedURLRef.current = queryString !== '' ? `?${queryString}` : '';
    }, 0);

    return () => {
      if (urlUpdateTimeoutRef.current !== undefined) {
        clearTimeout(urlUpdateTimeoutRef.current);
      }
    };
  }, [state, navigateTo, paramPreferences]);

  // Batch updates function
  const batchUpdates = useCallback(
    (callback: () => void) => {
      // Track if we were already batching before this call (for nested batching)
      const wasAlreadyBatching = isBatchingRef.current;

      isBatchingRef.current = true;

      // Execute all state updates synchronously
      callback();

      // If we were already batching, this is a nested call - don't schedule a new timeout
      // Let the outermost batchUpdates handle the URL navigation
      if (wasAlreadyBatching) {
        return;
      }

      // Clear any existing timeout
      if (batchTimeoutRef.current !== undefined) {
        clearTimeout(batchTimeoutRef.current);
      }

      // Use setState to capture the final state after all updates
      // This ensures we have the latest state including all batched changes
      setState(currentState => {
        // Don't actually change the state, just use this to read the latest value
        // Schedule the batch to complete and trigger URL update
        batchTimeoutRef.current = setTimeout(() => {
          isBatchingRef.current = false;

          // Navigate with the latest state PLUS existing URL params
          // ALWAYS merge with existing URL params to preserve them
          const currentParams = getQueryParamsFromURL(paramPreferences);
          const mergedParams = {...currentParams, ...currentState};

          const sanitizedParams = sanitize(mergedParams, paramPreferences);
          navigateTo(window.location.pathname, sanitizedParams, {replace: true});

          // Update ref to match the URL we just set (to avoid re-syncing)
          const queryString = new URLSearchParams(
            sanitizedParams as Record<string, string>
          ).toString();
          lastSyncedURLRef.current = queryString !== '' ? `?${queryString}` : '';
        }, 0);

        return currentState; // Return unchanged state
      });
    },
    [paramPreferences, navigateTo]
  );

  const contextValue = useMemo(
    () => ({
      navigateTo,
      state,
      setState,
      paramPreferences,
      batchUpdates,
    }),
    [navigateTo, state, setState, paramPreferences, batchUpdates]
  );

  return <URLStateContext.Provider value={contextValue}>{children}</URLStateContext.Provider>;
};

interface Options {
  defaultValue?: string | string[];
  debugLog?: boolean;
  alwaysReturnArray?: boolean;
}

export const useURLState = <T extends ParamValue>(
  param: string,
  _options?: Options
): [T, ParamValueSetter] => {
  const context = useContext(URLStateContext);
  if (context === undefined) {
    throw new Error('useURLState must be used within a URLStateProvider');
  }

  const {debugLog, defaultValue, alwaysReturnArray} = _options ?? {};

  const {state, setState} = context;

  const setParam: ParamValueSetter = useCallback(
    (val: ParamValue) => {
      if (debugLog === true) {
        console.log('useURLState setParam', param, val);
      }

      // Just update state - Provider handles URL sync automatically!
      setState(currentState => ({
        ...currentState,
        [param]: val,
      }));
    },
    [param, setState, debugLog]
  );

  if (debugLog === true) {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    useEffect(() => {
      console.log('useURLState state change', param, state[param]);

      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [state[param]]);
  }

  const value = useMemo<ParamValue>(() => {
    if (typeof state[param] === 'string') {
      if (alwaysReturnArray === true) {
        return [state[param]] as ParamValue;
      }
      return state[param];
    } else if (state[param] != null && Array.isArray(state[param])) {
      if (state[param]?.length === 1 && alwaysReturnArray !== true) {
        return state[param]?.[0] as ParamValue;
      } else {
        return state[param];
      }
    }
  }, [state, param, alwaysReturnArray]);

  return [(value ?? defaultValue) as T, setParam];
};

export interface OptionsCustom<T> {
  parse: (val: ParamValue) => T;
  stringify: (val: T) => ParamValue;
}

export type ParamValueSetterCustom<T> = (val: T) => void;

export const useURLStateCustom = <T,>(
  param: string,
  {parse, stringify, ..._options}: Options & OptionsCustom<T>
): [T, ParamValueSetterCustom<T>] => {
  const [urlValue, setURLValue] = useURLState<string>(param, _options);

  const val = useMemo<T>(() => {
    if (urlValue == null || (Array.isArray(urlValue) && urlValue.length === 0)) {
      return undefined as T;
    }
    return parse(urlValue);
  }, [parse, urlValue]);

  const setVal = useCallback(
    (val: T) => {
      setURLValue(stringify(val));
    },
    [setURLValue, stringify]
  );

  return [val, setVal];
};

export const JSONSerializer = (val: object): string => {
  return JSON.stringify(val, (_, v) => (typeof v === 'bigint' ? v.toString() : v));
};

export const JSONParser = <T,>(val: ParamValue): T => {
  return JSON.parse(val as string);
};

export const NumberParser = (val: string): number => {
  if (val == null || val === '' || val === 'undefined') {
    return 0;
  }
  return Number(val);
};

export const NumberSerializer = (val: number): string => {
  if (val == null) {
    return '';
  }
  return String(val);
};

// Hook to access batch functionality
export const useURLStateBatch = (): ((callback: () => void) => void) => {
  const context = useContext(URLStateContext);
  if (context === undefined) {
    throw new Error('useURLStateBatch must be used within a URLStateProvider');
  }

  return context.batchUpdates;
};

export default URLStateContext;
