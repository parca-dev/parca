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

import {useState, useEffect} from 'react';
import MetricsGraph from '../MetricsGraph';
import {ProfileSelection, SingleProfileSelection} from '..';
import {QueryServiceClient, QueryRangeResponse, Label, Timestamp} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {DateTimeRange, useGrpcMetadata, useParcaContext} from '@parca/components';
import {Query} from '@parca/parser';
import useDelayedLoader from '../useDelayedLoader';

interface ProfileMetricsGraphProps {
  queryClient: QueryServiceClient;
  queryExpression: string;
  profile: ProfileSelection | null;
  from: number;
  to: number;
  select: (source: ProfileSelection) => void;
  setTimeRange: (range: DateTimeRange) => void;
  addLabelMatcher: (key: string, value: string) => void;
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

      const call = client.queryRange(
        {
          query: queryExpression,
          start: Timestamp.fromDate(new Date(start)),
          end: Timestamp.fromDate(new Date(end)),
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
  select,
  setTimeRange,
  addLabelMatcher,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {isLoading, response, error} = useQueryRange(queryClient, queryExpression, from, to);
  const isLoaderVisible = useDelayedLoader(isLoading);
  const {loader, onError} = useParcaContext();

  if (isLoaderVisible) {
    return <>{loader}</>;
  }

  if (error !== null) {
    onError?.metricsGraphError(error);

    return (
      <div
        className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded relative"
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
      select(
        new SingleProfileSelection(Query.parse(queryExpression).profileName(), labels, timestamp)
      );
    };

    return (
      <div
        className="dark:bg-gray-700 rounded border-gray-300 dark:border-gray-500"
        style={{borderWidth: 1}}
      >
        <MetricsGraph
          data={series}
          from={from}
          to={to}
          profile={profile as SingleProfileSelection}
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
      <div className="py-20 flex justify-center">
        <p className="m-0">No data found. Try a different query.</p>
      </div>
    </div>
  );
};

export default ProfileMetricsGraph;
