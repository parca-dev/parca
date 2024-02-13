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

  getUIString(): string {
    if (typeof this.value === 'string') {
      return this.value;
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

  getRangeStringForUI(): string {
    if (this.from.isRelative() && this.to.isRelative() && (this.to as RelativeDate).value === 0) {
      const from = this.from as RelativeDate;
      return `Last ${from.value} ${from.unit}${from.value > 1 ? 's' : ''}`;
    }
    const formattedFrom = formatDateStringForUI(this.from);
    const formattedTo = formatDateStringForUI(this.to)
      .replace(getUtcStringForDate(this.from as AbsoluteDate, 'll'), '')
      .trim();
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

export const formatDateStringForUI: (dateString: DateUnion) => string = dateString => {
  if (dateString.isRelative()) {
    const {unit, value} = dateString as RelativeDate;
    if (value === 0) {
      return 'now';
    }
    return `${value} ${unit}${value > 1 ? 's' : ''} ago`;
  }
  return getUtcStringForDate(dateString as AbsoluteDate);
};

export const getDateHoursAgo = (hours = 1): Date => {
  const now = new Date();
  now.setHours(now.getHours() - hours);
  return now;
};

export const getHistoricalDate = ({value, unit}: {value: number; unit: string}): Date => {
  const now = new Date();
  switch (unit) {
    case UNITS.MINUTE:
      now.setMinutes(now.getMinutes() - value);
      return now;
    case UNITS.HOUR:
      now.setHours(now.getHours() - value);
      return now;
    case UNITS.DAY:
      now.setDate(now.getDate() - value);
      return now;
    default:
      return now;
  }
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

export const getUtcStringForDate = (date: AbsoluteDate, format = 'lll'): string => {
  return moment
    .tz(date.getTime().toISOString(), Intl.DateTimeFormat().resolvedOptions().timeZone)
    .utc()
    .format(format);
};
