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
    key: ['query', profileSource, reportType, options?.nodeTrimThreshold],
    queryFn: async () => {
      const req = profileSource.QueryRequest();
      req.reportType = reportType;
      req.nodeTrimThreshold = options?.nodeTrimThreshold;

      const {response} = await client.query(req, {meta: metadata});
      return response;
    },
    options: {
      enabled: !skip,
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  });

  return {isLoading, error: error as RpcError | null, response: data ?? null};
};
