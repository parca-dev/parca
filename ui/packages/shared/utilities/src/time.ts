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
import intervalToDuration from 'date-fns/intervalToDuration';

import {Duration} from '@parca/client';

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

export const TimeUnits = {
  Nanos: 'nanos',
  Micros: 'micros',
  Milliseconds: 'milliseconds',
  Seconds: 'seconds',
  Minutes: 'minutes',
  Hours: 'hours',
  Days: 'days',
} as const;

export type TimeUnit = (typeof TimeUnits)[keyof typeof TimeUnits];

export const unitsInTime = {
  [TimeUnits.Nanos]: {multiplier: 1, symbol: 'ns'},
  [TimeUnits.Micros]: {multiplier: 1e3, symbol: 'µs'},
  [TimeUnits.Milliseconds]: {multiplier: 1e6, symbol: 'ms'},
  [TimeUnits.Seconds]: {multiplier: 1e9, symbol: 's'},
  [TimeUnits.Minutes]: {multiplier: 6 * 1e10, symbol: 'm'},
  [TimeUnits.Hours]: {multiplier: 60 * 60 * 1e9, symbol: 'h'},
  [TimeUnits.Days]: {multiplier: 60 * 60 * 24 * 1e9, symbol: 'd'},
};

export const convertTime = (value: number, from: TimeUnit, to: TimeUnit): number => {
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
      return time !== undefined ? convertTime(time, unit as TimeUnit, TimeUnits.Nanos) : 0;
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

export const formatForTimespan = (from: number, to: number): string => {
  const duration = intervalToDuration({start: from, end: to});
  if (duration <= {minutes: 61}) {
    return 'H:mm';
  }
  if (duration <= {hours: 13}) {
    return 'H';
  }
  if (duration <= {hours: 25}) {
    return 'H:mm d/M';
  }
  return 'd/M';
};

export const getStepDuration = (start: number, end: number, stepCount = 1000): Duration => {
  const durationSeconds = (end - start) / stepCount / 1000;
  const whole = Math.floor(durationSeconds);
  const decimal = durationSeconds - whole;

  return {seconds: BigInt(whole), nanos: Math.floor(decimal * 1e9)};
};

export const getStepDurationInMilliseconds = (stepDuration: Duration): number => {
  // Converts both seconds and nanoseconds to milliseconds and adds them together
  return Number(stepDuration.seconds) * 1000 + stepDuration.nanos / 1e6;
};
