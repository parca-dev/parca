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

import {
  Label,
  ProfileDiffSelection,
  ProfileDiffSelection_Mode,
  QueryRequest,
  QueryRequest_Mode,
  QueryRequest_ReportType,
  Timestamp,
} from '@parca/client';
import {Matcher, ProfileType, Query} from '@parca/parser';
import {formatDate} from '@parca/utilities';

export interface ProfileSource {
  QueryRequest: () => QueryRequest;
  ProfileType: () => ProfileType;
  DiffSelection: () => ProfileDiffSelection;
  Describe: () => JSX.Element;
  toString: () => string;
}

export interface ProfileSelection {
  ProfileName: () => string;
  HistoryParams: () => {[key: string]: any};
  ProfileSource: () => ProfileSource;
  Type: () => string;
}

export const timeFormat = "MMM d, 'at' h:mm:s a '(UTC)'";
export const timeFormatShort = 'MMM d, h:mma';

export function ParamsString(params: {[key: string]: string}): string {
  return Object.keys(params)
    .map(function (key) {
      return `${key}=${params[key]}`;
    })
    .join('&');
}

export function SuffixParams(params: {[key: string]: any}, suffix: string): {[key: string]: any} {
  return Object.fromEntries(
    Object.entries(params).map(([key, value]) => [`${key}${suffix}`, value])
  );
}

export function ParseLabels(labels: string[]): Label[] {
  return labels
    .filter(str => str !== '')
    .map(function (labelString): Label {
      const parts = labelString.split('=', 2);
      return {name: parts[0], value: parts[1]};
    });
}

export function ProfileSelectionFromParams(
  expression: string | undefined,
  from: string | undefined,
  to: string | undefined,
  mergeFrom: string | undefined,
  mergeTo: string | undefined,
  labels: string[],
  filterQuery?: string
): ProfileSelection | null {
  if (
    from !== undefined &&
    to !== undefined &&
    mergeFrom !== undefined &&
    mergeTo !== undefined &&
    expression !== undefined
  ) {
    // TODO: Refactor parsing the query and adding matchers
    let query = Query.parse(expression);
    if (labels !== undefined) {
      ParseLabels(labels ?? ['']).forEach(l => {
        const hasLabels = labels.length > 0 && labels.filter(val => val !== '').length > 0;
        if (hasLabels) {
          const [newQuery, changed] = query.setMatcher(l.name, l.value);
          if (changed) {
            query = newQuery;
          }
        }
      });
    }

    return new MergedProfileSelection(parseInt(mergeFrom), parseInt(mergeTo), query, filterQuery);
  }

  return null;
}

export class MergedProfileSelection implements ProfileSelection {
  mergeFrom: number;
  mergeTo: number;
  query: Query;
  filterQuery: string | undefined;

  constructor(mergeFrom: number, mergeTo: number, query: Query, filterQuery?: string) {
    this.mergeFrom = mergeFrom;
    this.mergeTo = mergeTo;
    this.query = query;
    this.filterQuery = filterQuery;
  }

  ProfileName(): string {
    return this.query.profileName();
  }

  HistoryParams(): {[key: string]: any} {
    return {
      merge_from: this.mergeFrom.toString(),
      merge_to: this.mergeTo.toString(),
      query: this.query,
      profile_name: this.ProfileName(),
      labels: this.query.matchers.map(m => `${m.key}=${encodeURIComponent(m.value)}`),
    };
  }

  Type(): string {
    return 'merge';
  }

  ProfileSource(): ProfileSource {
    return new MergedProfileSource(this.mergeFrom, this.mergeTo, this.query, this.filterQuery);
  }
}

export class ProfileDiffSource implements ProfileSource {
  a: ProfileSource;
  b: ProfileSource;
  filterQuery: string | undefined;

  constructor(a: ProfileSource, b: ProfileSource, filterQuery?: string) {
    this.a = a;
    this.b = b;
    this.filterQuery = filterQuery;
  }

  DiffSelection(): ProfileDiffSelection {
    throw new Error('Method not implemented.');
  }

  QueryRequest(): QueryRequest {
    return {
      options: {
        oneofKind: 'diff',
        diff: {
          a: this.a.DiffSelection(),
          b: this.b.DiffSelection(),
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED,
      mode: QueryRequest_Mode.DIFF,
      filterQuery: this.filterQuery,
    };
  }

  ProfileType(): ProfileType {
    return this.a.ProfileType();
  }

  Describe(): JSX.Element {
    return (
      <>
        <p>Browse the comparison</p>
      </>
    );
  }

  toString(): string {
    return `${this.a.toString()} compared with ${this.b.toString()}`;
  }
}

export class MergedProfileSource implements ProfileSource {
  mergeFrom: number;
  mergeTo: number;
  query: Query;
  filterQuery: string | undefined;

  constructor(mergeFrom: number, mergeTo: number, query: Query, filterQuery?: string) {
    this.mergeFrom = mergeFrom;
    this.mergeTo = mergeTo;
    this.query = query;
    this.filterQuery = filterQuery;
  }

  DiffSelection(): ProfileDiffSelection {
    return {
      options: {
        oneofKind: 'merge',
        merge: {
          start: Timestamp.fromDate(new Date(this.mergeFrom)),
          end: Timestamp.fromDate(new Date(this.mergeTo)),
          query: this.query.toString(),
        },
      },
      mode: ProfileDiffSelection_Mode.MERGE,
    };
  }

  QueryRequest(): QueryRequest {
    return {
      options: {
        oneofKind: 'merge',
        merge: {
          start: Timestamp.fromDate(new Date(this.mergeFrom)),
          end: Timestamp.fromDate(new Date(this.mergeTo)),
          query: this.query.toString(),
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED,
      mode: QueryRequest_Mode.MERGE,
      filterQuery: this.filterQuery,
    };
  }

  ProfileType(): ProfileType {
    return ProfileType.fromString(Query.parse(this.query.toString()).profileName());
  }

  Describe(): JSX.Element {
    return (
      <a>
        Merge of &quot;{this.query.toString()}&quot; from {formatDate(this.mergeFrom, timeFormat)}{' '}
        to {formatDate(this.mergeTo, timeFormat)}
      </a>
    );
  }

  stringMatchers(): string[] {
    return this.query.matchers
      .filter((m: Matcher) => m.key !== '__name__')
      .map((m: Matcher) => `${m.key}=${m.value}`);
  }

  toString(): string {
    return `merged profiles of query "${this.query.toString()}" from ${formatDate(
      this.mergeFrom,
      timeFormat
    )} to ${formatDate(this.mergeTo, timeFormat)}`;
  }
}
