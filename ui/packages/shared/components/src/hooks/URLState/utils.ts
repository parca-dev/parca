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

import {parseParams} from '@parca/utilities';

export type ParamValue = string | string[] | undefined;

export interface ParamPreference {
  defaultValue?: ParamValue;
  splitOnCommas?: boolean; // Default: false
}

export type ParamPreferences = Record<string, ParamPreference>;

export const getQueryParamsFromURL = (
  preferences: ParamPreferences = {}
): Record<string, ParamValue> => {
  if (typeof window === 'undefined') {
    return {};
  }

  return parseParams(window.location.search, false, preferences);
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

  // ['flamegraph'] === 'flamegraph'
  if (Array.isArray(a) && a.length === 1 && typeof b === 'string') {
    return decodeURIComponent(a[0]) === decodeURIComponent(b);
  }

  // 'flamegraph' === ['flamegraph']
  if (Array.isArray(b) && b.length === 1 && typeof a === 'string') {
    return decodeURIComponent(b[0]) === decodeURIComponent(a);
  }

  if (a === undefined && b === undefined) {
    return true;
  }

  return false;
};

export const sanitize = (
  params: Record<string, ParamValue>,
  preferences: ParamPreferences
): Record<string, ParamValue> => {
  const sanitized: Record<string, ParamValue> = {};
  for (const [key, value] of Object.entries(params)) {
    const defaultValue = preferences[key]?.defaultValue;
    if (isEmpty(value) || isEqual(value, defaultValue) || value == null) {
      continue;
    }
    if (Array.isArray(value)) {
      sanitized[key] = value.join(',');
    } else {
      sanitized[key] = value;
    }
  }
  return sanitized;
};
