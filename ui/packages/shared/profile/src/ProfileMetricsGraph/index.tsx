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
import {AnimatePresence, motion} from 'framer-motion';

import {Duration, Label, QueryRangeResponse, QueryServiceClient, Timestamp} from '@parca/client';
import {
  DateTimeRange,
  MetricsGraphSkeleton,
  useGrpcMetadata,
  useParcaContext,
} from '@parca/components';
import {Query} from '@parca/parser';
import {capitalizeOnlyFirstLetter, getStepDuration} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import {useLabelNames} from '../MatchersInput';
import MetricsGraph from '../MetricsGraph';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import useDelayedLoader from '../useDelayedLoader';
import {Toolbar} from './Toolbar';

interface ProfileMetricsEmptyStateProps {
  message: string;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return (
    <div
      className="relative rounded border border-red-400 bg-red-100 px-4 py-3 text-red-700"
      role="alert"
    >
      <span className="block sm:inline">{errorMessage}</span>
    </div>
  );
};

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
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void;
  onPointClick: (
    timestamp: number,
    labels: Label[],
    queryExpression: string,
    duration: number
  ) => void;
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
  end: number,
  sumBy: string[] = []
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
          sumBy,
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
  }, [client, queryExpression, start, end, metadata, sumBy]);

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
  const {loading: labelNamesLoading, result: labelNamesResult} = useLabelNames(queryClient);
  const [sumBy, setSumBy] = useState<string[]>([]);
  const {isLoading, response, error} = useQueryRange(queryClient, queryExpression, from, to, sumBy);
  const isLoaderVisible = useDelayedLoader(isLoading);
  const {onError, perf, authenticationErrorMessage, isDarkMode} = useParcaContext();
  const {width, height, margin, heightStyle} = useMetricsGraphDimensions(comparing);

  useEffect(() => {
    if (error !== null) {
      onError?.(error);
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

  const metricsGraphLoading =
    isLoaderVisible || (isLoading && !dataAvailable && !labelNamesLoading);

  if (metricsGraphLoading) {
    return <MetricsGraphSkeleton heightStyle={heightStyle} isDarkMode={isDarkMode} />;
  }

  if (!metricsGraphLoading && error !== null) {
    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
  }

  if (!labelNamesLoading && labelNamesResult?.error != null) {
    return (
      <ErrorContent errorMessage={capitalizeOnlyFirstLetter(labelNamesResult.error.message)} />
    );
  }

  if (dataAvailable) {
    const handleSampleClick = (
      timestamp: number,
      _value: number,
      labels: Label[],
      duration: number
    ): void => {
      onPointClick(timestamp, labels, queryExpression, duration);
    };

    let sampleUnit = '';
    if (series.every((val, i, arr) => val?.sampleType?.unit === arr[0]?.sampleType?.unit)) {
      sampleUnit = series[0]?.sampleType?.unit ?? '';
    }
    if (sampleUnit === '') {
      sampleUnit = Query.parse(queryExpression).profileType().sampleUnit;
    }

    return (
      <AnimatePresence>
        <motion.div
          className="h-full w-full relative"
          key="metrics-graph-loaded"
          initial={{display: 'none', opacity: 0}}
          animate={{display: 'block', opacity: 1}}
          transition={{duration: 0.5}}
        >
          <Toolbar
            sumBy={sumBy}
            setSumBy={setSumBy}
            labels={labelNamesResult.response?.labelNames ?? []}
          />
          <MetricsGraph
            data={series}
            from={from}
            to={to}
            profile={profile as MergedProfileSelection}
            setTimeRange={setTimeRange}
            onSampleClick={handleSampleClick}
            addLabelMatcher={addLabelMatcher}
            sampleUnit={sampleUnit}
            height={height}
            width={width}
            margin={margin}
          />
        </motion.div>
      </AnimatePresence>
    );
  }

  return <ProfileMetricsEmptyState message="No data found. Try a different query." />;
};

export default ProfileMetricsGraph;
