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

import colors from 'tailwindcss/colors';

import {Label} from '@parca/client';

import {abs, divide} from './bigint';
import {unitsInTimeMs, unitsInTimeNs} from './time';

export * from './time';
export * from './string';
export * from './binary-search';
export * from './bigint';

export type NavigateFunction = (
  path: string,
  queryParams: any,
  options?: {replace?: boolean}
) => void;

export const SEARCH_STRING_COLOR = '#e39c9c';

export const capitalize = (a: string): string =>
  a
    .split(' ')
    .map(p => p[0].toUpperCase() + p.substring(1).toLocaleLowerCase())
    .join(' ');

export const capitalizeOnlyFirstLetter = (a: string): string => {
  return a[0].toUpperCase() + a.substring(1).toLocaleLowerCase();
};

interface Unit {
  multiplier: number;
  symbol: string;
}

const unitsInBytes = {
  bytes: {multiplier: 1, symbol: 'Bytes'},
  kilobytes: {multiplier: 1e3, symbol: 'kB'},
  megabytes: {multiplier: 1e6, symbol: 'MB'},
  gigabytes: {multiplier: 1e9, symbol: 'GB'},
  terabytes: {multiplier: 1e12, symbol: 'TB'},
  petabytes: {multiplier: 1e15, symbol: 'PB'},
  exabytes: {multiplier: 1e18, symbol: 'EB'},
};

const unitsInBytesPerSecond = {
  bytes_per_second: {multiplier: 1, symbol: 'Bytes/s'},
  kilobytes_per_second: {multiplier: 1e3, symbol: 'kB/s'},
  megabytes_per_second: {multiplier: 1e6, symbol: 'MB/s'},
  gigabytes_per_second: {multiplier: 1e9, symbol: 'GB/s'},
  terabytes_per_second: {multiplier: 1e12, symbol: 'TB/s'},
  petabytes_per_second: {multiplier: 1e15, symbol: 'PB/s'},
  exabytes_per_second: {multiplier: 1e18, symbol: 'EB/s'},
};

const unitsInCount = {
  unit: {multiplier: 1, symbol: ''},
  kilo: {multiplier: 1e3, symbol: 'k'},
  mega: {multiplier: 1e6, symbol: 'M'},
  giga: {multiplier: 1e9, symbol: 'G'},
  tera: {multiplier: 1e12, symbol: 'T'},
  peta: {multiplier: 1e15, symbol: 'P'},
  exa: {multiplier: 1e18, symbol: 'E'},
};

const unitsInCores = {
  unit: {multiplier: 1, symbol: 'cores'},
  kilo: {multiplier: 1e3, symbol: 'k'},
  mega: {multiplier: 1e6, symbol: 'M'},
  giga: {multiplier: 1e9, symbol: 'G'},
  tera: {multiplier: 1e12, symbol: 'T'},
  peta: {multiplier: 1e15, symbol: 'P'},
  exa: {multiplier: 1e18, symbol: 'E'},
};

const unitsInWatts = {
  unit: {multiplier: 1, symbol: 'W'},
  kilo: {multiplier: 1e3, symbol: 'kW'},
  mega: {multiplier: 1e6, symbol: 'MW'},
  giga: {multiplier: 1e9, symbol: 'GW'},
  tera: {multiplier: 1e12, symbol: 'TW'},
  peta: {multiplier: 1e15, symbol: 'PW'},
  exa: {multiplier: 1e18, symbol: 'EW'},
};

const unitsInCelsius = {
  unit: {multiplier: 1, symbol: '°C'},
};

const unitsInHertz = {
  unit: {multiplier: 1, symbol: 'Hz'},
  kilo: {multiplier: 1e3, symbol: 'kHz'},
  mega: {multiplier: 1e6, symbol: 'MHz'},
  giga: {multiplier: 1e9, symbol: 'GHz'},
  tera: {multiplier: 1e12, symbol: 'THz'},
};

const knownValueFormatters = {
  bytes: unitsInBytes,
  nanoseconds: unitsInTimeNs,
  count: unitsInCount,
  'CPU Cores': unitsInCores,
  milliseconds: unitsInTimeMs,
  watts: unitsInWatts,
  bytes_per_second: unitsInBytesPerSecond,
  celsius: unitsInCelsius,
  hertz: unitsInHertz,
};

export const roundToDecimals = (n: number, decimals: number): number => {
  const log10 = Math.floor(Math.log10(n));
  const div = log10 < 0 ? Math.pow(10, decimals - log10 - 1) : Math.pow(10, decimals);

  return Math.round(n * div) / div;
};

export const getPrecision = (value: number): number =>
  String(roundToDecimals(value, 2)).replace('.', '').length - value.toFixed().length;

export const valueFormatter = (
  num: bigint | number,
  unit: string,
  digits: number,
  renderTight = false
): string => {
  const isBigInt = typeof num === 'bigint';
  const absoluteNum = isBigInt ? abs(num) : Math.abs(num);
  if (unit === 'percent') {
    return `${num.toString()}%`;
  }
  const formatter = knownValueFormatters[unit as keyof typeof knownValueFormatters];
  if (formatter == null) {
    return num.toString();
  }
  const format: Unit[] = Object.values(formatter);
  const rx = /\.0+$|(\.[0-9]*[1-9])0+$/;
  let i: number;
  for (i = format.length - 1; i > 0; i--) {
    if (absoluteNum >= format[i].multiplier) {
      break;
    }
  }

  const multiplier = format[i].multiplier;
  return `${(isBigInt ? divide(num, BigInt(multiplier)) : num / multiplier)
    .toFixed(digits)
    .replace(rx, '$1')}${renderTight ? '' : ' '}${format[i].symbol}`;
};

export const isDevModeOrPreview = (): boolean => {
  return isDevMode() || import.meta.env.REACT_APP_VERCEL_ENV === 'preview';
};

export const isDevMode = (): boolean => {
  return import.meta.env.DEV;
};

export const getLastItem = (thePath: string | undefined | null): string | undefined => {
  if (thePath === undefined || thePath === null || thePath === '') return;

  const index = thePath.lastIndexOf('/');
  if (index === -1) return thePath;

  return thePath.substring(index + 1);
};

const transformToArray = (params: string): string[] => params.split(',');

export const parseParams = (
  querystring: string,
  encodeValues?: boolean
): Record<string, string | string[] | undefined> => {
  const params = new URLSearchParams(querystring);

  const obj: Record<string, string | string[]> = {};
  for (const key of Array.from(params.keys())) {
    let values = params.getAll(key).filter((v): v is string => v != null);

    // Handle expression parameters that might have multiple levels of encoding
    if (
      key === 'expression_a' ||
      key === 'expression_b' ||
      key === 'selection_a' ||
      key === 'selection_b'
    ) {
      values = values.map((value): string => {
        // First, decode multiple levels if present
        const decoded = decodeMultipleEncodings(value);
        // Then, if encodeValues is true, ensure it's encoded once
        if (encodeValues === true) {
          return decoded != null
            ? isUrlEncoded(decoded)
              ? decoded
              : encodeURIComponent(decoded)
            : '';
        }
        // Otherwise return the fully decoded value
        return decoded ?? '';
      });
    }

    if (values.length > 1) {
      obj[key] = values;
    } else {
      if (values[0]?.includes(',')) {
        obj[key] = transformToArray(values[0]);
      } else {
        obj[key] = values[0];
      }
    }
  }

  return obj;
};

export const selectQueryParam = (key: string): string | string[] | undefined => {
  if (typeof window === 'undefined') {
    return;
  }

  const router = parseParams(window.location.search);

  if (key === 'dashboard_items') {
    let dashboardItems = router[key] ?? [];
    if (typeof dashboardItems === 'string') {
      dashboardItems = [dashboardItems];
    }
    return dashboardItems;
  }

  if (key === 'compare_a' || key === 'compare_b') {
    return router[key] === 'true' ? 'true' : 'false';
  }

  return router[key];
};

export const convertToQueryParams = (params: {[key: string]: string | string[]}): string =>
  Object.keys(params)
    .map((key: string) => `${key}=${params[key] as string}`)
    .join('&');

export function convertUTCToLocalDate(date: Date): Date {
  if (date === null) {
    return date;
  }
  return new Date(
    date.getUTCFullYear(),
    date.getUTCMonth(),
    date.getUTCDate(),
    date.getUTCHours(),
    date.getUTCMinutes(),
    date.getUTCSeconds()
  );
}

export function convertLocalToUTCDate(date: Date): Date {
  if (date === null) {
    return date;
  }
  return new Date(
    Date.UTC(
      date.getFullYear(),
      date.getMonth(),
      date.getDate(),
      date.getHours(),
      date.getMinutes(),
      date.getSeconds()
    )
  );
}

export type ColorProfileName = 'default' | 'subtle' | 'ocean' | 'warm' | 'rainbow';
export type ColorsDuo = [string, string];

export interface ColorConfig {
  colors: ColorsDuo[];
  colorForSimilarNodes: string;
}

export const COLOR_PROFILES: {
  [key in ColorProfileName]: ColorConfig;
} = {
  ocean: {
    colors: [
      [colors.green['300'], colors.green['300']],
      [colors.emerald['300'], colors.emerald['300']],
      [colors.teal['300'], colors.teal['300']],
      [colors.cyan['300'], colors.cyan['300']],
      [colors.sky['300'], colors.sky['300']],
      [colors.blue['300'], colors.blue['300']],
      [colors.indigo['300'], colors.indigo['300']],
      [colors.violet['300'], colors.violet['300']],
      [colors.purple['300'], colors.purple['300']],
    ],
    colorForSimilarNodes: 'stroke-indigo-600 dark:stroke-sky-100',
  },
  warm: {
    colors: [
      [colors.red['300'], colors.red['300']],
      [colors.orange['300'], colors.orange['300']],
      [colors.amber['300'], colors.amber['300']],
      [colors.yellow['300'], colors.yellow['300']],
      [colors.lime['300'], colors.lime['300']],
      [colors.green['300'], colors.green['300']],
      [colors.emerald['300'], colors.emerald['300']],
    ],
    colorForSimilarNodes: 'stroke-blue-700 dark:stroke-sky-100',
  },
  subtle: {
    colors: [
      [colors.slate['200'], colors.slate['200']],
      [colors.orange['200'], colors.orange['200']],
      [colors.yellow['200'], colors.yellow['200']],
      [colors.green['100'], colors.green['100']],
      [colors.emerald['200'], colors.emerald['200']],
      [colors.indigo['200'], colors.indigo['200']],
      [colors.pink['200'], colors.pink['200']],
    ],
    colorForSimilarNodes: 'stroke-blue-700 dark:stroke-indigo-600',
  },
  rainbow: {
    colors: [
      [colors.red['300'], colors.red['300']],
      [colors.orange['300'], colors.orange['300']],
      [colors.amber['300'], colors.amber['300']],
      [colors.yellow['300'], colors.yellow['300']],
      [colors.lime['300'], colors.lime['300']],
      [colors.green['300'], colors.green['300']],
      [colors.emerald['300'], colors.emerald['300']],
      [colors.teal['300'], colors.teal['300']],
      [colors.cyan['300'], colors.cyan['300']],
      [colors.sky['300'], colors.sky['300']],
      [colors.blue['300'], colors.blue['300']],
      [colors.indigo['300'], colors.indigo['300']],
      [colors.violet['300'], colors.violet['300']],
      [colors.purple['300'], colors.purple['300']],
      [colors.fuchsia['300'], colors.fuchsia['300']],
      [colors.pink['300'], colors.pink['300']],
      [colors.rose['300'], colors.rose['300']],
    ],
    colorForSimilarNodes: 'stroke-blue-700 dark:stroke-sky-50',
  },
  default: {
    colors: [['#929FEB', '#B3BAE1']],
    colorForSimilarNodes: 'stroke-sky-400 dark:stroke-sky-50',
  },
};

export const getNewSpanColor = (isDarkMode: boolean): string => {
  return isDarkMode ? '#B3BAE1' : '#929FEB';
};

export const getIncreasedSpanColor = (transparency: number, isDarkMode: boolean): string => {
  return isDarkMode
    ? `rgba(255, 177, 204, ${transparency})`
    : `rgba(254, 153, 187, ${transparency})`;
};

export const getReducedSpanColor = (transparency: number, isDarkMode: boolean): string => {
  return isDarkMode
    ? `rgba(103, 158, 92, ${transparency})`
    : `rgba(164, 214, 153, ${transparency})`;
};

const DIFF_RATIO_THRESHOLD = 0.001;

export const diffColor = (diff: bigint, cumulative: bigint, isDarkMode: boolean): string => {
  const prevValue = cumulative - diff;
  const diffRatio = prevValue > 0 ? (diff !== 0n ? divide(diff, prevValue) : 0) : 1.0;
  const hasDiff = Math.abs(diffRatio) > DIFF_RATIO_THRESHOLD;

  return diffColorRatio(hasDiff, diffRatio, isDarkMode);
};

const diffColorRatio = (hasDiff: boolean, diffRatio: number, isDarkMode: boolean): string => {
  const diffTransparency = hasDiff ? Math.min((Math.abs(diffRatio) / 2 + 0.5) * 0.8, 0.8) : 0;

  const newSpanColor = getNewSpanColor(isDarkMode);
  const increasedSpanColor = getIncreasedSpanColor(diffTransparency, isDarkMode);
  const reducedSpanColor = getReducedSpanColor(diffTransparency, isDarkMode);

  const color: string = !hasDiff
    ? newSpanColor
    : diffRatio > 0
    ? increasedSpanColor
    : reducedSpanColor;

  return color;
};

export const isSearchMatch = (currentSearchString: string | undefined, name: string): boolean => {
  if (currentSearchString === undefined || currentSearchString === '') return false;

  return name.toLowerCase().includes(currentSearchString.toLowerCase());
};

export const saveAsBlob = (blob: Blob, filename: string): void => {
  const link = document.createElement('a');
  link.href = window.URL.createObjectURL(blob);
  link.download = filename;
  link.click();
};

export const sanitizeLabelValue = (labels: string[]): string[] => {
  return labels.map((value: string) =>
    value.includes('\\') ? value.replaceAll('\\', '\\\\') : value
  );
};

export const sanitizeHighlightedValues = (labels: Label[]): Label[] =>
  labels.map(v => {
    return {
      ...v,
      value: v.value.includes('\\') ? v.value.replaceAll('\\', '\\\\') : v.value,
    };
  });

export const isUrlEncoded = (str: string): boolean => {
  try {
    return decodeURIComponent(str) !== str;
  } catch (e) {
    return false; // Invalid encoding
  }
};

// Safely decode a string that might have multiple levels of URL encoding
export const decodeMultipleEncodings = (
  str: string | null | undefined
): string | null | undefined => {
  if (str == null) return str;

  let decoded = str;
  let previousDecoded = '';
  const maxIterations = 10; // Prevent infinite loops
  let iterations = 0;

  // Keep decoding until the string doesn't change or we hit the limit
  while (decoded !== previousDecoded && iterations < maxIterations) {
    previousDecoded = decoded;
    try {
      // Check if it's still encoded
      if (isUrlEncoded(decoded)) {
        decoded = decodeURIComponent(decoded);
      } else {
        break;
      }
    } catch (e) {
      // If decoding fails, return the last successful decode
      return previousDecoded;
    }
    iterations++;
  }

  return decoded;
};
