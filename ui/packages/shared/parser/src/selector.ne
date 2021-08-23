@{%

const moo = require('moo')

let lexer = moo.states({
  main: {
    strstart: {match: '"', push: 'lit'},
    space: {match: /\s+/, lineBreaks: true},
    ident: /(?:[a-zA-Z_:][a-zA-Z0-9_:]*)/,
    matcherType: ['=', '!=', '=~', '!~'],
    '{': '{',
    '}': '}',
    ',': ',',
  },
  lit: {
    strend:   {match: '"', pop: 1},
    constant: {match: /(?:[^"])+/, lineBreaks: true},
  },
});

function extractMatchers(d) {
    let matchers = [d[2]];

    for (let i in d[3]) {
        matchers.push(d[3][i][3]);
    }

    return matchers;
}

%}

@lexer lexer

profileSelector -> _ profileName _ {% function(d) { return { 'profileName': d[1], 'matchers': [] }; } %}
    | _ ident _ matchers _ {% function(d) { return { 'profileName': d[1], 'matchers': d[3] }; } %}
    | _ matchers _ {% function(d) { return { 'profileName': {}, 'matchers': d[1] }; } %}

matchers -> "{" _ "}" {% function(d) { return []; } %}
    | "{" _ matcher (_ "," _ matcher):* _ "}" {% extractMatchers %}

matcher -> labelName _ matcherType _ labelValue {% function(d) { return { 'key': d[0], 'matcherType': d[2], 'value': d[4] }; } %}

string ->
       strstart constant strend {% function(d) { return d[1]; } %}
       | strstart strend {% function(d) { return {type: 'constant', value: ''}; } %}

profileName -> ident {% id %}
labelName -> ident {% id %}
labelValue -> string {% id %}

strstart -> %strstart {% id %}
constant -> %constant {% id %}
strend -> %strend {% id %}

matcherType ->
            "=" {% id %}
            | "!=" {% id %}
            | "=~" {% id %}
            | "!~" {% id %}
ident -> %ident {% id %}

_ -> null | %space {% function(d) { return null; } %}
