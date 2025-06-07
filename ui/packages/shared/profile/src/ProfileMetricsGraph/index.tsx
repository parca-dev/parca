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

import {useEffect} from 'react';

import {AnimatePresence, motion} from 'framer-motion';

import {Label, QueryServiceClient} from '@parca/client';
import {DateTimeRange, MetricsGraphSkeleton, useParcaContext} from '@parca/components';
import {Query} from '@parca/parser';
import {capitalizeOnlyFirstLetter} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import MetricsGraph from '../MetricsGraph';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import {useQueryRange} from './hooks/useQueryRange';

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
  sumByLoading: boolean;
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
  sumByLoading,
}: ProfileMetricsGraphProps): JSX.Element => {
  const {
    isLoading: metricsGraphLoading,
    response,
    error,
  } = useQueryRange(queryClient, queryExpression, from, to, sumBy, sumByLoading);
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
  let sampleType = '';

  if (dataAvailable) {
    if (series.every((val, i, arr) => val?.sampleType?.unit === arr[0]?.sampleType?.unit)) {
      sampleUnit = series[0]?.sampleType?.unit ?? '';
      sampleType = series[0]?.sampleType?.type ?? '';
    }
    if (sampleUnit === '') {
      const profileType = Query.parse(queryExpression).profileType();
      sampleUnit = profileType.sampleUnit;
      sampleType = profileType.sampleType;
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
            sampleType={sampleType}
            height={height}
            width={width}
            margin={margin}
            sumBy={sumBy}
          />
        ) : (
          <ProfileMetricsEmptyState message="No data found. Try a different query." />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileMetricsGraph;
