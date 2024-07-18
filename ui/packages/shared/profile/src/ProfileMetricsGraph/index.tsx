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

import {useEffect, useMemo} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';
import {AnimatePresence, motion} from 'framer-motion';

import {Duration, Label, QueryRangeResponse, QueryServiceClient, Timestamp} from '@parca/client';
import {
  DateTimeRange,
  MetricsGraphSkeleton,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {Query} from '@parca/parser';
import {capitalizeOnlyFirstLetter, getStepDuration} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import {useLabelNames} from '../MatchersInput';
import MetricsGraph from '../MetricsGraph';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import useGrpcQuery from '../useGrpcQuery';
import {DEFAULT_EMPTY_SUM_BY, useSumBy} from '../useSumBy';

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
  dirtyQueryExpression: string;
  profile: ProfileSelection | null;
  from: number;
  to: number;
  sumBy: string[];
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

const getStepCountFromScreenWidth = (pixelsPerPoint: number): number => {
  let width =
    // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
    window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;

  // subtract the padding around the graph
  width = width - (20 + 24 + 68) * 2;
  return Math.round(width / pixelsPerPoint);
};

export const useQueryRange = (
  client: QueryServiceClient,
  queryExpression: string,
  start: number,
  end: number,
  sumBy: string[] = DEFAULT_EMPTY_SUM_BY,
  skip = false
): IQueryRangeState => {
  const metadata = useGrpcMetadata();
  const {navigateTo} = useParcaContext();
  const [stepCountStr, setStepCount] = useURLState({param: 'step_count', navigateTo});
  const defaultStepCount = useMemo(() => {
    return getStepCountFromScreenWidth(10);
  }, []);

  const stepCount = useMemo(() => {
    if (stepCountStr != null) {
      return parseInt(stepCountStr as string, 10);
    }

    return defaultStepCount;
  }, [stepCountStr, defaultStepCount]);

  useEffect(() => {
    if (stepCountStr == null) {
      setStepCount(defaultStepCount.toString());
    }
  }, [stepCountStr, defaultStepCount, setStepCount]);

  const {data, isLoading, error} = useGrpcQuery<QueryRangeResponse | undefined>({
    key: ['query-range', queryExpression, start, end, sumBy.join(','), stepCount, metadata],
    queryFn: async () => {
      const stepDuration = getStepDuration(start, end, stepCount);
      const {response} = await client.queryRange(
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
      return response;
    },
    options: {
      retry: false,
      enabled: !skip && sumBy !== DEFAULT_EMPTY_SUM_BY,
      staleTime: 1000 * 60 * 5, // 5 minutes
    },
  });

  return {isLoading, error: error as RpcError | null, response: data ?? null};
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
  sumBy,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {
    isLoading: metricsGraphLoading,
    response,
    error,
  } = useQueryRange(queryClient, queryExpression, from, to, sumBy);
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

  const loading = metricsGraphLoading;

  if (!metricsGraphLoading && error !== null) {
    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
  }

  let sampleUnit = '';

  if (dataAvailable) {
    if (series.every((val, i, arr) => val?.sampleType?.unit === arr[0]?.sampleType?.unit)) {
      sampleUnit = series[0]?.sampleType?.unit ?? '';
    }
    if (sampleUnit === '') {
      sampleUnit = Query.parse(queryExpression).profileType().sampleUnit;
    }
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
        {loading ? (
          <MetricsGraphSkeleton heightStyle={heightStyle} isDarkMode={isDarkMode} />
        ) : dataAvailable ? (
          <MetricsGraph
            data={series}
            from={from}
            to={to}
            profile={profile as MergedProfileSelection}
            setTimeRange={setTimeRange}
            onSampleClick={(
              timestamp: number,
              _value: number,
              labels: Label[],
              duration: number
            ): void => {
              onPointClick(timestamp, labels, queryExpression, duration);
            }}
            addLabelMatcher={addLabelMatcher}
            sampleUnit={sampleUnit}
            height={height}
            width={width}
            margin={margin}
          />
        ) : (
          <ProfileMetricsEmptyState message="No data found. Try a different query." />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileMetricsGraph;
