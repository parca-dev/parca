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
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import useDelayedLoader from '../useDelayedLoader';

interface ProfileMetricsEmptyStateProps {
  message: string;
}

export const ProfileMetricsEmptyState = ({message}: ProfileMetricsEmptyStateProps): JSX.Element => {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center">
      <p>{message}</p>
    </div>
  );
};

interface ProfileMetricsGraphProps {
  queryClient: QueryServiceClient;
  queryExpression: string;
  profile: ProfileSelection | null;
  from: number;
  to: number;
  setTimeRange: (range: DateTimeRange) => void;
  addLabelMatcher: (key: string, value: string) => void;
  onPointClick: (timestamp: number, labels: Label[], queryExpression: string) => void;
  comparing?: boolean;
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
  const [isLoading, setLoading] = useState<boolean>(true);
  const metadata = useGrpcMetadata();

  useEffect(() => {
    void (async () => {
      setLoading(true);

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
        .then(response => {
          setState({response, isLoading: false, error: null});
          setLoading(false);
          return null;
        })
        .catch(error => {
          setState({response: null, isLoading: false, error});
          setLoading(false);
        });
    })();
  }, [client, queryExpression, start, end, metadata]);

  return {...state, isLoading};
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
  comparing = false,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {isLoading, response, error} = useQueryRange(queryClient, queryExpression, from, to);
  const isLoaderVisible = useDelayedLoader(isLoading);
  const {loader, onError, perf} = useParcaContext();
  const {width, height, margin} = useMetricsGraphDimensions(comparing);

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

  const series = response?.series;
  const dataAvailable = series !== null && series !== undefined && series?.length > 0;

  if (isLoaderVisible || (isLoading && !dataAvailable)) {
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

  if (dataAvailable) {
    const handleSampleClick = (timestamp: number, _value: number, labels: Label[]): void => {
      onPointClick(timestamp, labels, queryExpression);
    };

    return (
      <div className="h-full w-full">
        <MetricsGraph
          data={series}
          from={from}
          to={to}
          profile={profile as MergedProfileSelection}
          setTimeRange={setTimeRange}
          onSampleClick={handleSampleClick}
          onLabelClick={addLabelMatcher}
          sampleUnit={Query.parse(queryExpression).profileType().sampleUnit}
          height={height}
          width={width}
          margin={margin}
        />
      </div>
    );
  }

  return <ProfileMetricsEmptyState message="No data found. Try a different query." />;
};

export default ProfileMetricsGraph;
