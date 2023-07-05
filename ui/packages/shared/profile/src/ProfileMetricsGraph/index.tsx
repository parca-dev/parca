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

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {Duration, Label, QueryRangeResponse, QueryServiceClient, Timestamp} from '@parca/client';
import {DateTimeRange, useGrpcMetadata, useParcaContext} from '@parca/components';
import {Query} from '@parca/parser';
import {getStepDuration} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import MetricsGraph from '../MetricsGraph';
import useDelayedLoader from '../useDelayedLoader';

interface ProfileMetricsGraphProps {
  queryClient: QueryServiceClient;
  queryExpression: string;
  profile: ProfileSelection | null;
  from: number;
  to: number;
  setTimeRange: (range: DateTimeRange) => void;
  addLabelMatcher: (key: string, value: string) => void;
  onPointClick: (timestamp: number, labels: Label[], queryExpression: string) => void;
}

export interface IQueryRangeState {
  response: QueryRangeResponse | null;
  isLoading: boolean;
  error: RpcError | null;
}

export const useQueryRange = (
  client: QueryServiceClient,
  queryExpression: string,
  start: number,
  end: number
): IQueryRangeState => {
  const [state, setState] = useState<IQueryRangeState>({
    response: null,
    isLoading: true,
    error: null,
  });
  const metadata = useGrpcMetadata();

  useEffect(() => {
    void (async () => {
      setState({
        response: null,
        isLoading: true,
        error: null,
      });

      const stepDuration = getStepDuration(start, end);
      const call = client.queryRange(
        {
          query: queryExpression,
          start: Timestamp.fromDate(new Date(start)),
          end: Timestamp.fromDate(new Date(end)),
          step: Duration.create(stepDuration),
          limit: 0,
        },
        {meta: metadata}
      );

      call.response
        .then(response => setState({response, isLoading: false, error: null}))
        .catch(error => setState({response: null, isLoading: false, error}));
    })();
  }, [client, queryExpression, start, end, metadata]);

  return state;
};

const ProfileMetricsGraph = ({
  queryClient,
  queryExpression,
  profile,
  from,
  to,
  setTimeRange,
  addLabelMatcher,
  onPointClick,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {isLoading, response, error} = useQueryRange(queryClient, queryExpression, from, to);
  const isLoaderVisible = useDelayedLoader(isLoading);
  const {loader, onError, perf} = useParcaContext();

  useEffect(() => {
    if (error !== null) {
      onError?.(error, 'metricsGraph');
    }
  }, [error, onError]);

  useEffect(() => {
    if (response === null) {
      return;
    }

    perf?.markInteraction('Metrics graph render', response.series[0].samples.length);
  }, [perf, response]);

  if (isLoaderVisible) {
    return <>{loader}</>;
  }

  if (error !== null) {
    return (
      <div
        className="relative rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700"
        role="alert"
      >
        <strong className="font-bold">Error! </strong>
        <span className="block sm:inline">{error.message}</span>
      </div>
    );
  }

  const series = response?.series;
  if (series !== null && series !== undefined && series?.length > 0) {
    const handleSampleClick = (timestamp: number, _value: number, labels: Label[]): void => {
      onPointClick(timestamp, labels, queryExpression);
    };

    return (
      <div
        className="rounded border-gray-300 dark:border-gray-500 dark:bg-gray-700"
        style={{borderWidth: 1}}
      >
        <MetricsGraph
          data={series}
          from={from}
          to={to}
          profile={profile as MergedProfileSelection}
          setTimeRange={setTimeRange}
          onSampleClick={handleSampleClick}
          onLabelClick={addLabelMatcher}
          width={0}
          sampleUnit={Query.parse(queryExpression).profileType().sampleUnit}
        />
      </div>
    );
  }
  return (
    <div className="grid grid-cols-1">
      <div className="flex justify-center py-20">
        <p className="m-0">No data found. Try a different query.</p>
      </div>
    </div>
  );
};

export default ProfileMetricsGraph;
