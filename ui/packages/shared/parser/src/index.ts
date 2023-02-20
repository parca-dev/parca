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

export const MatcherTypes = {
  MatchEqual: '=',
  MatchNotEqual: '!=',
  MatchRegexp: '=~',
  MatchNotRegexp: '!~',
} as const;

export type MatcherType = (typeof MatcherTypes)[keyof typeof MatcherTypes];

function matcherTypeFromString(matcherTypeString: string): MatcherType {
  switch (matcherTypeString) {
    case MatcherTypes.MatchEqual: {
      return MatcherTypes.MatchEqual;
    }
    case MatcherTypes.MatchNotEqual: {
      return MatcherTypes.MatchNotEqual;
    }
    case MatcherTypes.MatchRegexp: {
      return MatcherTypes.MatchRegexp;
    }
    case MatcherTypes.MatchNotRegexp: {
      return MatcherTypes.MatchNotRegexp;
    }
    default: {
      throw new Error('Unknown matcher type: ' + matcherTypeString);
    }
  }
}

function findMatcherLabelName(stateStack: any): string {
  let currentState = stateStack.find((e: any) => e.rule.name === 'matcher');

  if (currentState === undefined) {
    return '';
  }

  while (currentState.right.rule.name !== 'labelName') {
    currentState = currentState.left;
  }

  return currentState.right.data.value;
}

export interface LiteralSuggestion {
  type: 'literal';
  value: string;
  typeahead: string;
}

export interface MatcherTypeSuggestion {
  type: 'matcherType';
  typeahead: string;
}

export interface LabelNameSuggestion {
  type: 'labelName';
  typeahead: string;
}

export interface LabelValueSuggestion {
  type: 'labelValue';
  typeahead: string;
  labelName: string;
}

export interface ProfileNameSuggestion {
  type: 'profileName';
  typeahead: string;
}

export type Suggestion =
  | LiteralSuggestion
  | MatcherTypeSuggestion
  | LabelNameSuggestion
  | LabelValueSuggestion
  | ProfileNameSuggestion;

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
  return m.key === '__name__' && m.matcherType === MatcherTypes.MatchEqual;
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
      (e: any) =>
        new Matcher(e.key.value, matcherTypeFromString(e.matcherType.value), e.value.value)
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
    const parserTable = (p as any).table;
    const column = parserTable.filter((c: any) =>
      c.states.find(
        (s: any) =>
          s.data !== undefined &&
          s.data != null &&
          Object.prototype.hasOwnProperty.call(s.data, 'profileName')
      )
    )[0];
    if (column !== undefined) {
      const data = column.states.find(
        (s: any) =>
          s.data !== undefined && Object.prototype.hasOwnProperty.call(s.data, 'profileName')
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
    successfulParse: boolean;
  } {
    try {
      p.feed(input);
      p.save();
      return {
        successfulParse: true,
      };
    } catch (error) {
      return {
        successfulParse: false,
      };
    }
  }

  static suggest(input: string): any[] {
    const p = NewParser();
    p.save();
    const {successfulParse} = Query.tryParse(p, input);
    const parserTable = (p as any).table as any[];

    // we want the last column with states, if there is a column with no states
    // it means nothing could sucessfully be produced.
    let lastColumnIndex = parserTable.length - 1;
    for (; lastColumnIndex >= 0; lastColumnIndex--) {
      if (parserTable[lastColumnIndex].states.length > 0) {
        break;
      }
    }

    const column = parserTable[lastColumnIndex];

    const lastLexerStateIndex = parserTable
      .reverse()
      .findIndex((e: any) => e.lexerState !== undefined);
    const lastValidCursor =
      lastLexerStateIndex >= 0 ? parserTable[lastLexerStateIndex].lexerState.col - 1 : input.length;
    const rest: string = input.slice(lastValidCursor);

    // Filter out states that don't expect any more input. If the dot is within
    // the range of the list of symbols of the rule, then they are eligible.
    const expectantStates = column.states.filter(function (state: any) {
      return state.dot < state.rule.symbols.length;
    });

    // Build all the state stacks, meaning, take all the possible states and
    // for each state, walk the stack of states to the root. That way we
    // essentially have the "callstack" of states that led to this state.
    const stateStacks = expectantStates.map(function (this: any, state: any) {
      const firstStateStack = this.buildFirstStateStack(state, []);
      return firstStateStack === undefined ? [state] : firstStateStack;
    }, p);

    const suggestions: Suggestion[] = [];

    const prevLabelNameStates = column.states.filter(
      (e: any) => e.rule.name === 'labelName' && e.isComplete
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

    stateStacks.forEach(function (stateStack: any[]) {
      const state = stateStack[0];
      const nextSymbol = state.rule.symbols[state.dot];

      // We're not going to skip suggesting to type a whitespace character.
      if (!(nextSymbol.type !== undefined && nextSymbol.type === 'space')) {
        if (nextSymbol.literal !== undefined) {
          const suggestedValue = nextSymbol.literal as string;
          if (
            suggestions.findIndex(s => s.type === 'literal' && s.value === suggestedValue) === -1
          ) {
            if (successfulParse || suggestedValue.startsWith(rest)) {
              suggestions.push({type: 'literal', value: suggestedValue, typeahead: rest});
            }
          }
        }

        // Find the high level concept that we can complete.
        // For an ident type, those can be: profileName, labelName or labelValue.
        const types = ['profileName', 'labelName', 'labelValue'];

        if (nextSymbol.type !== undefined && nextSymbol.type === 'ident') {
          const found = state.wantedBy.filter((e: any) => types.includes(e.rule.name));
          const s = found === undefined ? [] : found;
          s.map((e: any) => e.rule.name).forEach(function (e: any) {
            const suggestion = {type: e, typeahead: ''};
            suggestions.push(suggestion);
          });
        }

        // Matcher type is unambiguous, so we can go ahead and check if
        // the label name may be incomplete and suggest any matcher.
        if (nextSymbol.type !== undefined && nextSymbol.type === 'matcherType') {
          suggestions.push({
            type: 'matcherType',
            typeahead: rest,
          });
        }

        // A valid strstart always means a label value.
        if (nextSymbol.type !== undefined && nextSymbol.type === 'strstart') {
          const prevMatcherTypeStates = column.states.filter(
            (e: any) => e.rule.name === 'matcherType' && e.isComplete
          );
          if (prevMatcherTypeStates.length > 0 && prevMatcherTypeStates[0].data !== undefined) {
            suggestions.push({
              type: 'matcherType',
              typeahead: prevMatcherTypeStates[0].data.value,
            });
          }
        }

        if (nextSymbol.type !== undefined && nextSymbol.type === 'constant') {
          suggestions.push({
            type: 'labelValue',
            labelName: findMatcherLabelName(stateStack),
            typeahead: '',
          });
        }

        if (nextSymbol.type !== undefined && nextSymbol.type === 'strend') {
          const prevConstStates = column.states.filter(
            (e: any) => e.rule.name === 'constant' && e.isComplete
          );
          if (prevConstStates.length > 0 && prevConstStates[0].data !== undefined) {
            suggestions.push({
              type: 'labelValue',
              labelName: findMatcherLabelName(stateStack),
              typeahead: `${prevConstStates[0].data.value as string}`,
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
    const matcher = new Matcher(key, MatcherTypes.MatchEqual, value);
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

function stripCurlyBrackets(input: string): string {
  const withoutStartingCurly = input.startsWith('{') ? input.slice(1) : input;
  const withoutEndingCurly = withoutStartingCurly.endsWith('}')
    ? withoutStartingCurly.slice(0, withoutStartingCurly.length - 1)
    : withoutStartingCurly;

  return withoutEndingCurly;
}
