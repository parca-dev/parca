import {capitalize, valueFormatter, convertTime, formatDuration, TimeUnits} from './index';

describe('capitalize', () => {
  it('capitalizes each word in a string', () => {
    expect(capitalize('john doe')).toBe('John Doe');
  });
});

describe('valueFormatter', () => {
  it('formats the value passed in count', () => {
    expect(valueFormatter(987654321, 'count', 1)).toBe('987.7M');
  });

  it('formats the value passed in time', () => {
    expect(valueFormatter(9654321, 'nanoseconds', 1)).toBe('9.7ms');
  });

  it('formats the value passed in bytes', () => {
    expect(valueFormatter(9654321, 'bytes', 1)).toBe('9.7MB');
  });

  it('formats a negative value passed correctly', () => {
    expect(valueFormatter(-987654321, 'bytes', 1)).toBe('-987.7MB');
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
