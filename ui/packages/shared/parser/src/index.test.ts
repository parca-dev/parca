import { MatcherType, Matcher, Query } from './index'

test('QueryParseEmpty', () => {
  expect(Query.parse('')).toMatchObject(new Query([], ''))
})

test('QueryParseProfile', () => {
  expect(Query.parse('heap')).toMatchObject(new Query([
    new Matcher('__name__', MatcherType.MatchEqual, 'heap')
  ], ''))
})

test('QueryParseWithMatcher', () => {
  expect(Query.parse('heap{instance="abc"}')).toMatchObject(new Query([
    new Matcher('__name__', MatcherType.MatchEqual, 'heap'),
    new Matcher('instance', MatcherType.MatchEqual, 'abc')
  ], ''))
})

test('Query.toString', () => {
  expect(Query.parse('heap{instance="abc"}').toString()).toBe('heap{instance="abc"}')
  expect(Query.parse('{i}').toString()).toBe('{i}')
})

test('Partial Parsing ProfileName and rest', () => {
  [{
    input: 'threadcreate{instance="abc",a',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'instance="abc",a'
  }, {
    input: 'threadcreate{instance="abc",',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'instance="abc",'
  }, {
    input: 'threadcreate{instance="ab',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'instance="ab'
  }, {
    input: 'threadcreate{instance="',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'instance="'
  }, {
    input: 'threadcreate{instance=a',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'instance=a'
  }, {
    input: 'threadcreate{=a',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: '=a'
  }, {
    input: 'threadcreate{a',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: 'a'
  }, {
    input: 'threadcreate{',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: ''
  }, {
    input: 'threadcreate',
    expectedProfileName: 'threadcreate',
    expectedMatcherString: ''
  }, {
    input: '{job=}',
    expectedProfileName: '',
    expectedMatcherString: 'job='
  }].forEach(function (test) {
    const q = Query.parse(test.input)

    expect(q.profileName()).toBe(test.expectedProfileName)
    expect(q.matchersString()).toBe(test.expectedMatcherString)
  })
})

test('SuggestEmpty', () => {
  expect(Query.suggest('')).toMatchObject([{
    type: 'literal',
    value: '{'
  }, {
    type: 'profileName',
    typeahead: ''
  }])
})

test('SuggestMatcherStart', () => {
  expect(Query.suggest('{')).toMatchObject([{
    type: 'literal',
    value: '}'
  }, {
    type: 'labelName',
    typeahead: ''
  }])
})

test('SuggestLabelNameStart', () => {
  expect(Query.suggest('{tes')).toMatchObject([{
    type: 'labelName',
    typeahead: 'tes'
  }, {
    type: 'literal',
    value: '='
  }, {
    type: 'literal',
    value: '!='
  }, {
    type: 'literal',
    value: '=~'
  }, {
    type: 'literal',
    value: '!~'
  }])
})

test('SuggestLabelMatcherType', () => {
  expect(Query.suggest('{test!')).toMatchObject([{
    type: 'literal',
    value: '!='
  }, {
    type: 'literal',
    value: '!~'
  }])
})

test('SuggestValueMatcherType', () => {
  expect(Query.suggest('{test=')).toMatchObject([{
    type: 'matcherType',
    typeahead: '='
  }, {
    type: 'labelValue',
    typeahead: ''
  }])
})

test('SuggestMatcherValue', () => {
  expect(Query.suggest('{test="')).toMatchObject([{
    type: 'labelValue',
    typeahead: '"'
  }])
})

test('SuggestMatcherValueWithStart', () => {
  expect(Query.suggest('{test="a')).toMatchObject([{
    type: 'labelValue',
    typeahead: '"a'
  }])
})

test('SuggestMatcherComma', () => {
  expect(Query.suggest('{test="a"')).toMatchObject([{
    type: 'literal',
    value: '}'
  }, {
    type: 'literal',
    value: ','
  }])
})

test('SuggestNextLabelName', () => {
  expect(Query.suggest('{test="a",')).toMatchObject([{
    type: 'labelName',
    typeahead: ''
  }])
})

test('SuggestNextLabelNameSpace', () => {
  expect(Query.suggest('test{test="a", ')).toMatchObject([{
    type: 'labelName',
    typeahead: ''
  }])
})
