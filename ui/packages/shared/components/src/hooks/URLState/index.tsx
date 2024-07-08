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

import {ReactNode, createContext, useContext, useState} from 'react';

import {type NavigateFunction} from '@parca/utilities';

type ParamValue = string | string[] | undefined;
type ParamValueSetter = (val: ParamValue) => void;

interface URLState {
  navigateTo: NavigateFunction;
  state: Record<string, string | string[] | undefined>;
  setState: (state: Record<string, ParamValue>) => void;
}

const URLStateContext = createContext<URLState | undefined>(undefined);

export const URLStateProvider = ({
  children,
  navigateTo,
}: {
  children: ReactNode;
  navigateTo: NavigateFunction;
}): JSX.Element => {
  const [state, setState] = useState<Record<string, ParamValue>>(getQueryParamsFromURL());
  return (
    <URLStateContext.Provider value={{navigateTo, state, setState}}>
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

const sanitize = (params: Record<string, ParamValue>): Record<string, ParamValue> => {
  console.log('Before sanitize', params);
  const sanitized: Record<string, ParamValue> = {};
  for (const [key, value] of Object.entries(params)) {
    if (isEmpty(value)) {
      console.log('isEmpty', key, value);
      continue;
    }
    sanitized[key] = value;
  }
  return sanitized;
};

interface Options {
  defaultValue?: string | string[];
  debugLog?: boolean;
}

export const useURLStateNew = (
  param: string,
  _options?: Options
): [ParamValue, ParamValueSetter] => {
  const context = useContext(URLStateContext);
  if (context === undefined) {
    throw new Error('useURLState must be used within a URLStateProvider');
  }

  const {debugLog} = _options ?? {};

  const {navigateTo, state, setState} = context;

  const setParam: ParamValueSetter = (val: ParamValue) => {
    setState({...state, [param]: val});
    navigateTo(window.location.pathname, sanitize({...getQueryParamsFromURL(), [param]: val}), {
      replace: true,
    });
    if (debugLog === true) {
      console.log('useURLStateNew setParam', param, val);
    }
  };

  if (debugLog === true) {
    console.log('useURLStateNew state', param, state[param]);
  }

  let value: ParamValue;
  if (typeof state[param] === 'string') {
    value = state[param];
  } else if (state[param] != null && Array.isArray(state[param])) {
    if (state[param]?.length === 1) {
      value = state[param]?.[0];
    } else {
      value = state[param];
    }
  }

  return [value, setParam];
};

export default URLStateContext;
