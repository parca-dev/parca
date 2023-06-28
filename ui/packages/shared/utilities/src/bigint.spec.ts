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

import {abs, divide, lowestNumberWithSameNumberOfDigits, scaleLinear} from './bigint';

describe('bigint divide', () => {
  it('divides two bigints and returns a number', () => {
    expect(divide(4n, 2n)).toBe(2);
  });
  it('divides two bigints and returns a number with decimals if available', () => {
    expect(divide(5n, 2n)).toBe(2.5);
  });
});

describe('bigint abs', () => {
  it('returns the absolute value of a negative bigint', () => {
    expect(abs(-4n)).toBe(4n);
  });

  it('returns the absolute value of a positive bigint', () => {
    expect(abs(4n)).toBe(4n);
  });
});

describe('bigint scaleLinear', () => {
  it('scale works fine with a 1:1 function', () => {
    const scale = scaleLinear([0n, 100n], [0, 100]);
    expect(scale(50n)).toBe(50);
    expect(scale(100n)).toBe(100);
    expect(scale(0n)).toBe(0);
    expect(scale(20n)).toBe(20);
  });
  it('scale works fine with a 1:2 function', () => {
    const scale = scaleLinear([0n, 100n], [0, 200]);
    expect(scale(50n)).toBe(100);
    expect(scale(100n)).toBe(200);
    expect(scale(0n)).toBe(0);
    expect(scale(20n)).toBe(40);
  });

  it('scale works fine with a 2:1 function', () => {
    const scale = scaleLinear([0n, 100n], [0, 50]);
    expect(scale(50n)).toBe(25);
    expect(scale(100n)).toBe(50);
    expect(scale(0n)).toBe(0);
    expect(scale(20n)).toBe(10);
  });

  it('scale works with decimal values to range', () => {
    const scale = scaleLinear([0n, 100n], [0, 51.5]);
    expect(scale(50n)).toBe(25.5);
    expect(scale(100n)).toBe(51);
    expect(scale(0n)).toBe(0);
    expect(scale(20n)).toBe(10.2);
  });
});

describe('lowestNumberWithSameNumberOfDigits', () => {
  it('returns the lowest number with the same number of digits', () => {
    expect(lowestNumberWithSameNumberOfDigits(1)).toBe(1);
    expect(lowestNumberWithSameNumberOfDigits(8)).toBe(1);
    expect(lowestNumberWithSameNumberOfDigits(10)).toBe(10);
    expect(lowestNumberWithSameNumberOfDigits(99)).toBe(10);
    expect(lowestNumberWithSameNumberOfDigits(100)).toBe(100);
    expect(lowestNumberWithSameNumberOfDigits(600)).toBe(100);
    expect(lowestNumberWithSameNumberOfDigits(999)).toBe(100);
    expect(lowestNumberWithSameNumberOfDigits(1000)).toBe(1000);
    expect(lowestNumberWithSameNumberOfDigits(6000)).toBe(1000);
    expect(lowestNumberWithSameNumberOfDigits(9999)).toBe(1000);
    expect(lowestNumberWithSameNumberOfDigits(10000)).toBe(10000);
    expect(lowestNumberWithSameNumberOfDigits(9007199254740991)).toBe(1000000000000000);
  });
});

describe('divide', () => {
  it('divides two bigints and returns a number', () => {
    expect(divide(4n, 2n)).toBe(2);
  });
  it('divides two bigints and returns a number with decimals if available', () => {
    expect(divide(5n, 2n)).toBe(2.5);
  });
  it('handles divide by zero', () => {
    expect(divide(5n, 0n)).toBe(Infinity);
  });
});
