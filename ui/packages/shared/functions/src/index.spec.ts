import {capitalize, valueFormatter} from './index';

describe('Functions', () => {
  it('capitalize', () => {
    expect(capitalize('john doe')).toBe('John Doe');
  });

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
