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
import {unitsInTime} from './time';

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

const unitsInCount = {
  unit: {multiplier: 1, symbol: ''},
  kilo: {multiplier: 1e3, symbol: 'k'},
  mega: {multiplier: 1e6, symbol: 'M'},
  giga: {multiplier: 1e9, symbol: 'G'},
  tera: {multiplier: 1e12, symbol: 'T'},
  peta: {multiplier: 1e15, symbol: 'P'},
  exa: {multiplier: 1e18, symbol: 'E'},
};

const knownValueFormatters = {
  bytes: unitsInBytes,
  nanoseconds: unitsInTime,
  count: unitsInCount,
};

export const valueFormatter = (num: bigint | number, unit: string, digits: number): string => {
  const isBigInt = typeof num === 'bigint';
  const absoluteNum = isBigInt ? abs(num) : Math.abs(num);
  const format: Unit[] = Object.values(
    knownValueFormatters[unit as keyof typeof knownValueFormatters]
  );

  if (format === undefined || format === null) {
    return num.toString();
  }

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
    .replace(rx, '$1')}${format[i].symbol}`;
};

export const isDevMode = (): boolean => {
  return process.env.NODE_ENV === 'development';
};
export const getLastItem = (thePath: string | undefined): string | undefined => {
  if (thePath === undefined || thePath === '') return;

  const index = thePath.lastIndexOf('/');
  if (index === -1) return thePath;

  return thePath.substring(index + 1);
};

const transformToArray = (params: string): string[] => params.split(',');

export const parseParams = (querystring: string): Record<string, string | string[] | undefined> => {
  const params = new URLSearchParams(querystring);

  const obj: Record<string, string | string[]> = {};
  for (const key of Array.from(params.keys())) {
    const values = params.getAll(key);
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
    let dashboardItems = router[key];
    if (typeof dashboardItems === 'string') {
      dashboardItems = [dashboardItems] ?? [];
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

export const COLOR_PROFILES: {[key in ColorProfileName]: {colors: ColorsDuo[]}} = {
  default: {colors: [['#929FEB', '#B3BAE1']]},
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
  },
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

export const diffColor = (diff: bigint, cumulative: bigint, isDarkMode: boolean): string => {
  const prevValue = cumulative - diff;
  const diffRatio = prevValue > 0 ? (diff !== 0n ? divide(diff, prevValue) : 0) : 1.0;

  const diffTransparency = diff !== 0n ? Math.min((Math.abs(diffRatio) / 2 + 0.5) * 0.8, 0.8) : 0;

  const newSpanColor = getNewSpanColor(isDarkMode);
  const increasedSpanColor = getIncreasedSpanColor(diffTransparency, isDarkMode);
  const reducedSpanColor = getReducedSpanColor(diffTransparency, isDarkMode);

  const color: string =
    diff === 0n ? newSpanColor : diff > 0n ? increasedSpanColor : reducedSpanColor;

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
