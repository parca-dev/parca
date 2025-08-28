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
  ProfileDiffSelection,
  ProfileDiffSelection_Mode,
  QueryRequest,
  QueryRequest_Mode,
  QueryRequest_ReportType,
  Timestamp,
} from '@parca/client';
import {Matcher, NewParser, ProfileType, Query} from '@parca/parser';
import {formatDate, formatDuration} from '@parca/utilities';

export interface ProfileSource {
  QueryRequest: () => QueryRequest;
  ProfileType: () => ProfileType;
  DiffSelection: () => ProfileDiffSelection;
  toString: (timezone?: string) => string;
  toKey: () => string;
}

export interface ProfileSelection {
  ProfileName: () => string;
  HistoryParams: () => {[key: string]: any};
  ProfileSource: () => ProfileSource;
  Type: () => string;
}

export const timeFormat = (timezone?: string): string => {
  if (timezone !== undefined) {
    return 'yyyy-MM-dd HH:mm:ss';
  }

  return "yyyy-MM-dd HH:mm:ss '(UTC)'";
};

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

export function ProfileSelectionFromParams(
  mergeFrom: string | undefined,
  mergeTo: string | undefined,
  selection: string | undefined
): ProfileSelection | null {
  if (
    mergeFrom !== undefined &&
    mergeTo !== undefined &&
    selection !== undefined &&
    selection !== ''
  ) {
    const p = NewParser();
    p.save();
    const {successfulParse} = Query.tryParse(p, selection);

    if (!successfulParse) {
      console.log('Failed to parse selected query.');
      console.log(selection);
      return null;
    }

    return new MergedProfileSelection(BigInt(mergeFrom), BigInt(mergeTo), Query.parse(selection));
  }

  return null;
}

export class MergedProfileSelection implements ProfileSelection {
  mergeFrom: bigint;
  mergeTo: bigint;
  query: Query;
  profileSource: ProfileSource;

  constructor(mergeFrom: bigint, mergeTo: bigint, query: Query) {
    this.mergeFrom = mergeFrom;
    this.mergeTo = mergeTo;
    this.query = query;
    this.profileSource = new MergedProfileSource(this.mergeFrom, this.mergeTo, this.query);
  }

  ProfileName(): string {
    return this.query.profileName();
  }

  HistoryParams(): {[key: string]: any} {
    return {
      merge_from: this.mergeFrom.toString(),
      merge_to: this.mergeTo.toString(),
      selection: this.query.toString(),
    };
  }

  Type(): string {
    return 'merge';
  }

  ProfileSource(): ProfileSource {
    return this.profileSource;
  }
}

export class ProfileDiffSource implements ProfileSource {
  a: ProfileSource;
  b: ProfileSource;
  profileType: ProfileType;
  absolute?: boolean;

  constructor(a: ProfileSource, b: ProfileSource, absolute?: boolean) {
    this.a = a;
    this.b = b;
    this.profileType = a.ProfileType();
    this.absolute = absolute;
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
          absolute: this.absolute,
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_ARROW,
      mode: QueryRequest_Mode.DIFF,
      filter: [],
    };
  }

  ProfileType(): ProfileType {
    return this.profileType;
  }

  Describe(): JSX.Element {
    return (
      <>
        <p>Browse the comparison</p>
      </>
    );
  }

  toString(): string {
    const aDesc = this.a.toString();
    const bDesc = this.b.toString();

    if (aDesc === bDesc) {
      return 'Profile comparison';
    }

    return `${this.a.toString()} compared with ${this.b.toString()}`;
  }

  toKey(): string {
    return `${this.a.toKey()}-${this.b.toKey()}`;
  }
}

function nanosToTimestamp(nanos: bigint): Timestamp {
  const NANOS_PER_SECOND = 1_000_000_000n;

  const seconds = nanos / NANOS_PER_SECOND;
  const remainingNanos = nanos % NANOS_PER_SECOND;

  return {
    seconds,
    nanos: Number(remainingNanos), // Safe since remainingNanos < 1e9
  };
}

export class MergedProfileSource implements ProfileSource {
  mergeFrom: bigint;
  mergeTo: bigint;
  query: Query;
  profileType: ProfileType;

  constructor(mergeFrom: bigint, mergeTo: bigint, query: Query) {
    this.mergeFrom = mergeFrom;
    this.mergeTo = mergeTo;
    this.query = query;
    this.profileType = ProfileType.fromString(Query.parse(this.query.toString()).profileName());
  }

  DiffSelection(): ProfileDiffSelection {
    return {
      options: {
        oneofKind: 'merge',
        merge: {
          start: nanosToTimestamp(this.mergeFrom),
          end: nanosToTimestamp(this.mergeTo),
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
          start: nanosToTimestamp(this.mergeFrom),
          end: nanosToTimestamp(this.mergeTo),
          query: this.query.toString(),
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_ARROW,
      mode: QueryRequest_Mode.MERGE,
      filter: [],
    };
  }

  ProfileType(): ProfileType {
    return this.profileType;
  }

  stringMatchers(): string[] {
    return this.query.matchers
      .filter((m: Matcher) => m.key !== '__name__')
      .map((m: Matcher) => `${m.key}=${m.value}`);
  }

  toString(timezone?: string): string {
    let queryPart = '';
    if (this.query.toString()?.length > 0) {
      queryPart = ` of query "${this.query.toString()}"`;
    }

    let timePart = '';
    if (this.mergeFrom !== 0n) {
      timePart = `over ${formatDuration({
        nanos: Number(this.mergeTo - this.mergeFrom),
      })} from ${formatDate(this.mergeFrom, timeFormat(timezone), timezone)} to ${formatDate(
        this.mergeTo,
        timeFormat(timezone),
        timezone
      )}`;
    }

    return `Merged profiles${queryPart}${timePart}`;
  }

  toKey(): string {
    return `${this.mergeFrom.toString()}-${this.mergeTo.toString()}-${this.query.toString()}`;
  }
}
