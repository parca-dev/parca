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
  useState,
} from 'react';

import {type NavigateFunction} from '@parca/utilities';

import {compressParam, decompressParam} from './compression';
import {getQueryParamsFromURL, sanitize, type ParamValue} from './utils';

export type ParamValueSetter = (val: ParamValue) => void;

interface URLState {
  navigateTo: NavigateFunction;
  state: Record<string, string | string[] | undefined>;
  setState: Dispatch<SetStateAction<Record<string, ParamValue>>>;
  defaultValues: Record<string, ParamValue>;
}

const URLStateContext = createContext<URLState | undefined>(undefined);

export const URLStateProvider = ({
  children,
  navigateTo,
  defaultValues = {},
}: {
  children: ReactNode;
  navigateTo: NavigateFunction;
  defaultValues?: Record<string, ParamValue>;
}): JSX.Element => {
  const [state, setState] = useState<Record<string, ParamValue>>({
    ...defaultValues,
    ...getQueryParamsFromURL(),
  });

  return (
    <URLStateContext.Provider value={{navigateTo, state, setState, defaultValues}}>
      {children}
    </URLStateContext.Provider>
  );
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

  const {navigateTo, state, setState, defaultValues} = context;

  const setParam: ParamValueSetter = useCallback(
    (val: ParamValue) => {
      setTimeout(() => {
        if (debugLog === true) {
          console.log('useURLState setParam', param, val);
        }
        setState(state => ({...state, [param]: val}));

        navigateTo(
          window.location.pathname,
          sanitize({...getQueryParamsFromURL(), [param]: val}, defaultValues),
          {
            replace: true,
          }
        );
      });
    },
    [param, navigateTo, setState, defaultValues, debugLog]
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

  // TODO(manoj) Fix the forced type
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

export interface OptionsCompressed<T> {
  parse?: (val: string) => T;
  stringify?: (val: T) => string;
}

/**
 * Hook for URL state with automatic LZ4 compression/decompression
 *
 * This hook automatically compresses values when writing to the URL and
 * decompresses when reading. Useful for large parameter values like JSON objects.
 *
 * If parse/stringify options are provided, the flow is:
 * - Writing: value -> stringify (if provided) -> compress -> URL
 * - Reading: URL -> decompress -> parse (if provided) -> value
 *
 * @example
 * // Simple string compression
 * const [config, setConfig] = useURLStateCompressed('config');
 * setConfig(JSON.stringify({filters: [...], groupBy: [...]}));
 *
 * @example
 * // With custom parse/stringify
 * const [filters, setFilters] = useURLStateCompressed<Filter[]>('filters', {
 *   parse: (str) => JSON.parse(str),
 *   stringify: (val) => JSON.stringify(val)
 * });
 */
export const useURLStateCompressed = <T,>(
  param: string,
  options?: Options & OptionsCompressed<T>
): [T | undefined, (val: T) => void] => {
  const {parse, stringify, ...urlStateOptions} = options ?? {};

  return useURLStateCustom<T>(param, {
    parse: (val: ParamValue): T => {
      if (val == null || val === '') {
        return (parse != null ? parse('') : '') as T;
      }

      // Decompress first
      const stringVal = Array.isArray(val) ? val[0] : val;
      const decompressed = decompressParam(stringVal);

      // Then parse if parser provided
      if (parse != null) {
        return parse(decompressed);
      }

      return decompressed as T;
    },
    stringify: (val: T): ParamValue => {
      if (val == null || val === '') return '';

      // Stringify first if stringifier provided
      const stringified = stringify != null ? stringify(val) : String(val);

      // Then compress
      return compressParam(stringified);
    },
    ...urlStateOptions,
  });
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

export default URLStateContext;
