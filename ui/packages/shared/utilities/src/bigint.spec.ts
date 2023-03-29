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

import {abs, divide} from './bigint';

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
