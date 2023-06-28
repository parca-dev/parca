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

/**
 * Divides two bigints and returns a number with two decimal places
 */

export const lowestNumberWithSameNumberOfDigits = (a: number): number => {
  const digits = Math.floor(Math.log10(a)) + 1;
  return 10 ** (digits - 1);
};

const MULTIPLE = lowestNumberWithSameNumberOfDigits(Number.MAX_SAFE_INTEGER);
const MULTIPLE_BIGINT = BigInt(MULTIPLE);

export const divide = (a: bigint, b: bigint): number => {
  if (b === 0n) {
    return Infinity;
  }
  return Number((a * MULTIPLE_BIGINT) / b) / MULTIPLE;
};

/**
 * Returns the absolute value of a bigint
 */
export const abs = (a: bigint): bigint => {
  return a < 0n ? -a : a;
};

export type ScaleFunction = (x: bigint) => number;

export const scaleLinear = (domain: [bigint, bigint], range: [number, number]): ScaleFunction => {
  const [domainMin, domainMax] = domain;
  const [rangeMin, rangeMax] = range;
  const domainRange = domainMax - domainMin;
  const rangeRange = BigInt(Math.floor(rangeMax - rangeMin));

  // rate * MULTIPLE to retain the decimal places in BigInt format, then divide by MULTIPLE to get the final result
  const rate = BigInt(Math.round(divide(rangeRange, domainRange) * MULTIPLE));

  return x => {
    return Number(BigInt(rangeMin) + (x - domainMin) * rate) / MULTIPLE;
  };
};
