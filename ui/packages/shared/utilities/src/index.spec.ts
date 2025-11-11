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

import {
  TimeUnits,
  capitalize,
  convertTime,
  formatDate,
  formatDuration,
  parseParams,
  valueFormatter,
} from './index';

describe('capitalize', () => {
  it('capitalizes each word in a string', () => {
    expect(capitalize('john doe')).toBe('John Doe');
  });
});

describe('valueFormatter', () => {
  it('formats the value passed in count', () => {
    expect(valueFormatter(987654321n, 'count', 1)).toBe('987.7 M');
  });

  it('formats the value passed in time', () => {
    expect(valueFormatter(9654321n, 'nanoseconds', 1)).toBe('9.7 ms');
  });

  it('formats the value passed in time with 2 digits', () => {
    expect(valueFormatter(9654321n, 'nanoseconds', 2)).toBe('9.65 ms');
  });

  it('formats the value passed in time with 2 digits with tight true', () => {
    expect(valueFormatter(9654321n, 'nanoseconds', 2, true)).toBe('9.65ms');
  });

  it('formats the value passed in bytes', () => {
    expect(valueFormatter(9654321n, 'bytes', 1)).toBe('9.7 MB');
  });

  it('formats a negative value passed correctly', () => {
    expect(valueFormatter(-987654321n, 'bytes', 1)).toBe('-987.7 MB');
  });
});

describe('convertTime', () => {
  it('converts larger units to smaller units', () => {
    expect(convertTime(9, TimeUnits.Seconds, TimeUnits.Nanos)).toBe(9 * 1e9);
    expect(convertTime(9, TimeUnits.Micros, TimeUnits.Nanos)).toBe(9 * 1e3);
    expect(convertTime(9, TimeUnits.Hours, TimeUnits.Nanos)).toBe(9 * 60 * 60 * 1e9);
    expect(convertTime(9, TimeUnits.Milliseconds, TimeUnits.Nanos)).toBe(9 * 1e6);
  });

  it('converts smaller units to larger units', () => {
    expect(convertTime(9000000000, TimeUnits.Nanos, TimeUnits.Seconds)).toBe(9);
    expect(convertTime(9000000000, TimeUnits.Nanos, TimeUnits.Milliseconds)).toBe(9000);
    expect(convertTime(9000, TimeUnits.Milliseconds, TimeUnits.Seconds)).toBe(9);
  });
});

describe('formatDuration', () => {
  it('formats a duration over one second correctly', () => {
    expect(formatDuration({nanos: 9654321000})).toBe('9s');
  });

  it('formats a relative duration over one second correctly', () => {
    const seconds = 1644414491;
    const nanos = 630134000;
    const currentNanos = 1644415325511000000;
    const durationInNanos = seconds * 1e9 + nanos;
    expect(formatDuration({seconds, nanos}, currentNanos)).toBe(
      formatDuration({nanos: currentNanos - durationInNanos})
    );
    expect(formatDuration({seconds, nanos}, currentNanos)).toBe('13m 53s');
  });

  it('formats a duration below one second correctly', () => {
    expect(formatDuration({nanos: 9654321})).toBe('9ms');
  });

  it('formats a relative duration below one second correctly', () => {
    const seconds = 1644415325;
    const nanos = 511000000;
    const currentNanos = 1644415325530134000;
    const durationInNanos = seconds * 1e9 + nanos;
    expect(formatDuration({seconds, nanos}, currentNanos)).toBe(
      formatDuration({nanos: currentNanos - durationInNanos})
    );
    expect(formatDuration({seconds, nanos}, currentNanos)).toBe('19ms');
  });
});

describe('formatDate', () => {
  it('formats date to given time format', () => {
    expect(
      formatDate(new Date('2019-01-01T00:00:00Z'), "'Date:' dd/MM/yyyy 'Time:' hh:mm:s'")
    ).toBe('Date: 01/01/2019 Time: 12:00:0');
  });
});

describe('parseParams', () => {
  it('transforms legacy icicle parameter to flamegraph for backward compatibility', () => {
    const result = parseParams('?dashboard_items=icicle');
    expect(result.dashboard_items).toBe('flamegraph');
  });

  it('transforms legacy iciclechart parameter to flamechart for backward compatibility', () => {
    const result = parseParams('?dashboard_items=iciclechart');
    expect(result.dashboard_items).toBe('flamechart');
  });

  it('preserves new flamegraph parameter values', () => {
    const result = parseParams('?dashboard_items=flamegraph');
    expect(result.dashboard_items).toBe('flamegraph');
  });

  it('preserves new flamechart parameter values', () => {
    const result = parseParams('?dashboard_items=flamechart');
    expect(result.dashboard_items).toBe('flamechart');
  });

  it('handles multiple dashboard_items with mixed legacy and new values', () => {
    const result = parseParams('?dashboard_items=icicle,table,iciclechart,flamegraph', undefined, {dashboard_items: {splitOnCommas: true}});
    expect(result.dashboard_items).toEqual(['flamegraph', 'table', 'flamechart', 'flamegraph']);
  });

  it('does not affect other parameters', () => {
    const result = parseParams('?dashboard_items=icicle&other_param=value');
    expect(result.dashboard_items).toBe('flamegraph');
    expect(result.other_param).toBe('value');
  });
});
