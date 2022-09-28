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

import format from 'date-fns/format';

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
export interface TimeObject {
  nanos?: number;
  micros?: number;
  milliseconds?: number;
  seconds?: number;
  minutes?: number;
  hours?: number;
  days?: number;
  weeks?: number;
  years?: number;
}

export enum TimeUnits {
  Nanos = 'nanos',
  Micros = 'micros',
  Milliseconds = 'milliseconds',
  Seconds = 'seconds',
  Minutes = 'minutes',
  Hours = 'hours',
  Days = 'days',
  Weeks = 'weeks',
  Years = 'years',
}

const unitsInTime = {
  [TimeUnits.Nanos]: {multiplier: 1, symbol: 'ns'},
  [TimeUnits.Micros]: {multiplier: 1e3, symbol: 'µs'},
  [TimeUnits.Milliseconds]: {multiplier: 1e6, symbol: 'ms'},
  [TimeUnits.Seconds]: {multiplier: 1e9, symbol: 's'},
  [TimeUnits.Minutes]: {multiplier: 6 * 1e10, symbol: 'm'},
  [TimeUnits.Hours]: {multiplier: 60 * 60 * 1e9, symbol: 'h'},
  [TimeUnits.Days]: {multiplier: 60 * 60 * 24 * 1e9, symbol: 'd'},
  [TimeUnits.Weeks]: {multiplier: 60 * 60 * 24 * 7 * 1e9, symbol: 'w'},
  [TimeUnits.Years]: {multiplier: 60 * 60 * 24 * 365 * 1e9, symbol: 'y'},
};

export const convertTime = (value: number, from: TimeUnits, to: TimeUnits): number => {
  const startUnit = unitsInTime[from];
  const endUnit = unitsInTime[to];
  if (startUnit === undefined || endUnit === undefined) {
    console.error('invalid start or end unit provided');
    return value;
  }

  return (value * startUnit.multiplier) / endUnit.multiplier;
};

export const formatDuration = (timeObject: TimeObject, to?: number): string => {
  let values: string[] = [];
  const unitsLargeToSmall = Object.values(TimeUnits).reverse();

  let nanos = (Object.keys(timeObject) as Array<keyof TimeObject>)
    .map(unit => {
      const time = timeObject[unit];
      return time !== undefined ? convertTime(time, unit as TimeUnits, TimeUnits.Nanos) : 0;
    })
    .reduce((prev, curr) => prev + curr, 0);

  if (to !== undefined) {
    nanos = to - nanos;
  }

  // for more than one second, just show up until whole seconds; otherwise, show whole micros
  if (Math.floor(nanos / unitsInTime[TimeUnits.Seconds].multiplier) > 0) {
    for (let i = 0; i < unitsLargeToSmall.length; i++) {
      const multiplier = unitsInTime[unitsLargeToSmall[i]].multiplier;

      if (nanos > multiplier) {
        if (unitsLargeToSmall[i] === TimeUnits.Milliseconds) {
          break;
        } else {
          const amount = Math.floor(nanos / multiplier);
          values = [...values, `${amount}${unitsInTime[unitsLargeToSmall[i]].symbol}`];
          nanos -= amount * multiplier;
        }
      }
    }
  } else {
    const milliseconds = Math.floor(nanos / unitsInTime[TimeUnits.Milliseconds].multiplier);
    if (milliseconds > 0) {
      values = [`${milliseconds}${unitsInTime[TimeUnits.Milliseconds].symbol}`];
    } else {
      return '<1ms';
    }
  }

  return values.join(' ');
};

export const formatDate = (date: number | Date, timeFormat: string): string => {
  if (typeof date === 'number') {
    date = new Date(date);
  }

  const ISOString = date.toISOString().slice(0, -1);
  return format(new Date(ISOString), timeFormat);
};

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

export const valueFormatter = (num: number, unit: string, digits: number): string => {
  // TODO: remove this after the columnstore backend is the main storage
  // backend. This is a temporary fix while the columnstore backend does not
  // return the correct unit.
  if (unit === undefined || unit === '') {
    return num.toFixed(digits);
  }

  const absoluteNum = Math.abs(num);
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
  return `${(num / format[i].multiplier).toFixed(digits).replace(rx, '$1')}${format[i].symbol}`;
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

export const parseParams = (querystring: string): Record<string, string | string[]> => {
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

export const convertToQueryParams = (params: Record<string, string>): string =>
  Object.keys(params)
    .map(key => key + '=' + params[key])
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

export const getNewSpanColor = (isDarkMode: boolean): string =>
  isDarkMode ? '#B3BAE1' : '#929FEB';
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

export const diffColor = (diff: number, cumulative: number, isDarkMode: boolean): string => {
  const prevValue = cumulative - diff;
  const diffRatio = prevValue > 0 ? (Math.abs(diff) > 0 ? diff / prevValue : 0) : 1.0;

  const diffTransparency =
    Math.abs(diff) > 0 ? Math.min((Math.abs(diffRatio) / 2 + 0.5) * 0.8, 0.8) : 0;

  const newSpanColor = getNewSpanColor(isDarkMode);
  const increasedSpanColor = getIncreasedSpanColor(diffTransparency, isDarkMode);
  const reducedSpanColor = getReducedSpanColor(diffTransparency, isDarkMode);

  const color = diff === 0 ? newSpanColor : diff > 0 ? increasedSpanColor : reducedSpanColor;

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
