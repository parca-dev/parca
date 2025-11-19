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

import cx from 'classnames';

import {Label, QueryServiceClient} from '@parca/client';
import {DateTimeRange, useURLStateBatch} from '@parca/components';
import {Query} from '@parca/parser';

import {ProfileSelection} from '..';
import UtilizationMetricsGraph from '../MetricsGraph/UtilizationMetrics';
import AreaChart from '../MetricsGraph/UtilizationMetrics/Throughput';
import ProfileMetricsGraph, {ProfileMetricsEmptyState} from '../ProfileMetricsGraph';
import {useResetStateOnSeriesChange} from '../ProfileView/hooks/useResetStateOnSeriesChange';
import {QuerySelection, type UtilizationMetrics as UtilizationMetricsType} from './index';

interface MetricsGraphSectionProps {
  showMetricsGraph: boolean;
  setDisplayHideMetricsGraphButton?: (show: boolean) => void;
  heightStyle: string;
  querySelection: QuerySelection;
  profileSelection: ProfileSelection | null;
  comparing: boolean;
  sumBy: string[] | null;
  defaultSumByLoading: boolean;
  queryClient: QueryServiceClient;
  queryExpressionString: string;
  setTimeRangeSelection: (range: DateTimeRange) => void;
  selectQuery: (query: QuerySelection) => void;
  setProfileSelection: (mergeFrom: bigint, mergeTo: bigint, query: Query) => void;
  query: Query;
  setNewQueryExpression: (queryExpression: string, commit?: boolean) => void;
  setQueryExpression: (updateTs?: boolean) => void;
  utilizationMetrics?: Array<{
    name: string;
    humanReadableName: string;
    data: UtilizationMetricsType[];
  }>;
  utilizationMetricsLoading?: boolean;
  onUtilizationSeriesSelect?: (name: string, seriesIndex: number) => void;
}

export function MetricsGraphSection({
  showMetricsGraph,
  setDisplayHideMetricsGraphButton,
  heightStyle,
  querySelection,
  profileSelection,
  comparing,
  sumBy,
  defaultSumByLoading,
  queryClient,
  queryExpressionString,
  setTimeRangeSelection,
  selectQuery,
  setProfileSelection,
  query,
  setNewQueryExpression,
  utilizationMetrics,
  utilizationMetricsLoading,
  onUtilizationSeriesSelect,
}: MetricsGraphSectionProps): JSX.Element {
  const resetStateOnSeriesChange = useResetStateOnSeriesChange();
  const batchUpdates = useURLStateBatch();
  const handleTimeRangeChange = (range: DateTimeRange): void => {
    const from = range.getFromMs();
    const to = range.getToMs();
    let mergedProfileParams = {};
    if (query.profileType().delta) {
      mergedProfileParams = {mergeFrom: from * 1_000_000, mergeTo: to * 1_000_000};
    }
    setTimeRangeSelection(range);
    selectQuery({
      expression: queryExpressionString,
      from,
      to,
      timeSelection: range.getRangeKey(),
      ...mergedProfileParams,
    });
  };

  const addLabelMatcher = (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ): void => {
    // When a user clicks on a label on the metrics graph tooltip,
    // replace single `\` in the `value` string with doubles `\\` if available.
    const replaceBackslash = (value: string): string => {
      return value.includes('\\') ? value.replaceAll('\\', '\\\\') : value;
    };

    let newQuery: Query;
    let hasChanged: boolean;

    if (Array.isArray(labels)) {
      const newLabels = labels.map(({key, value}) => {
        const newValue = replaceBackslash(value);
        return {key, value: newValue};
      });
      const [query, changed] = Query.parse(queryExpressionString).setMultipleMatchers(newLabels);
      hasChanged = changed;
      newQuery = query;
    } else {
      const {key, value} = labels;
      const newValue = replaceBackslash(value);
      const [query, changed] = Query.parse(queryExpressionString).setMatcher(key, newValue);
      hasChanged = changed;
      newQuery = query;
    }

    if (hasChanged) {
      // TODO: Change this to store the query object
      // Pass commit: true to immediately apply the filter when clicking on metrics graph labels
      setNewQueryExpression(newQuery.toString(), true);
    }
  };

  const handlePointClick = (
    timestamp: bigint,
    labels: Label[],
    queryExpression: string,
    duration: number
  ): void => {
    let query = Query.parse(queryExpression);

    labels.forEach(l => {
      const [newQuery, updated] = query.setMatcher(l.name, l.value);
      if (updated) {
        query = newQuery;
      }
    });

    const mergeFrom = timestamp;
    const mergeTo = query.profileType().delta ? mergeFrom + BigInt(duration) : mergeFrom;
    batchUpdates(() => {
      resetStateOnSeriesChange(); // reset some state when a new series is selected
      setProfileSelection(mergeFrom, mergeTo, query);
    });
  };

  const UtilizationGraphToShow = ({
    utilizationMetrics,
  }: {
    utilizationMetrics: Array<{
      name: string;
      humanReadableName: string;
      data: UtilizationMetricsType[];
    }>;
  }): JSX.Element => {
    const throughputMetrics = utilizationMetrics.filter(
      metric =>
        metric.name === 'gpu_pcie_throughput_transmit_bytes' ||
        metric.name === 'gpu_pcie_throughput_receive_bytes'
    );
    const transmitData =
      throughputMetrics.find(metric => metric.name === 'gpu_pcie_throughput_transmit_bytes')
        ?.data ?? [];
    const receiveData =
      throughputMetrics.find(metric => metric.name === 'gpu_pcie_throughput_receive_bytes')?.data ??
      [];

    if (utilizationMetrics.length === 0) {
      return <></>;
    }

    return (
      <div>
        {utilizationMetrics.map(({name, humanReadableName, data}) => {
          if (
            name !== 'gpu_pcie_throughput_transmit_bytes' &&
            name !== 'gpu_pcie_throughput_receive_bytes'
          ) {
            return (
              <UtilizationMetricsGraph
                key={name}
                data={data}
                setTimeRange={handleTimeRangeChange}
                utilizationMetricsLoading={utilizationMetricsLoading}
                humanReadableName={humanReadableName}
                from={querySelection.from}
                to={querySelection.to}
                yAxisUnit="percentage"
                addLabelMatcher={addLabelMatcher}
                onSeriesClick={seriesIndex => {
                  // For generic UtilizationMetrics, just pass the series index
                  if (onUtilizationSeriesSelect != null) {
                    onUtilizationSeriesSelect(name, seriesIndex);
                  }
                }}
              />
            );
          }
          return null;
        })}
        {throughputMetrics.length > 0 && (
          <AreaChart
            transmitData={transmitData}
            receiveData={receiveData}
            addLabelMatcher={addLabelMatcher}
            setTimeRange={handleTimeRangeChange}
            name={throughputMetrics[0].name}
            humanReadableName={throughputMetrics[0].humanReadableName}
            from={querySelection.from}
            to={querySelection.to}
            utilizationMetricsLoading={utilizationMetricsLoading}
            selectedSeries={undefined}
            onSeriesClick={(name, seriesIndex) => {
              // For throughput metrics, just pass the series index
              if (onUtilizationSeriesSelect != null) {
                let name = 'gpu_pcie_throughput_transmit_bytes';
                if (seriesIndex > transmitData.length - 1) {
                  name = 'gpu_pcie_throughput_receive_bytes';
                  seriesIndex -= transmitData.length;
                }
                onUtilizationSeriesSelect(name, seriesIndex);
              }
            }}
          />
        )}
      </div>
    );
  };

  return (
    <div className={cx('relative', {'py-4': !showMetricsGraph})}>
      {setDisplayHideMetricsGraphButton != null ? (
        <button
          onClick={() => setDisplayHideMetricsGraphButton(!showMetricsGraph)}
          className={cx(
            'hidden px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:bg-gray-900 z-[5]',
            showMetricsGraph && 'absolute right-0 bottom-3 !flex',
            !showMetricsGraph && 'relative !flex ml-auto'
          )}
        >
          {showMetricsGraph ? 'Hide' : 'Show'} Metrics Graph
        </button>
      ) : null}
      {showMetricsGraph && (
        <>
          <div style={{height: heightStyle}}>
            {(querySelection.expression !== '' || defaultSumByLoading) &&
            querySelection.from !== undefined &&
            querySelection.to !== undefined ? (
              <>
                {utilizationMetrics !== undefined ? (
                  <UtilizationGraphToShow utilizationMetrics={utilizationMetrics} />
                ) : (
                  <>
                    <ProfileMetricsGraph
                      queryClient={queryClient}
                      queryExpression={querySelection.expression}
                      from={querySelection.from}
                      to={querySelection.to}
                      profile={profileSelection}
                      comparing={comparing}
                      sumBy={querySelection.sumBy ?? sumBy ?? []}
                      sumByLoading={defaultSumByLoading}
                      setTimeRange={handleTimeRangeChange}
                      addLabelMatcher={addLabelMatcher}
                      onPointClick={handlePointClick}
                    />
                  </>
                )}
              </>
            ) : (
              profileSelection === null && (
                <div className="p-2">
                  <ProfileMetricsEmptyState message="Please select a profile type and click 'Search' to begin." />
                </div>
              )
            )}
          </div>
        </>
      )}
    </div>
  );
}
