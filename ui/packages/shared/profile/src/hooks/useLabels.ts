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

import {LabelsRequest, LabelsResponse, QueryServiceClient, ValuesRequest} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {millisToProtoTimestamp, sanitizeLabelValue} from '@parca/utilities';

import useGrpcQuery from '../useGrpcQuery';

export interface ILabelValuesResult {
  response: string[];
  error?: Error;
}

interface UseLabelValues {
  result: ILabelValuesResult;
  loading: boolean;
  refetch: () => Promise<void>;
}

export interface ILabelNamesResult {
  response?: LabelsResponse;
  error?: Error;
}

interface UseLabelNames {
  result: ILabelNamesResult;
  loading: boolean;
  refetch: () => Promise<void>;
}

export const useLabelNames = (
  client: QueryServiceClient,
  profileType: string,
  start?: number,
  end?: number,
  match?: string[]
): UseLabelNames => {
  const metadata = useGrpcMetadata();
  const enabled = profileType !== undefined && profileType !== '';

  const {data, isLoading, error, refetch} = useGrpcQuery<LabelsResponse>({
    key: ['labelNames', profileType, match?.join(','), start, end],
    queryFn: async signal => {
      const request: LabelsRequest = {match: match !== undefined ? match : []};
      if (start !== undefined && end !== undefined) {
        request.start = millisToProtoTimestamp(start);
        request.end = millisToProtoTimestamp(end);
      }
      if (profileType !== undefined) {
        request.profileType = profileType;
      }
      const {response} = await client.labels(request, {meta: metadata, abort: signal});
      return response;
    },
    options: {
      enabled,
      keepPreviousData: false,
    },
  });

  return {
    result: {response: data, error: error as Error},
    loading: enabled && isLoading,
    refetch: async () => {
      await refetch();
    },
  };
};

export const useLabelValues = (
  client: QueryServiceClient,
  labelName: string,
  profileType: string,
  start?: number,
  end?: number
): UseLabelValues => {
  const metadata = useGrpcMetadata();

  const {data, isLoading, error, refetch} = useGrpcQuery<string[]>({
    key: ['labelValues', labelName, profileType, start, end],
    queryFn: async signal => {
      const request: ValuesRequest = {labelName, match: [], profileType};
      if (start !== undefined && end !== undefined) {
        request.start = millisToProtoTimestamp(start);
        request.end = millisToProtoTimestamp(end);
      }
      const {response} = await client.values(request, {meta: metadata, abort: signal});
      return sanitizeLabelValue(response.labelValues);
    },
    options: {
      enabled:
        profileType !== undefined &&
        profileType !== '' &&
        labelName !== undefined &&
        labelName !== '',
      keepPreviousData: false,
    },
  });

  return {
    result: {response: data ?? [], error: error as Error},
    loading: isLoading,
    refetch: async () => {
      await refetch();
    },
  };
};
