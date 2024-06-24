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

import {QueryRequest, QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {parseParams, type NavigateFunction} from '@parca/utilities';

import {ProfileSelectionFromParams, SuffixParams, getExpressionAsAString} from '.';

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
  defaultDashboardItems: string[] = ['icicle']
): void => {
  const queryParams = parseParams(window.location.search);

  /* eslint-disable @typescript-eslint/naming-convention */
  const {
    from_a,
    to_a,
    merge_from_a,
    merge_to_a,
    time_selection_a,
    filter_by_function,
    dashboard_items,
  } = queryParams;

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const selection_a = getExpressionAsAString(queryParams.selection_a as string | []);

  // eslint-disable-next-line @typescript-eslint/naming-convention
  const expression_a = getExpressionAsAString(queryParams.expression_a as string | []);

  if (expression_a === undefined || selection_a === undefined) {
    return;
  }

  const mergeFrom = merge_from_a ?? undefined;
  const mergeTo = merge_to_a ?? undefined;
  const profileA = ProfileSelectionFromParams(
    mergeFrom as string,
    mergeTo as string,
    selection_a,
    filter_by_function as string
  );
  const queryA = {
    expression: expression_a,
    from: parseInt(from_a as string),
    to: parseInt(to_a as string),
    timeSelection: time_selection_a as string,
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
    dashboard_items: dashboard_items ?? defaultDashboardItems,
  });
};
