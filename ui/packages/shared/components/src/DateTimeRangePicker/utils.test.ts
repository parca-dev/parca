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

import {UNITS, formatRange, getRelativeTimeRangeBetweenDates, parseInput} from './utils';

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
