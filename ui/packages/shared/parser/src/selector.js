// Generated automatically by nearley, version 2.19.7
// http://github.com/Hardmath123/nearley
(function () {
  function id (x) { return x[0] }

  const moo = require('moo')

  const lexer = moo.states({
    main: {
      strstart: { match: '"', push: 'lit' },
      space: { match: /\s+/, lineBreaks: true },
      ident: /(?:[a-zA-Z_:][a-zA-Z0-9_:]*)/,
      matcherType: ['=', '!=', '=~', '!~'],
      '{': '{',
      '}': '}',
      ',': ','
    },
    lit: {
      strend: { match: '"', pop: 1 },
      constant: { match: /(?:[^"])+/, lineBreaks: true }
    }
  })

  function extractMatchers (d) {
    const matchers = [d[2]]

    for (const i in d[3]) {
      matchers.push(d[3][i][3])
    }

    return matchers
  }

  const grammar = {
    Lexer: lexer,
    ParserRules: [
      { name: 'profileSelector', symbols: ['_', 'profileName', '_'], postprocess: function (d) { return { profileName: d[1], matchers: [] } } },
      { name: 'profileSelector', symbols: ['_', 'ident', '_', 'matchers', '_'], postprocess: function (d) { return { profileName: d[1], matchers: d[3] } } },
      { name: 'profileSelector', symbols: ['_', 'matchers', '_'], postprocess: function (d) { return { profileName: {}, matchers: d[1] } } },
      { name: 'matchers', symbols: [{ literal: '{' }, '_', { literal: '}' }], postprocess: function (d) { return [] } },
      { name: 'matchers$ebnf$1', symbols: [] },
      { name: 'matchers$ebnf$1$subexpression$1', symbols: ['_', { literal: ',' }, '_', 'matcher'] },
      { name: 'matchers$ebnf$1', symbols: ['matchers$ebnf$1', 'matchers$ebnf$1$subexpression$1'], postprocess: function arrpush (d) { return d[0].concat([d[1]]) } },
      { name: 'matchers', symbols: [{ literal: '{' }, '_', 'matcher', 'matchers$ebnf$1', '_', { literal: '}' }], postprocess: extractMatchers },
      { name: 'matcher', symbols: ['labelName', '_', 'matcherType', '_', 'labelValue'], postprocess: function (d) { return { key: d[0], matcherType: d[2], value: d[4] } } },
      { name: 'string', symbols: ['strstart', 'constant', 'strend'], postprocess: function (d) { return d[1] } },
      { name: 'string', symbols: ['strstart', 'strend'], postprocess: function (d) { return { type: 'constant', value: '' } } },
      { name: 'profileName', symbols: ['ident'], postprocess: id },
      { name: 'labelName', symbols: ['ident'], postprocess: id },
      { name: 'labelValue', symbols: ['string'], postprocess: id },
      { name: 'strstart', symbols: [(lexer.has('strstart') ? { type: 'strstart' } : strstart)], postprocess: id },
      { name: 'constant', symbols: [(lexer.has('constant') ? { type: 'constant' } : constant)], postprocess: id },
      { name: 'strend', symbols: [(lexer.has('strend') ? { type: 'strend' } : strend)], postprocess: id },
      { name: 'matcherType', symbols: [{ literal: '=' }], postprocess: id },
      { name: 'matcherType', symbols: [{ literal: '!=' }], postprocess: id },
      { name: 'matcherType', symbols: [{ literal: '=~' }], postprocess: id },
      { name: 'matcherType', symbols: [{ literal: '!~' }], postprocess: id },
      { name: 'ident', symbols: [(lexer.has('ident') ? { type: 'ident' } : ident)], postprocess: id },
      { name: '_', symbols: [] },
      { name: '_', symbols: [(lexer.has('space') ? { type: 'space' } : space)], postprocess: function (d) { return null } }
    ],
    ParserStart: 'profileSelector'
  }
  if (typeof module !== 'undefined' && typeof module.exports !== 'undefined') {
    module.exports = grammar
  } else {
    window.grammar = grammar
  }
})()
