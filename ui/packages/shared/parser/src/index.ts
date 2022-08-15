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

import {Grammar, Parser} from 'nearley';
import grammar from './selector';

export function NewParser(): Parser {
  return new Parser(Grammar.fromCompiled(grammar), {keepHistory: true});
}

export enum MatcherType {
  MatchEqual = '=',
  MatchNotEqual = '!=',
  MatchRegexp = '=~',
  MatchNotRegexp = '!~',
}

function matcherTypeFromString(matcherTypeString: string): MatcherType {
  switch (matcherTypeString) {
    case MatcherType.MatchEqual: {
      return MatcherType.MatchEqual;
    }
    case MatcherType.MatchNotEqual: {
      return MatcherType.MatchNotEqual;
    }
    case MatcherType.MatchRegexp: {
      return MatcherType.MatchRegexp;
    }
    case MatcherType.MatchNotRegexp: {
      return MatcherType.MatchNotRegexp;
    }
    default: {
      throw new Error('Unknown matcher type: ' + matcherTypeString);
    }
  }
}

export class Matcher {
  key: string;
  matcherType: MatcherType;
  value: string;

  constructor(key: string, matcherType: MatcherType, value: string) {
    this.key = key;
    this.matcherType = matcherType;
    this.value = value;
  }

  toString(): string {
    return `${this.key}${this.matcherType}"${this.value}"`;
  }
}

function isProfileNameMatcher(m: Matcher): boolean {
  return m.key === '__name__' && m.matcherType === MatcherType.MatchEqual;
}

export class ProfileType {
  profileName: string;
  sampleType: string;
  sampleUnit: string;
  periodType: string;
  periodUnit: string;
  delta: boolean;

  constructor(
    profileName: string,
    sampleType: string,
    sampleUnit: string,
    periodType: string,
    periodUnit: string,
    delta: boolean
  ) {
    this.profileName = profileName;
    this.sampleType = sampleType;
    this.sampleUnit = sampleUnit;
    this.periodType = periodType;
    this.periodUnit = periodUnit;
    this.delta = delta;
  }

  static fromString(profileType: string): ProfileType {
    const str = profileType.toString();
    const parts = str.split(':');
    if (parts.length !== 5 && parts.length !== 6) {
      throw new Error('Invalid profile type: ' + str);
    }
    return new ProfileType(parts[0], parts[1], parts[2], parts[3], parts[4], parts[5] === 'delta');
  }

  toString(): string {
    if (
      this.profileName === '' &&
      this.sampleType === '' &&
      this.sampleUnit === '' &&
      this.periodType === '' &&
      this.periodUnit === ''
    ) {
      return '';
    }
    return `${this.profileName}:${this.sampleType}:${this.sampleUnit}:${this.periodType}:${
      this.periodUnit
    }${this.delta ? ':delta' : ''}`;
  }
}

export class Query {
  profType: ProfileType;
  matchers: Matcher[];
  inputMatcherString: string;

  constructor(profileType: ProfileType, matchers: Matcher[], inputMatcherString: string) {
    this.profType = profileType;
    this.matchers = matchers;
    this.inputMatcherString = inputMatcherString;
  }

  static fromAst(ast: any): Query {
    if (ast === undefined || ast == null) {
      return new Query(new ProfileType('', '', '', '', '', false), [], '');
    }

    const matchers = ast.matchers.map(
      e => new Matcher(e.key.value, matcherTypeFromString(e.matcherType.value), e.value.value)
    );
    return new Query(ProfileType.fromString(ast.profileName.value), matchers, '');
  }

  static parse(input: string): Query {
    const p = NewParser();
    p.save();
    try {
      p.feed(input);
      p.save();
    } catch (error) {
      // do nothing... this means we've got an incomplete or entirely incorrect query
    }

    if (p.results?.length > 0) {
      return Query.fromAst(p.results[0]);
    }

    // partial parse result, also ok, we'll try our best with it :)

    // Parser.table is not defined in the type definitions, so we need to do this unfortunately.
    const parserTable = p.table as any[];
    const column = parserTable.filter(c =>
      c.states.find(
        s =>
          s.data !== undefined &&
          s.data != null &&
          Object.prototype.hasOwnProperty.call(s.data, 'profileName')
      )
    )[0];
    if (column !== undefined) {
      const data = column.states.find(
        s => s.data !== undefined && Object.prototype.hasOwnProperty.call(s.data, 'profileName')
      ).data;
      const rest = input.slice(column.lexerState.col - 2);
      return new Query(
        ProfileType.fromString(data.profileName),
        [],
        rest.length > 0 ? rest : input
      );
    }

    return new Query(new ProfileType('', '', '', '', '', false), [], '');
  }

  static tryParse(
    p: Parser,
    input: string
  ): {
    lastIndex: number;
    successfulParse: boolean;
  } {
    try {
      p.feed(input);
      p.save();
      return {
        successfulParse: true,
        lastIndex: p.table.length - 1,
      };
    } catch (error) {
      return {
        successfulParse: false,
        lastIndex: p.table.length - 2,
      };
    }
  }

  static suggest(input: string): any[] {
    const p = NewParser();
    p.save();
    const {lastIndex, successfulParse} = Query.tryParse(p, input);

    const parserTable = p.table;
    const column = parserTable[lastIndex];
    const lastLexerStateIndex = parserTable.reverse().findIndex(e => e.lexerState !== undefined);
    const lastValidCursor =
      lastLexerStateIndex >= 0 ? parserTable[lastLexerStateIndex].lexerState.col - 1 : input.length;
    const rest: string = input.slice(lastValidCursor);

    const expectantStates = column.states.filter(function (state) {
      const nextSymbol = state.rule.symbols[state.dot];
      return nextSymbol;
    });

    const stateStacks = expectantStates.map(function (state) {
      const firstStateStack = this.buildFirstStateStack(state, []);
      return firstStateStack === undefined ? [state] : firstStateStack;
    }, p);

    const suggestions: any[] = [];

    const prevLabelNameStates = column.states.filter(
      e => e.rule.name === 'labelName' && e.isComplete
    );

    if (
      successfulParse &&
      prevLabelNameStates.length > 0 &&
      prevLabelNameStates[0].data !== undefined
    ) {
      suggestions.push({
        type: 'labelName',
        typeahead: prevLabelNameStates[0].data.value,
      });
    }

    stateStacks.forEach(function (stateStack) {
      const state = stateStack[0];
      const nextSymbol = state.rule.symbols[state.dot];

      // We're not going to skip suggesting to type a whitespace character.
      if (!(nextSymbol.type !== undefined && nextSymbol.type === 'space')) {
        if (nextSymbol.literal !== undefined) {
          const suggestion = {type: 'literal', value: nextSymbol.literal as string};
          if (
            suggestions.findIndex(s => s.type === 'literal' && s.value === suggestion.value) === -1
          ) {
            if (successfulParse || suggestion.value.startsWith(rest)) {
              suggestions.push(suggestion);
            }
          }
        }

        // Find the high level concept that we can complete.
        // For an ident type, those can be: profileName, labelName or labelValue.
        const types = ['profileName', 'labelName', 'labelValue'];

        if (nextSymbol.type !== undefined && nextSymbol.type === 'ident') {
          const found = state.wantedBy.filter(e => types.includes(e.rule.name));
          const s = found === undefined ? [] : found;
          s.map(e => e.rule.name).forEach(function (e) {
            const suggestion = {type: e, typeahead: ''};
            suggestions.push(suggestion);
          });
        }

        // Matcher type is unambiguous, so we can go ahead and check if
        // the label name may be incomplete and suggest any matcher.
        if (nextSymbol.type !== undefined && nextSymbol.type === 'matcherType') {
          const suggestion = {
            type: 'matcherType',
            typeahead: rest,
          };
          suggestions.push(suggestion);
        }

        // A valid strstart always means a label value.
        if (nextSymbol.type !== undefined && nextSymbol.type === 'strstart') {
          const prevMatcherTypeStates = column.states.filter(
            e => e.rule.name === 'matcherType' && e.isComplete
          );
          if (prevMatcherTypeStates.length > 0 && prevMatcherTypeStates[0].data !== undefined) {
            suggestions.push({
              type: 'matcherType',
              typeahead: prevMatcherTypeStates[0].data.value,
            });
          }

          suggestions.push({
            type: 'labelValue',
            typeahead: '',
          });
        }

        if (nextSymbol.type !== undefined && nextSymbol.type === 'constant') {
          suggestions.push({
            type: 'labelValue',
            typeahead: '"',
          });
        }

        if (nextSymbol.type !== undefined && nextSymbol.type === 'strend') {
          const prevConstStates = column.states.filter(
            e => e.rule.name === 'constant' && e.isComplete
          );
          if (prevConstStates.length > 0 && prevConstStates[0].data !== undefined) {
            suggestions.push({
              type: 'labelValue',
              typeahead: `"${prevConstStates[0].data.value as string}`,
            });
          }
        }
      }
    });

    return suggestions;
  }

  setMatcher(key: string, value: string): [Query, boolean] {
    if (this.matchers.find(e => e.key === key && e.value === `"${value}"`) != null) {
      return [this, false];
    }
    const matcher = new Matcher(key, MatcherType.MatchEqual, value);
    if (this.matchers.find(e => e.key === key) != null) {
      return [
        new Query(
          this.profType,
          this.matchers.map(e => (e.key === key ? matcher : e)),
          ''
        ),
        true,
      ];
    }
    return [new Query(this.profType, this.matchers.concat([matcher]), ''), true];
  }

  setProfileName(name: string): [Query, boolean] {
    const profileType = ProfileType.fromString(name);
    if (this.inputMatcherString !== undefined && this.inputMatcherString.length > 0) {
      return [new Query(profileType, this.matchers, this.inputMatcherString), true];
    }
    if (this.profType === profileType) {
      return [this, false];
    }
    return [new Query(profileType, this.matchers, this.inputMatcherString), true];
  }

  profileName(): string {
    return this.profType.toString();
  }

  profileType(): ProfileType {
    return this.profType;
  }

  nonProfileNameMatchers(): Matcher[] {
    return this.matchers.filter(m => !isProfileNameMatcher(m));
  }

  matchersString(): string {
    if (this.inputMatcherString !== undefined && this.inputMatcherString.length > 0) {
      return stripCurlyBrackets(this.inputMatcherString);
    }
    const m = this.nonProfileNameMatchers();
    return m.length > 0 ? m.map(m => m.toString()).join(', ') : '';
  }

  toString(): string {
    return `${this.profileName()}{${this.matchersString()}}`;
  }
}

function stripCurlyBrackets(input: string) {
  const withoutStartingCurly = input.startsWith('{') ? input.slice(1) : input;
  const withoutEndingCurly = withoutStartingCurly.endsWith('}')
    ? withoutStartingCurly.slice(0, withoutStartingCurly.length - 1)
    : withoutStartingCurly;

  return withoutEndingCurly;
}
