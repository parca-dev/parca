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

import {useMemo} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {QueryRequest_ReportType, QueryResponse, QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';

import {ProfileSource} from './ProfileSource';
import useGrpcQuery from './useGrpcQuery';

export interface IQueryResult {
  response: QueryResponse | null;
  error: RpcError | null;
  isLoading: boolean;
  refetch?: () => void;
}

interface UseQueryOptions {
  skip?: boolean;
  nodeTrimThreshold?: number;
  groupBy?: string[];
  sourceBuildID?: string;
  sourceFilename?: string;
  sourceOnly?: boolean;
  invertCallStack?: boolean;
  sandwichByFunction?: string;
  protoFilters?: any[]; // Using any[] to match the Filter type from hook
  staleTime?: number;
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource,
  reportType: QueryRequest_ReportType,
  options?: UseQueryOptions
): IQueryResult => {
  const {skip = false} = options ?? {};
  const metadata = useGrpcMetadata();

  const protoFiltersKey = useMemo(() => {
    return JSON.stringify(options?.protoFilters ?? []);
  }, [options?.protoFilters]);

  const {data, isLoading, error, refetch} = useGrpcQuery<QueryResponse | undefined>({
    key: [
      'query',
      profileSource.toKey(),
      reportType,
      options?.nodeTrimThreshold,
      options?.groupBy,
      options?.sourceBuildID,
      options?.sourceOnly,
      options?.sourceOnly === true ? '' : options?.sourceFilename,
      options?.invertCallStack ?? false,
      options?.sandwichByFunction ?? '',
      protoFiltersKey,
    ],
    queryFn: async signal => {
      const req = profileSource.QueryRequest();
      req.reportType = reportType;
      req.nodeTrimThreshold = options?.nodeTrimThreshold;
      req.groupBy = {
        fields: options?.groupBy ?? [],
      };
      if (options?.sourceBuildID !== undefined && options?.sourceFilename !== undefined) {
        req.sourceReference = {
          buildId: options?.sourceBuildID ?? '',
          filename: options?.sourceFilename ?? '',
          sourceOnly: options?.sourceOnly ?? false,
        };
      }
      req.invertCallStack = options?.invertCallStack ?? false;

      if (options?.protoFilters != null && options?.protoFilters?.length > 0) {
        req.filter = options.protoFilters;
      }
      // Handle sandwich view filter separately
      if (options?.sandwichByFunction !== undefined) {
        req.sandwichByFunction = options.sandwichByFunction;
      }

      try {
        const {response} = await client.query(req, {meta: metadata, abort: signal});
        return response;
      } catch (e) {
        if (options?.sourceOnly === true) {
          // ignore
          return {} as unknown as QueryResponse;
        }
        throw e;
      }
    },
    options: {
      retry: false,
      enabled: !skip,
      staleTime: options?.staleTime ?? 1000 * 60 * 5, // 5 minutes
    },
  });

  return {
    isLoading,
    error: error as RpcError | null,
    response: data ?? null,
    refetch: () => {
      void refetch();
    },
  };
};
