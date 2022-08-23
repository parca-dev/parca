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

import {SuffixParams, ParseLabels} from '../ProfileSource';

test('prefixes keys', () => {
  const input = {key: 'value'};
  expect(SuffixParams(input, '_a')).toMatchObject({key_a: 'value'});
});

test('parses labels', () => {
  const input = ['key=value'];
  expect(ParseLabels(input)).toMatchObject([{name: 'key', value: 'value'}]);
});
