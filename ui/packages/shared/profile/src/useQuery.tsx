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
