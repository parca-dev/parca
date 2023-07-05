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
