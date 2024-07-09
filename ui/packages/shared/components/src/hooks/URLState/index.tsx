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

import {ReactNode, createContext, useContext, useMemo, useState} from 'react';

import {type NavigateFunction} from '@parca/utilities';

type ParamValue = string | string[] | undefined;
type ParamValueSetter = (val: ParamValue) => void;

interface URLState {
  navigateTo: NavigateFunction;
  state: Record<string, string | string[] | undefined>;
  setState: (state: Record<string, ParamValue>) => void;
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

const getQueryParamsFromURL = (): Record<string, ParamValue> => {
  if (typeof window === 'undefined') {
    return {};
  }

  const searchParams = new URLSearchParams(window.location.search);
  const params: Record<string, ParamValue> = {};

  searchParams.forEach((value, key) => {
    params[key] = value.split(',');
  });

  return params;
};

const isEmpty = (val: string | string[] | undefined): boolean => {
  return val === undefined || val == null || val === '' || (Array.isArray(val) && val.length === 0);
};

const isEqual = (a: ParamValue, b: ParamValue): boolean => {
  if (typeof a === 'string' && typeof b === 'string') {
    return decodeURIComponent(a) === decodeURIComponent(b);
  }

  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) {
      return false;
    }

    for (let i = 0; i < a.length; i++) {
      if (a[i] !== b[i]) {
        return false;
      }
    }
    return true;
  }

  if (Array.isArray(a) && a.length === 1 && typeof b === 'string') {
    return decodeURIComponent(a[0]) === decodeURIComponent(b);
  }

  if (Array.isArray(b) && b.length === 1 && typeof a === 'string') {
    return decodeURIComponent(b[0]) === decodeURIComponent(a);
  }

  return false;
};

const sanitize = (
  params: Record<string, ParamValue>,
  defaultValues: Record<string, ParamValue>
): Record<string, ParamValue> => {
  const sanitized: Record<string, ParamValue> = {};
  for (const [key, value] of Object.entries(params)) {
    if (isEmpty(value) || isEqual(value, defaultValues[key])) {
      continue;
    }
    sanitized[key] = value;
  }
  return sanitized;
};

interface Options {
  defaultValue?: string | string[];
  debugLog?: boolean;
  alwaysReturnArray?: boolean;
}

export const useURLStateNew = <T extends ParamValue>(
  param: string,
  _options?: Options
): [T, ParamValueSetter] => {
  const context = useContext(URLStateContext);
  if (context === undefined) {
    throw new Error('useURLState must be used within a URLStateProvider');
  }

  const {debugLog, defaultValue, alwaysReturnArray} = _options ?? {};

  const {navigateTo, state, setState, defaultValues} = context;

  const setParam: ParamValueSetter = (val: ParamValue) => {
    if (debugLog === true) {
      console.log('useURLStateNew setParam', param, val);
    }
    setState({...state, [param]: val});
    navigateTo(
      window.location.pathname,
      sanitize({...getQueryParamsFromURL(), [param]: val}, defaultValues),
      {
        replace: true,
      }
    );
  };

  if (debugLog === true) {
    console.log('useURLStateNew state', param, state[param]);
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

export default URLStateContext;
