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

import {QueryRequest_ReportType, QueryResponse, QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';

import {ProfileSource} from './ProfileSource';
import useGrpcQuery from './useGrpcQuery';

export interface IQueryResult {
  response: QueryResponse | null;
  error: RpcError | null;
  isLoading: boolean;
}

interface UseQueryOptions {
  skip?: boolean;
  nodeTrimThreshold?: number;
  groupBy?: string[];
  sourceBuildID?: string;
  sourceFilename?: string;
  sourceOnly?: boolean;
  invertCallStack?: boolean;
  binaryFrameFilter?: string[];
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource,
  reportType: QueryRequest_ReportType,
  options?: UseQueryOptions
): IQueryResult => {
  const {skip = false} = options ?? {};
  const metadata = useGrpcMetadata();
  const {data, isLoading, error} = useGrpcQuery<QueryResponse | undefined>({
    key: [
      'query',
      profileSource,
      reportType,
      options?.nodeTrimThreshold,
      options?.groupBy,
      options?.sourceBuildID,
      options?.sourceOnly,
      options?.sourceOnly === true ? '' : options?.sourceFilename,
      options?.invertCallStack ?? false,
      options?.binaryFrameFilter ?? '',
    ],
    queryFn: async () => {
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
      const functionToFilter = req.filterQuery;
      if (functionToFilter !== undefined) {
        req.filter = [
          ...req.filter,
          {
            filter: {
              oneofKind: 'stackFilter',
              stackFilter: {
                filter: {
                  oneofKind: 'functionNameStackFilter',
                  functionNameStackFilter: {
                    functionToFilter,
                  },
                },
              },
            },
          },
        ];
      }

      if (options?.binaryFrameFilter !== undefined && options?.binaryFrameFilter.length > 0) {
        req.filter = [
          ...req.filter,
          {
            filter: {
              oneofKind: 'frameFilter',
              frameFilter: {
                filter: {
                  oneofKind: 'binaryFrameFilter',
                  binaryFrameFilter: {
                    includeBinaries: options?.binaryFrameFilter ?? [],
                  },
                },
              },
            },
          },
        ];
      }

      try {
        const {response} = await client.query(req, {meta: metadata});
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
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  });

  return {isLoading, error: error as RpcError | null, response: data ?? null};
};
