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

import moment from 'moment-timezone';

import {ABSOLUTE_TIME_ALIASES, AbsoluteDateValue, DATE_FORMAT} from '../DateTimePicker';

export const UNITS = {
  SECOND: 'second',
  MINUTE: 'minute',
  HOUR: 'hour',
  DAY: 'day',
  WEEK: 'week',
  YEAR: 'year',
};

export const POSITIONS = {
  FROM: 'from',
  TO: 'to',
};

export type UNIT_TYPE = (typeof UNITS)[keyof typeof UNITS];
export type POSITION_TYPE = (typeof POSITIONS)[keyof typeof POSITIONS];

export interface UnitsMap {
  [key: string]: string;
}

export const unitLong: UnitsMap = {
  s: UNITS.SECOND,
  m: UNITS.MINUTE,
  h: UNITS.HOUR,
  d: UNITS.DAY,
  w: UNITS.WEEK,
  y: UNITS.YEAR,
};

export const unitShort: UnitsMap = {
  [UNITS.SECOND]: 's',
  [UNITS.MINUTE]: 'm',
  [UNITS.HOUR]: 'h',
  [UNITS.DAY]: 'd',
  [UNITS.WEEK]: 'w',
  [UNITS.YEAR]: 'y',
};

export const units: Array<[UNIT_TYPE, number]> = [
  [UNITS.YEAR, 31536000],
  [UNITS.WEEK, 604800],
  [UNITS.DAY, 86400],
  [UNITS.HOUR, 3600],
  [UNITS.MINUTE, 60],
  [UNITS.SECOND, 1],
];

export const presetRanges = [
  {value: 1, unit: UNITS.MINUTE},
  {value: 5, unit: UNITS.MINUTE},
  {value: 15, unit: UNITS.MINUTE},
  {value: 30, unit: UNITS.MINUTE},
  {value: 1, unit: UNITS.HOUR},
  {value: 3, unit: UNITS.HOUR},
  {value: 6, unit: UNITS.HOUR},
  {value: 12, unit: UNITS.HOUR},
  {value: 1, unit: UNITS.DAY},
  {value: 2, unit: UNITS.DAY},
  {value: 1, unit: UNITS.WEEK},
  {value: 2, unit: UNITS.WEEK},
  {value: 4, unit: UNITS.WEEK},
  {value: 8, unit: UNITS.WEEK},
  {value: 16, unit: UNITS.WEEK},
  {value: 26, unit: UNITS.WEEK},
  {value: 1, unit: UNITS.YEAR},
  {value: 2, unit: UNITS.YEAR},
];

interface BaseDate {
  isRelative: () => boolean;
  lastEvaluated: number;
  getMs: (forceEvaluate?: boolean) => number;
}
export class RelativeDate implements BaseDate {
  isRelative = (): boolean => true;
  lastEvaluated = 0;
  unit: UNIT_TYPE;
  value: number;

  constructor(unit: UNIT_TYPE, value: number, lastEvaluated = 0) {
    this.unit = unit;
    this.value = value;
    this.lastEvaluated = lastEvaluated;
  }

  getMs(forceEvaluate = false): number {
    if (forceEvaluate || this.lastEvaluated === 0) {
      this.lastEvaluated = getRelativeDateMs(this);
    }
    return this.lastEvaluated;
  }
}

export class AbsoluteDate implements BaseDate {
  isRelative = (): boolean => false;
  lastEvaluated = 0;
  value: AbsoluteDateValue;
  constructor(value?: AbsoluteDateValue, lastEvaluated = 0) {
    this.value = value ?? getDateHoursAgo(1);
    this.lastEvaluated = lastEvaluated;
  }

  getTime(): Date {
    if (typeof this.value === 'string') {
      if (this.value === 'now') {
        return new Date();
      }
    }
    return this.value;
  }

  getUIString(timezone?: string): string {
    if (typeof this.value === 'string') {
      return this.value;
    }

    if (timezone !== undefined) {
      return getStringForDateInTimezone(this, timezone, DATE_FORMAT);
    }

    return getUtcStringForDate(this, DATE_FORMAT);
  }

  getKey(): string {
    if (typeof this.value === 'string') {
      return this.value;
    }
    return this.getTime().getTime().toString();
  }

  getMs(forceEvaluate = false): number {
    if (forceEvaluate || this.lastEvaluated === 0) {
      this.lastEvaluated = this.getTime().getTime();
    }
    return this.lastEvaluated;
  }
}

export type DateUnion = RelativeDate | AbsoluteDate;

export class DateTimeRange {
  from: DateUnion;
  to: DateUnion;

  constructor(from: null | DateUnion = null, to: null | DateUnion = null) {
    this.from = from ?? new RelativeDate(UNITS.MINUTE, 15);
    this.to = to ?? new RelativeDate(UNITS.MINUTE, 0);
  }

  getRangeStringForUI(timezone?: string): string {
    if (this.from.isRelative() && this.to.isRelative() && (this.to as RelativeDate).value === 0) {
      const from = this.from as RelativeDate;
      return `Last ${from.value} ${from.unit}${from.value > 1 ? 's' : ''}`;
    }
    const formattedFrom = formatDateStringForUI(this.from, timezone);
    const fromDatePart = timezone !== undefined
      ? getStringForDateInTimezone(this.from as AbsoluteDate, timezone, 'YYYY-MM-DD')
      : getUtcStringForDate(this.from as AbsoluteDate, 'YYYY-MM-DD');
    const formattedTo = formatDateStringForUI(this.to, timezone).replace(fromDatePart, '').trim();

    return `${formattedFrom} → ${formattedTo}`;
  }

  getDateForPosition(position: POSITION_TYPE): DateUnion {
    if (position === POSITIONS.FROM) {
      return this.from;
    }
    return this.to;
  }

  setDateForPosition(date: DateUnion, position: string): void {
    if (position === POSITIONS.FROM) {
      this.from = date;
    } else {
      this.to = date;
    }
  }

  getFromMs(forceEvaluate = false): number {
    return this.from.getMs(forceEvaluate);
  }

  getToMs(forceEvaluate = false): number {
    return this.to.getMs(forceEvaluate);
  }

  getDateStringKey(date: DateUnion): string {
    if (date.isRelative()) {
      const relativeDate = date as RelativeDate;
      return `${relativeDate.unit}|${relativeDate.value}`;
    }
    const absoluteDate = date as AbsoluteDate;
    return absoluteDate.getKey();
  }

  getFromDateStringKey(): string {
    return this.getDateStringKey(this.from);
  }

  getToDateStringKey(): string {
    return this.getDateStringKey(this.to);
  }

  getRangeKey(): string {
    if (this.from.isRelative()) {
      return `relative:${this.getFromDateStringKey()}`;
    }
    return `absolute:${this.getFromDateStringKey()}-${this.getToDateStringKey()}`;
  }

  static fromRangeKey(
    rangeKey: string | undefined,
    evaluatedFrom?: number | undefined,
    evaluatedTo?: number | undefined
  ): DateTimeRange {
    if (rangeKey === undefined) {
      return new DateTimeRange();
    }
    try {
      const [rangeType, rangeValueKey] = rangeKey.split(':');
      if (rangeType === 'relative') {
        const [unit, value] = rangeValueKey.split('|');
        return new DateTimeRange(
          new RelativeDate(unit, parseInt(value, 10), evaluatedFrom),
          new RelativeDate(UNITS.MINUTE, 0, evaluatedTo)
        );
      }
      if (rangeType === 'absolute') {
        const [fromKey, toKey] = rangeValueKey.split('-');
        const from = parseAbsoluteDateExpression(fromKey);
        if (from != null) {
          from.lastEvaluated = evaluatedFrom ?? 0;
        }
        const to = parseAbsoluteDateExpression(toKey);
        if (to != null) {
          to.lastEvaluated = evaluatedTo ?? 0;
        }
        return new DateTimeRange(from, to);
      }
      throw new Error('Invalid range key');
    } catch (err) {
      console.error('Error while parsing range key', rangeKey, err);
    }
    return new DateTimeRange();
  }

  static fromAbsoluteDates(from: number, to: number): DateTimeRange {
    return new DateTimeRange(new AbsoluteDate(new Date(from)), new AbsoluteDate(new Date(to)));
  }
}

const parseAbsoluteDateExpression = (expression: string): AbsoluteDate | undefined => {
  if (expression === ABSOLUTE_TIME_ALIASES.NOW) {
    return new AbsoluteDate(expression);
  }
  try {
    const date = new Date(parseInt(expression, 10));
    if (isNaN(date.getTime())) {
      return undefined;
    }
    return new AbsoluteDate(date);
  } catch (err) {
    return undefined;
  }
};

export const formatDateStringForUI: (dateString: DateUnion, timezone?: string) => string = (dateString, timezone) => {
  if (dateString.isRelative()) {
    const {unit, value} = dateString as RelativeDate;
    if (value === 0) {
      return 'now';
    }
    return `${value} ${unit}${value > 1 ? 's' : ''} ago`;
  }
  return getStringForDateInTimezone(dateString as AbsoluteDate, timezone ?? 'Etc/GMT');
};

export const getDateHoursAgo = (hours = 1): Date => {
  const now = new Date();
  now.setHours(now.getHours() - hours);
  return now;
};

export const getHistoricalDate = ({value, unit}: {value: number; unit: string}): Date => {
  const now = new Date();
  const msToSubtract = (() => {
    switch (unit) {
      case UNITS.MINUTE:
        return value * 60 * 1000;
      case UNITS.HOUR:
        return value * 60 * 60 * 1000;
      case UNITS.DAY:
        return value * 24 * 60 * 60 * 1000;
      case UNITS.WEEK:
        return value * 7 * 24 * 60 * 60 * 1000;
      default:
        return 0;
    }
  })();

  return new Date(now.getTime() - msToSubtract);
};

const getRelativeDateMs = (date: RelativeDate): number => {
  const now = new Date().getTime();
  const {unit, value} = date;
  switch (unit) {
    case UNITS.MINUTE:
      return now - value * 60 * 1000;
    case UNITS.HOUR:
      return now - value * 60 * 60 * 1000;
    case UNITS.DAY:
      return now - value * 24 * 60 * 60 * 1000;
    case UNITS.WEEK:
      return now - value * 7 * 24 * 60 * 60 * 1000;
    case UNITS.YEAR:
      return now - value * 365 * 24 * 60 * 60 * 1000;
    default:
      return now;
  }
};

export const getUtcStringForDate = (date: AbsoluteDate, format = 'YYYY-MM-DD HH:mm:ss'): string => {
  return moment
    .tz(date.getTime().toISOString(), Intl.DateTimeFormat().resolvedOptions().timeZone)
    .utc()
    .format(format);
};

export const getStringForDateInTimezone = (
  date: AbsoluteDate,
  timezone: string,
  format = 'YYYY-MM-DD HH:mm:ss'
): string => {
  return moment.tz(date.getTime().toISOString(), timezone).format(format);
};

export const parseInput = (input: string): {value: number; unit: UNIT_TYPE} | null => {
  if (input.length > 100) return null;

  const parts = input.match(/(\d+[smhdwy])/g);
  if (parts === null) return null;

  let totalSeconds = 0;
  for (const part of parts) {
    const value = parseInt(part.slice(0, -1));
    const unit = part.slice(-1) as keyof typeof unitLong;
    if (isNaN(value) || unitLong[unit] === '') return null;

    const unitInSeconds = {
      s: 1,
      m: 60,
      h: 3600,
      d: 86400,
      w: 604800,
      y: 31536000,
    }[unit] as number;

    totalSeconds += value * unitInSeconds;
  }

  for (const [unit, seconds] of units) {
    if (totalSeconds >= seconds) {
      return {value: totalSeconds / seconds, unit};
    }
  }

  return {value: totalSeconds, unit: UNITS.SECOND};
};

export const getRelativeTimeRangeBetweenDates = (
  timeRange: number
): {unit: UNIT_TYPE; value: number} => {
  const roundToHundredth = (value: number): number => {
    return Number(value.toFixed(2));
  };

  if (timeRange < 1000 * 60 * 60) {
    const timeRangeToMinutes = timeRange / 1000 / 60;
    return {unit: UNITS.MINUTE, value: roundToHundredth(timeRangeToMinutes)};
  }
  if (timeRange < 1000 * 60 * 60 * 24) {
    const timeRangeToHours = timeRange / 1000 / 60 / 60;
    return {unit: UNITS.HOUR, value: roundToHundredth(timeRangeToHours)};
  }
  if (timeRange < 1000 * 60 * 60 * 24 * 7) {
    const timeRangeToDays = timeRange / 1000 / 60 / 60 / 24;
    return {unit: UNITS.DAY, value: roundToHundredth(timeRangeToDays)};
  }
  const timeRangeToWeeks = timeRange / 1000 / 60 / 60 / 24 / 7;
  return {unit: UNITS.WEEK, value: roundToHundredth(timeRangeToWeeks)};
};

export const formatRange = (value: number, unit: UNIT_TYPE): string => {
  if (value === 0) return `0${unitShort[UNITS.SECOND]}`;

  const parts: string[] = [];
  let remainingValue = value;

  const addPart = (currentUnit: UNIT_TYPE, nextUnit: UNIT_TYPE | null, divisor: number): void => {
    if (remainingValue > 0) {
      const wholePart = Math.floor(remainingValue);
      const fraction = remainingValue - wholePart;

      if (wholePart > 0 || currentUnit === unit) {
        parts.push(`${wholePart}${unitShort[currentUnit]}`);
      }

      if (fraction > 0 && nextUnit !== null) {
        remainingValue = Math.round(fraction * divisor * 100) / 100; // Round to 2 decimal places
      } else {
        remainingValue = 0;
      }
    }
  };

  switch (unit) {
    case UNITS.YEAR:
      addPart(UNITS.YEAR, UNITS.DAY, 365);
      addPart(UNITS.DAY, UNITS.HOUR, 24);
      addPart(UNITS.HOUR, UNITS.MINUTE, 60);
      addPart(UNITS.MINUTE, null, 60);
      break;
    case UNITS.WEEK:
      addPart(UNITS.WEEK, UNITS.DAY, 7);
      addPart(UNITS.DAY, UNITS.HOUR, 24);
      addPart(UNITS.HOUR, UNITS.MINUTE, 60);
      addPart(UNITS.MINUTE, null, 60);
      break;
    case UNITS.DAY:
      addPart(UNITS.DAY, UNITS.HOUR, 24);
      addPart(UNITS.HOUR, UNITS.MINUTE, 60);
      addPart(UNITS.MINUTE, null, 60);
      break;
    case UNITS.HOUR:
      addPart(UNITS.HOUR, UNITS.MINUTE, 60);
      addPart(UNITS.MINUTE, null, 60);
      break;
    case UNITS.MINUTE:
      addPart(UNITS.MINUTE, UNITS.SECOND, 60);
      addPart(UNITS.SECOND, null, 1);
      break;
    case UNITS.SECOND:
      addPart(UNITS.SECOND, null, 1);
      break;
  }

  return parts.join('');
};

export const createNow = (): RelativeDate => new RelativeDate(UNITS.MINUTE, 0);
