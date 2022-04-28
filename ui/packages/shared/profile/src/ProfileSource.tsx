import React from 'react';
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
  merge: string | undefined,
  labels: string[] | undefined,
  profileName: string | undefined,
  time: string | undefined
): ProfileSelection | null {
  if (
    merge !== undefined &&
    merge === 'true' &&
    from !== undefined &&
    to !== undefined &&
    expression !== undefined
  ) {
    return new MergedProfileSelection(parseInt(from), parseInt(to), expression);
  }
  if (labels !== undefined && time !== undefined && profileName !== undefined) {
    return new SingleProfileSelection(profileName, ParseLabels(labels), parseInt(time));
  }
  return null;
}

export class SingleProfileSelection implements ProfileSelection {
  profileName: string;
  labels: Label[];
  time: number;

  constructor(profileName: string, labels: Label[], time: number) {
    this.profileName = profileName;
    this.labels = labels;
    this.time = time;
  }

  ProfileName(): string {
    return this.profileName;
  }

  HistoryParams(): {[key: string]: any} {
    return {
      profile_name: this.profileName,
      labels: this.labels.map(label => `${label.name}=${label.value}`),
      time: this.time,
    };
  }

  Type(): string {
    return 'single';
  }

  ProfileSource(): ProfileSource {
    return new SingleProfileSource(this.profileName, this.labels, this.time);
  }
}

export class MergedProfileSelection implements ProfileSelection {
  from: number;
  to: number;
  query: string;

  constructor(from: number, to: number, query: string) {
    this.from = from;
    this.to = to;
    this.query = query;
  }

  ProfileName(): string {
    return Query.parse(this.query).profileName();
  }

  HistoryParams(): {[key: string]: string} {
    return {
      mode: 'merge',
      from: this.from.toString(),
      to: this.to.toString(),
      query: this.query,
    };
  }

  Type(): string {
    return 'merge';
  }

  ProfileSource(): ProfileSource {
    return new MergedProfileSource(this.from, this.to, this.query);
  }
}

export class SingleProfileSource implements ProfileSource {
  profName: string;
  labels: Label[];
  time: number;

  constructor(profileName: string, labels: Label[], time: number) {
    this.profName = profileName;
    this.labels = labels;
    this.time = time;
  }

  query(): string {
    const seriesQuery =
      this.profName +
      this.labels.reduce(function (agg: string, label: Label) {
        return agg + `${label.name}="${label.value}",`;
      }, '{');
    return seriesQuery + '}';
  }

  DiffSelection(): ProfileDiffSelection {
    return {
      options: {
        oneofKind: 'single',
        single: {
          time: Timestamp.fromDate(new Date(this.time)),
          query: this.query(),
        },
      },
      mode: ProfileDiffSelection_Mode.SINGLE_UNSPECIFIED,
    };
  }

  QueryRequest(): QueryRequest {
    return {
      options: {
        oneofKind: 'single',
        single: {
          time: Timestamp.fromDate(new Date(this.time)),
          query: this.query(),
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED,
      mode: QueryRequest_Mode.SINGLE_UNSPECIFIED,
    };
  }

  ProfileType(): ProfileType {
    return ProfileType.fromString(this.profName);
  }

  profileName(): string {
    return this.profName;
  }

  Describe(): JSX.Element {
    const profileName = this.profileName();
    return (
      <>
        <p>
          {profileName !== '' ? <a>{profileName} profile of </a> : ''}
          {'  '}
          {this.labels
            .filter(label => label.name !== '__name__')
            .map(label => (
              <button
                key={label.name}
                type="button"
                className="inline-block rounded-lg text-gray-700 bg-gray-200 dark:bg-gray-700 dark:text-gray-400 px-2 py-1 text-xs font-bold mr-3"
              >
                {`${label.name}="${label.value}"`}
              </button>
            ))}
        </p>
        <p>{formatDate(this.time, timeFormat)}</p>
      </>
    );
  }

  stringLabels(): string[] {
    return this.labels
      .filter((label: Label) => label.name !== '__name__')
      .map((label: Label) => `${label.name}=${label.value}`);
  }

  toString(): string {
    return `single profile of type ${this.profileName()} with labels ${this.stringLabels().join(
      ', '
    )} collected at ${formatDate(this.time, timeFormat)}`;
  }
}

export class ProfileDiffSource implements ProfileSource {
  a: ProfileSource;
  b: ProfileSource;

  constructor(a: ProfileSource, b: ProfileSource) {
    this.a = a;
    this.b = b;
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
  from: number;
  to: number;
  query: string;

  constructor(from: number, to: number, query: string) {
    this.from = from;
    this.to = to;
    this.query = query;
  }

  DiffSelection(): ProfileDiffSelection {
    return {
      options: {
        oneofKind: 'merge',
        merge: {
          start: Timestamp.fromDate(new Date(this.from)),
          end: Timestamp.fromDate(new Date(this.to)),
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
          start: Timestamp.fromDate(new Date(this.from)),
          end: Timestamp.fromDate(new Date(this.to)),
          query: this.query,
        },
      },
      reportType: QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED,
      mode: QueryRequest_Mode.MERGE,
    };
  }

  ProfileType(): ProfileType {
    return ProfileType.fromString(Query.parse(this.query).profileName());
  }

  Describe(): JSX.Element {
    return (
      <a>
        Merge of "{this.query}" from {formatDate(this.from, timeFormat)} to{' '}
        {formatDate(this.to, timeFormat)}
      </a>
    );
  }

  toString(): string {
    return `merged profiles of query "${this.query}" from ${formatDate(
      this.from,
      timeFormat
    )} to ${formatDate(this.to, timeFormat)}`;
  }
}
