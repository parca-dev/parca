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

import {Matcher, MatcherTypes, ProfileType, Query} from './index';

test('QueryParseEmpty', () => {
  expect(Query.parse('')).toMatchObject(
    new Query(new ProfileType('', '', '', '', '', false), [], '')
  );
});

test('QueryParseProfile', () => {
  expect(Query.parse('memory:alloc_objects:count:space:bytes:delta')).toMatchObject(
    new Query(new ProfileType('memory', 'alloc_objects', 'count', 'space', 'bytes', true), [], '')
  );
});

test('QueryParseWithMatcher', () => {
  expect(Query.parse('memory:inuse_objects:count:space:bytes{instance="abc"}')).toMatchObject(
    new Query(
      new ProfileType('memory', 'inuse_objects', 'count', 'space', 'bytes', false),
      [new Matcher('instance', MatcherTypes.MatchEqual, 'abc')],
      ''
    )
  );
});

test('Query.toString', () => {
  expect(Query.parse('memory:inuse_objects:count:space:bytes{instance="abc"}').toString()).toBe(
    'memory:inuse_objects:count:space:bytes{instance="abc"}'
  );
});

test('Partial Parsing ProfileName and rest', () => {
  [
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{instance="abc",a',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'instance="abc",a',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{instance="abc",',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'instance="abc",',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{instance="ab',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'instance="ab',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{instance="',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'instance="',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{instance=a',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'instance=a',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{=a',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: '=a',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{a',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: 'a',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta{',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: '',
    },
    {
      input: 'memory:alloc_objects:count:space:bytes:delta',
      expectedProfileName: 'memory:alloc_objects:count:space:bytes:delta',
      expectedMatcherString: '',
    },
  ].forEach(function (test) {
    const q = Query.parse(test.input);

    expect(q.profileName()).toBe(test.expectedProfileName);
    expect(q.matchersString()).toBe(test.expectedMatcherString);
  });
});

test('Parse Multiline query', () => {
  expect(
    Query.parse(`memory:alloc_objects:count:space:bytes:delta{
    instance="abc"
  }`)
  ).toMatchObject(
    new Query(
      new ProfileType('memory', 'alloc_objects', 'count', 'space', 'bytes', true),
      [new Matcher('instance', MatcherTypes.MatchEqual, 'abc')],
      ''
    )
  );
});

test('SuggestEmpty', () => {
  expect(Query.suggest('')).toMatchObject([
    {
      type: 'literal',
      value: '{',
    },
    {
      type: 'profileName',
      typeahead: '',
    },
  ]);
});

test('SuggestMatcherStart', () => {
  expect(Query.suggest('{')).toMatchObject([
    {
      type: 'literal',
      value: '}',
    },
    {
      type: 'labelName',
      typeahead: '',
    },
  ]);
});

test('SuggestLabelNameStart', () => {
  expect(Query.suggest('{tes')).toMatchObject([
    {
      type: 'labelName',
      typeahead: 'tes',
    },
    {
      type: 'literal',
      value: '=',
    },
    {
      type: 'literal',
      value: '!=',
    },
    {
      type: 'literal',
      value: '=~',
    },
    {
      type: 'literal',
      value: '!~',
    },
  ]);
});

test('SuggestLabelMatcherType', () => {
  expect(Query.suggest('{test!')).toMatchObject([
    {
      type: 'literal',
      typeahead: '!',
      value: '!=',
    },
    {
      type: 'literal',
      typeahead: '!',
      value: '!~',
    },
  ]);
});

test('SuggestValueMatcherType', () => {
  expect(Query.suggest('{test=')).toMatchObject([
    {
      type: 'matcherType',
      typeahead: '=',
    },
  ]);
});

test('SuggestMatcherValue', () => {
  expect(Query.suggest('{test="')).toMatchObject([
    {
      type: 'labelValue',
      labelName: 'test',
      typeahead: '',
    },
  ]);
});

test('SuggestMatcherValueWithStart', () => {
  expect(Query.suggest('{test="a')).toMatchObject([
    {
      type: 'labelValue',
      labelName: 'test',
      typeahead: 'a',
    },
  ]);
});

test('SuggestMatcherComma', () => {
  expect(Query.suggest('{test="a"')).toMatchObject([
    {
      type: 'literal',
      value: '}',
    },
    {
      type: 'literal',
      value: ',',
    },
  ]);
});

test('SuggestNextLabelName', () => {
  expect(Query.suggest('{test="a",')).toMatchObject([
    {
      type: 'labelName',
      typeahead: '',
    },
  ]);
});

test('SuggestNextLabelNameSpace', () => {
  expect(Query.suggest('test{test="a", ')).toMatchObject([
    {
      type: 'labelName',
      typeahead: '',
    },
  ]);
});
