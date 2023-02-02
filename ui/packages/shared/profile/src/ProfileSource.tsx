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

import {formatDate} from '@parca/functions';
import {Query, ProfileType} from '@parca/parser';
import {
  Label,
  QueryRequest,
  QueryRequest_Mode,
  QueryRequest_ReportType,
  ProfileDiffSelection,
  ProfileDiffSelection_Mode,
  Timestamp,
} from '@parca/client';

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
  return labels.map(function (labelString): Label {
    const parts = labelString.split('=', 2);
    return {name: parts[0], value: parts[1]};
  });
}

export function ProfileSelectionFromParams(
  expression: string | undefined,
  from: string | undefined,
  to: string | undefined,
  merge_from: string | undefined,
  merge_to: string | undefined,
  labels: string[],
  filterQuery?: string
): ProfileSelection | null {
  if (
    from !== undefined &&
    to !== undefined &&
    merge_from !== undefined &&
    merge_to !== undefined &&
    expression !== undefined
  ) {
    return new MergedProfileSelection(
      parseInt(merge_from),
      parseInt(merge_to),
      ParseLabels(labels ?? ['']),
      expression,
      filterQuery
    );
  }

  return null;
}

export class MergedProfileSelection implements ProfileSelection {
  merge_from: number;
  merge_to: number;
  query: string;
  filterQuery: string | undefined;
  labels: Label[];

  constructor(
    merge_from: number,
    merge_to: number,
    labels: Label[],
    query: string,
    filterQuery?: string
  ) {
    this.merge_from = merge_from;
    this.merge_to = merge_to;
    this.query = query;
    this.filterQuery = filterQuery;
    this.labels = labels;
  }

  ProfileName(): string {
    return Query.parse(this.query).profileName();
  }

  HistoryParams(): {[key: string]: any} {
    return {
      merge_from: this.merge_from.toString(),
      merge_to: this.merge_to.toString(),
      query: this.query,
      profile_name: this.ProfileName(),
      labels: this.labels.map(label => `${label.name}=${encodeURIComponent(label.value)}`),
    };
  }

  Type(): string {
    return 'merge';
  }

  ProfileSource(): ProfileSource {
    return new MergedProfileSource(
      this.merge_from,
      this.merge_to,
      this.labels,
      this.query,
      this.filterQuery
    );
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
  merge_from: number;
  merge_to: number;
  labels: Label[];
  query: string;
  filterQuery: string | undefined;

  constructor(
    merge_from: number,
    merge_to: number,
    labels: Label[],
    query: string,
    filterQuery?: string
  ) {
    this.merge_from = merge_from;
    this.merge_to = merge_to;
    this.labels = labels;
    this.query = query;
    this.filterQuery = filterQuery;
  }

  DiffSelection(): ProfileDiffSelection {
    return {
      options: {
        oneofKind: 'merge',
        merge: {
          start: Timestamp.fromDate(new Date(this.merge_from)),
          end: Timestamp.fromDate(new Date(this.merge_to)),
          query: this.query,
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
          start: Timestamp.fromDate(new Date(this.merge_from)),
          end: Timestamp.fromDate(new Date(this.merge_to)),
          query: this.query,
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED,
      mode: QueryRequest_Mode.MERGE,
      filterQuery: this.filterQuery,
    };
  }

  ProfileType(): ProfileType {
    return ProfileType.fromString(Query.parse(this.query).profileName());
  }

  Describe(): JSX.Element {
    return (
      <a>
        Merge of &quot;{this.query}&quot; from {formatDate(this.merge_from, timeFormat)} to{' '}
        {formatDate(this.merge_to, timeFormat)}
      </a>
    );
  }

  stringLabels(): string[] {
    return this.labels
      .filter((label: Label) => label.name !== '__name__')
      .map((label: Label) => `${label.name}=${label.value}`);
  }

  toString(): string {
    return `merged profiles of query "${this.query}" from ${formatDate(
      this.merge_from,
      timeFormat
    )} to ${formatDate(this.merge_to, timeFormat)}`;
  }
}
