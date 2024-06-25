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

import type {RpcMetadata} from '@protobuf-ts/runtime-rpc';

import {
  ProfileType,
  QueryRequest,
  QueryRequest_ReportType,
  QueryServiceClient,
} from '@parca/client';
import {DateTimeRange} from '@parca/components';
import {type NavigateFunction} from '@parca/utilities';

import {ProfileSelectionFromParams, SuffixParams} from '.';
import {constructProfileName} from './ProfileTypeSelector';

export const hexifyAddress = (address?: bigint): string => {
  if (address == null) {
    return '';
  }
  return `0x${address.toString(16)}`;
};

export const downloadPprof = async (
  request: QueryRequest,
  queryClient: QueryServiceClient,
  metadata: RpcMetadata
): Promise<Blob> => {
  const req = {
    ...request,
    reportType: QueryRequest_ReportType.PPROF,
  };

  const {response} = await queryClient.query(req, {meta: metadata});
  if (response.report.oneofKind !== 'pprof') {
    throw new Error(
      `Expected pprof report, got: ${
        response.report.oneofKind !== undefined ? response.report.oneofKind : 'undefined'
      }`
    );
  }
  const blob = new Blob([response.report.pprof], {type: 'application/octet-stream'});
  return blob;
};

export const truncateString = (str: string, num: number): string => {
  if (str.length <= num) {
    return str;
  }

  return str.slice(0, num) + '...';
};

export const truncateStringReverse = (str: string, num: number): string => {
  if (str.length <= num) {
    return str;
  }

  return '...' + str.slice(str.length - num);
};

export const compareProfile = (
  navigateTo: NavigateFunction,
  defaultDashboardItems: string[] = ['icicle'],
  profileTypes: ProfileType[]
): void => {
  const timeSelection = 'relative:minute|15';
  const dashboardItems = ['icicle'];

  let profileType;

  if (profileTypes == null || profileTypes.length === 0) {
    profileType = undefined;
  }

  if (profileTypes !== null && profileTypes.length > 0) {
    if (profileType == null) {
      profileType = profileTypes.find(
        type => type.name === 'otel_profiling_agent_on_cpu' && type.delta
      );
    }

    if (profileType == null) {
      profileType = profileTypes.find(type => type.name === 'parca_agent_cpu' && type.delta);
    }

    if (profileType == null) {
      profileType = profileTypes.find(type => type.name === 'process_cpu' && type.delta);
    }

    if (profileType == null) {
      profileType = profileTypes[0];
    }
  }

  const selection =
    profileType !== undefined
      ? constructProfileName(profileType)
      : 'process_cpu:samples:count:cpu:nanoseconds:delta{}';
  const expression =
    profileType !== undefined
      ? constructProfileName(profileType)
      : 'process_cpu:samples:count:cpu:nanoseconds:delta{}';

  const range = DateTimeRange.fromRangeKey(timeSelection, undefined, undefined);
  const from = range.getFromMs();
  const to = range.getToMs();

  const profileA = ProfileSelectionFromParams(from.toString(), to.toString(), selection, '');
  const queryA = {
    expression,
    from,
    to,
    timeSelection: timeSelection as string,
  };

  let compareQuery = {
    compare_a: 'true',
    expression_a: encodeURIComponent(queryA.expression),
    from_a: queryA.from.toString(),
    to_a: queryA.to.toString(),
    time_selection_a: queryA.timeSelection,

    compare_b: 'true',
    expression_b: encodeURIComponent(queryA.expression),
    from_b: queryA.from.toString(),
    to_b: queryA.to.toString(),
    time_selection_b: queryA.timeSelection,
  };

  if (profileA != null) {
    compareQuery = {
      ...SuffixParams(profileA.HistoryParams(), '_a'),
      ...compareQuery,
    };
  }

  void navigateTo('/', {
    ...compareQuery,
    search_string: '',
    dashboard_items: dashboardItems ?? defaultDashboardItems,
  });
};
