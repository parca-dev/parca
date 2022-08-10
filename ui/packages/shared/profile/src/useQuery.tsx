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

import {useEffect, useState} from 'react';

import {QueryServiceClient, QueryResponse, QueryRequest_ReportType} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {useGrpcMetadata} from '@parca/components';

import {ProfileSource} from './ProfileSource';

export interface IQueryResult {
  response: QueryResponse | null;
  error: RpcError | null;
  isLoading: boolean;
}

interface UseQueryOptions {
  skip?: boolean;
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource,
  reportType: QueryRequest_ReportType,
  options?: UseQueryOptions
): IQueryResult => {
  const {skip = false} = options || {};
  const [result, setResult] = useState<IQueryResult>({
    response: null,
    error: null,
    isLoading: false,
  });
  const metadata = useGrpcMetadata();

  useEffect(() => {
    if (skip) {
      return;
    }
    setResult({
      response: null,
      error: null,
      isLoading: true,
    });
    const req = profileSource.QueryRequest();
    req.reportType = reportType;

    const call = client.query(req, {meta: metadata});

    call.response
      .then(response => setResult({response: response, error: null, isLoading: false}))
      .catch(error => setResult({error: error, response: null, isLoading: false}));
  }, [client, profileSource, metadata, reportType]);

  return result;
};
