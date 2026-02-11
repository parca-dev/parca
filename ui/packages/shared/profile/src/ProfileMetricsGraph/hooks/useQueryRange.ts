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

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {Duration, QueryRangeResponse, QueryServiceClient, Timestamp} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {getStepDuration, getStepDurationInMilliseconds} from '@parca/utilities';

import useGrpcQuery from '../../useGrpcQuery';

interface QueryRangeResult {
  response: QueryRangeResponse;
  stepDurationMs: number;
}

interface IQueryRangeState {
  response: QueryRangeResponse | null;
  isLoading: boolean;
  error: RpcError | null;
  stepDurationMs: number;
}

export const getStepCountFromScreenWidth = (pixelsPerPoint: number): number => {
  let width =
    // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
    window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;

  // subtract the padding around the graph
  width = width - (20 + 24 + 68) * 2;
  return Math.round(width / pixelsPerPoint);
};

export const useQueryRange = (
  client: QueryServiceClient,
  queryExpression: string,
  start: number,
  end: number,
  sumBy: string[],
  stepCount: number,
  skip = false
): IQueryRangeState => {
  const metadata = useGrpcMetadata();

  const {data, isLoading, error} = useGrpcQuery<QueryRangeResult | undefined>({
    key: ['query-range', queryExpression, start, end, (sumBy ?? []).join(','), stepCount, metadata],
    queryFn: async signal => {
      const stepDuration = getStepDuration(start, end, stepCount);
      const stepDurationMs = getStepDurationInMilliseconds(stepDuration);
      const {response} = await client.queryRange(
        {
          query: queryExpression,
          start: Timestamp.fromDate(new Date(start)),
          end: Timestamp.fromDate(new Date(end)),
          step: Duration.create(stepDuration),
          limit: 0,
          sumBy,
        },
        {meta: metadata, abort: signal}
      );
      return {response, stepDurationMs};
    },
    options: {
      retry: false,
      enabled: !skip,
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  });

  return {
    isLoading,
    error: error as RpcError | null,
    response: data?.response ?? null,
    stepDurationMs: data?.stepDurationMs ?? 0,
  };
};
