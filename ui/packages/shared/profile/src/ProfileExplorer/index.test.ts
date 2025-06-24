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

import { describe, it, expect } from 'vitest';
import { filterEmptyParams } from './index';

describe('filterEmptyParams', () => {
  it('should return an array with 2 elements when given object with 2 valid and multiple invalid values', () => {
    const input = {
      validString: 'hello',
      validArray: ['item1', 'item2'],
      emptyString: '',
      undefinedValue: undefined,
      emptyArray: [],
      anotherEmptyString: '',
    };

    const result = filterEmptyParams(input);
    const resultEntries = Object.entries(result);

    expect(resultEntries).toHaveLength(2);
    expect(result).toEqual({
      validString: 'hello',
      validArray: ['item1', 'item2'],
    });
  });

  it('should filter out empty strings', () => {
    const input = {
      valid: 'test',
      empty: '',
    };

    const result = filterEmptyParams(input);
    expect(result).toEqual({ valid: 'test' });
  });

  it('should filter out undefined values', () => {
    const input = {
      valid: 'test',
      notDefined: undefined,
    };

    const result = filterEmptyParams(input);
    expect(result).toEqual({ valid: 'test' });
  });

  it('should filter out empty arrays', () => {
    const input = {
      valid: 'test',
      emptyArray: [],
      nonEmptyArray: ['item'],
    };

    const result = filterEmptyParams(input);
    expect(result).toEqual({
      valid: 'test',
      nonEmptyArray: ['item'],
    });
  });

  it('should keep all valid values including numbers, booleans, and objects', () => {
    const input = {
      string: 'test',
      number: 0,
      boolean: false,
      object: { key: 'value' },
      array: ['item'],
    };

    const result = filterEmptyParams(input);
    expect(result).toEqual(input);
  });

  it('should return empty object when all values are invalid', () => {
    const input = {
      empty1: '',
      empty2: '',
      undefined1: undefined,
      emptyArray: [],
    };

    const result = filterEmptyParams(input);
    expect(result).toEqual({});
  });
});
