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

import {describe, expect, it} from 'vitest';

import {UNITS, formatRange, getRelativeTimeRangeBetweenDates, parseInput, formatDateStringForUI, AbsoluteDate, RelativeDate} from './utils';

describe('parseInput', () => {
  it('should parse single unit inputs correctly', () => {
    expect(parseInput('30s')).toEqual({value: 30, unit: UNITS.SECOND});
    expect(parseInput('45m')).toEqual({value: 45, unit: UNITS.MINUTE});
    expect(parseInput('2h')).toEqual({value: 2, unit: UNITS.HOUR});
    expect(parseInput('3d')).toEqual({value: 3, unit: UNITS.DAY});
    expect(parseInput('1w')).toEqual({value: 1, unit: UNITS.WEEK});
    expect(parseInput('2y')).toEqual({value: 2, unit: UNITS.YEAR});
  });

  it('should parse multi-unit inputs correctly', () => {
    expect(parseInput('1h30m')).toEqual({value: 1.5, unit: UNITS.HOUR});
    expect(parseInput('2d12h')).toEqual({value: 2.5, unit: UNITS.DAY});
    expect(parseInput('1w2d')).toEqual({value: 1.2857142857142858, unit: UNITS.WEEK});
  });

  it('should return null for invalid inputs', () => {
    expect(parseInput('')).toBeNull();
    expect(parseInput('invalid')).toBeNull();
    expect(parseInput('1x')).toBeNull();
  });

  it('should return null for inputs longer than 100 characters', () => {
    const longInput = '1h'.repeat(51); // 102 characters
    expect(parseInput(longInput)).toBeNull();
  });
});

describe('getRelativeTimeRangeBetweenDates', () => {
  it('should return correct unit and value for various time ranges', () => {
    expect(getRelativeTimeRangeBetweenDates(30 * 1000)).toEqual({unit: UNITS.MINUTE, value: 0.5});
    expect(getRelativeTimeRangeBetweenDates(45 * 60 * 1000)).toEqual({
      unit: UNITS.MINUTE,
      value: 45,
    });
    expect(getRelativeTimeRangeBetweenDates(2 * 60 * 60 * 1000)).toEqual({
      unit: UNITS.HOUR,
      value: 2,
    });
    expect(getRelativeTimeRangeBetweenDates(3 * 24 * 60 * 60 * 1000)).toEqual({
      unit: UNITS.DAY,
      value: 3,
    });
    expect(getRelativeTimeRangeBetweenDates(10 * 24 * 60 * 60 * 1000)).toEqual({
      unit: UNITS.WEEK,
      value: 1.43,
    });
  });

  it('should round values to two decimal places', () => {
    expect(getRelativeTimeRangeBetweenDates(2 * 60 * 60 * 1000 + 5 * 60 * 1000)).toEqual({
      unit: UNITS.HOUR,
      value: 2.08,
    });
  });
});

describe('formatRange', () => {
  it('should format single unit ranges correctly', () => {
    expect(formatRange(30, UNITS.SECOND)).toBe('30s');
    expect(formatRange(45, UNITS.MINUTE)).toBe('45m');
    expect(formatRange(2, UNITS.HOUR)).toBe('2h');
    expect(formatRange(3, UNITS.DAY)).toBe('3d');
    expect(formatRange(1, UNITS.WEEK)).toBe('1w');
    expect(formatRange(2, UNITS.YEAR)).toBe('2y');
  });

  it('should preserve original units when possible', () => {
    expect(formatRange(25, UNITS.HOUR)).toBe('25h');
    expect(formatRange(36, UNITS.HOUR)).toBe('36h');
    expect(formatRange(1.5, UNITS.DAY)).toBe('1d12h');
    expect(formatRange(1.25, UNITS.HOUR)).toBe('1h15m');
  });

  it('should handle fractional values correctly', () => {
    expect(formatRange(1.75, UNITS.HOUR)).toBe('1h45m');
    expect(formatRange(2.5, UNITS.DAY)).toBe('2d12h');
  });

  it('should return 0s for zero values', () => {
    expect(formatRange(0, UNITS.SECOND)).toBe('0s');
    expect(formatRange(0, UNITS.HOUR)).toBe('0s');
  });

  it('should handle edge cases correctly', () => {
    expect(formatRange(0.1, UNITS.MINUTE)).toBe('0m6s');
    expect(formatRange(1.99, UNITS.DAY)).toBe('1d23h45m');
    expect(formatRange(1.001, UNITS.HOUR)).toBe('1h');
  });
});

describe('formatDateStringForUI', () => {
  it('should format relative dates correctly', () => {
    const relativeDate = new RelativeDate(UNITS.MINUTE, 15);
    expect(formatDateStringForUI(relativeDate)).toBe('15 minutes ago');

    const singleMinute = new RelativeDate(UNITS.MINUTE, 1);
    expect(formatDateStringForUI(singleMinute)).toBe('1 minute ago');

    const nowDate = new RelativeDate(UNITS.MINUTE, 0);
    expect(formatDateStringForUI(nowDate)).toBe('now');
  });

  it('should format absolute dates in UTC when no timezone is provided', () => {
    const testDate = new Date('2023-12-01T15:30:00Z');
    const absoluteDate = new AbsoluteDate(testDate);
    
    const result = formatDateStringForUI(absoluteDate);
    expect(result).toBe('2023-12-01 15:30:00');
  });

  it('should format absolute dates in specified timezone when timezone is provided', () => {
    const testDate = new Date('2023-12-01T15:30:00Z');
    const absoluteDate = new AbsoluteDate(testDate);
    
    const result = formatDateStringForUI(absoluteDate, 'America/New_York');
    expect(result).toBe('2023-12-01 10:30:00');
  });

  it('should handle different timezones correctly', () => {
    const testDate = new Date('2023-06-15T12:00:00Z');
    const absoluteDate = new AbsoluteDate(testDate);
    
    const utcResult = formatDateStringForUI(absoluteDate);
    expect(utcResult).toBe('2023-06-15 12:00:00');
    
    const pacificResult = formatDateStringForUI(absoluteDate, 'America/Los_Angeles');
    expect(pacificResult).toBe('2023-06-15 05:00:00');
    
    const tokyoResult = formatDateStringForUI(absoluteDate, 'Asia/Tokyo');
    expect(tokyoResult).toBe('2023-06-15 21:00:00');
  });

  it('should handle timezone date boundary crossing', () => {
    const testDate = new Date('2023-12-01T02:00:00Z');
    const absoluteDate = new AbsoluteDate(testDate);
    
    const result = formatDateStringForUI(absoluteDate, 'America/Los_Angeles');
    expect(result).toBe('2023-11-30 18:00:00');
  });
});
