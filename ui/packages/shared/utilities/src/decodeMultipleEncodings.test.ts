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

import {expect, test} from 'vitest';

import {decodeMultipleEncodings} from './index';

test('decodeMultipleEncodings - should handle single encoding', () => {
  const input = 'parca_agent%3Asamples%3Acount%3Acpu%3Ananoseconds%3Adelta';
  const expected = 'parca_agent:samples:count:cpu:nanoseconds:delta';
  expect(decodeMultipleEncodings(input)).toBe(expected);
});

test('decodeMultipleEncodings - should handle double encoding', () => {
  const input = 'parca_agent%253Asamples%253Acount%253Acpu%253Ananoseconds%253Adelta';
  const expected = 'parca_agent:samples:count:cpu:nanoseconds:delta';
  expect(decodeMultipleEncodings(input)).toBe(expected);
});

test('decodeMultipleEncodings - should handle triple encoding', () => {
  const input = 'parca_agent%25253Asamples%25253Acount%25253Acpu%25253Ananoseconds%25253Adelta';
  const expected = 'parca_agent:samples:count:cpu:nanoseconds:delta';
  expect(decodeMultipleEncodings(input)).toBe(expected);
});

test('decodeMultipleEncodings - should handle the exact case from the bug report', () => {
  const input =
    'parca_agent%2525252525253Asamples%2525252525253Acount%2525252525253Acpu%2525252525253Ananoseconds%2525252525253Adelta%2525252525257Bnamespace%2525252525253D%25252525252522environment-1c86a3b5-9073-4ba5-9f3f-daf63532ad0a-0%25252525252522%2525252525257D';
  const expected =
    'parca_agent:samples:count:cpu:nanoseconds:delta{namespace="environment-1c86a3b5-9073-4ba5-9f3f-daf63532ad0a-0"}';
  expect(decodeMultipleEncodings(input)).toBe(expected);
});

test('decodeMultipleEncodings - should handle unencoded strings', () => {
  const input = 'parca_agent:samples:count:cpu:nanoseconds:delta';
  expect(decodeMultipleEncodings(input)).toBe(input);
});

test('decodeMultipleEncodings - should handle empty strings', () => {
  expect(decodeMultipleEncodings('')).toBe('');
});

test('decodeMultipleEncodings - should handle null/undefined gracefully', () => {
  expect(decodeMultipleEncodings(null)).toBe(null);
  expect(decodeMultipleEncodings(undefined)).toBe(undefined);
});

test('decodeMultipleEncodings - should handle invalid encoding gracefully', () => {
  const input = 'invalid%ZZencoding';
  expect(decodeMultipleEncodings(input)).toBe(input);
});

test('decodeMultipleEncodings - should prevent infinite loops with malformed input', () => {
  const input = '%25%25%25%25%25%25%25%25%25%25%25%25%25%25%25%25';
  const result = decodeMultipleEncodings(input);
  // Should return something and not hang
  expect(typeof result).toBe('string');
});
