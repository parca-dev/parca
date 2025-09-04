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

import {useState} from 'react';

import {DateTimeRange} from '@parca/components';
import {Query} from '@parca/parser';

import UtilizationMetricsGraph from '../MetricsGraph/UtilizationMetricsGraph';
import AreaChart from '../MetricsGraph/UtilizationMetricsGraph/Throughput';
import {SharedQueryControls} from '../SharedQueryControls';
import {
  UtilizationLabels,
  UtilizationMetrics,
  UtilizationProvider,
} from '../contexts/UtilizationContext';

export interface QuerySelection {
  expression: string;
  from: number;
  to: number;
  timeSelection: string;
  sumBy?: string[];
  mergeFrom?: string;
  mergeTo?: string;
}

interface UtilizationMetricsSelectorProps {
  queryClient: any;
  querySelection: QuerySelection;
  selectQuery: (query: QuerySelection) => void;
  utilizationMetrics?: UtilizationMetrics;
  utilizationLabels?: UtilizationLabels;
  onUtilizationSeriesSelect?: (seriesIndex: number) => void;
}

export const UtilizationMetricsSelector = ({
  queryClient,
  querySelection,
  selectQuery,
  utilizationMetrics,
  utilizationLabels,
  onUtilizationSeriesSelect,
}: UtilizationMetricsSelectorProps): JSX.Element => {
  const [timeRangeSelection, setTimeRangeSelection] = useState(
    DateTimeRange.fromRangeKey(querySelection.timeSelection, querySelection.from, querySelection.to)
  );

  const [queryExpressionString, setQueryExpressionString] = useState(querySelection.expression);

  const query = Query.parse(queryExpressionString);
  const selectedProfileName = query.profileName();

  const setQueryExpression = (updateTs = false): void => {
    const from = timeRangeSelection.getFromMs(updateTs);
    const to = timeRangeSelection.getToMs(updateTs);

    selectQuery({
      expression: queryExpressionString,
      from,
      to,
      timeSelection: timeRangeSelection.getRangeKey(),
    });
  };

  const handleTimeRangeChange = (range: DateTimeRange): void => {
    const from = range.getFromMs();
    const to = range.getToMs();

    setTimeRangeSelection(range);
    selectQuery({
      expression: queryExpressionString,
      from,
      to,
      timeSelection: range.getRangeKey(),
    });
  };

  const setMatchersString = (matchers: string): void => {
    const newExpressionString = `${selectedProfileName}{${matchers}}`;
    setQueryExpressionString(newExpressionString);
  };

  const searchDisabled =
    queryExpressionString === undefined ||
    queryExpressionString === '' ||
    queryExpressionString === '{}';

  const contextValue = {
    utilizationMetrics,
    utilizationLabels,
    onUtilizationSeriesSelect,
  };

  return (
    <UtilizationProvider value={contextValue}>
      <div>
        <div className="mb-2">
          <SharedQueryControls
            queryClient={queryClient}
            query={query}
            profileType={selectedProfileName}
            timeRangeSelection={timeRangeSelection}
            setTimeRangeSelection={setTimeRangeSelection}
            setMatchersString={setMatchersString}
            setQueryExpression={setQueryExpression}
            searchDisabled={searchDisabled}
            showAdvancedMode={true}
          />
        </div>

        {utilizationMetrics !== undefined && utilizationMetrics.metrics.length > 0 && (
          <div>
            {utilizationMetrics.metrics
              .filter(({renderAs}) => renderAs !== 'throughput')
              .map(({name, humanReadableName, data, yAxisUnit}) => (
                <UtilizationMetricsGraph
                  key={name}
                  data={data}
                  setTimeRange={handleTimeRangeChange}
                  utilizationMetricsLoading={utilizationMetrics.loading}
                  humanReadableName={humanReadableName}
                  from={querySelection.from}
                  to={querySelection.to}
                  yAxisUnit={yAxisUnit ?? 'percent'}
                  onSeriesClick={() => {
                    if (onUtilizationSeriesSelect != null) {
                      const globalSeriesIndex =
                        utilizationMetrics?.metrics.findIndex(metric => metric.name === name) ?? 0;

                      onUtilizationSeriesSelect(globalSeriesIndex);
                    }
                  }}
                />
              ))}

            {(() => {
              const throughputMetrics = utilizationMetrics.metrics.filter(
                metric => metric.renderAs === 'throughput'
              );

              if (throughputMetrics.length === 0) return null;

              const groupedMetrics = throughputMetrics.filter(
                metric => metric.groupWith != null && metric.groupWith.length > 0
              );

              if (groupedMetrics.length === 0) return null;

              const primaryMetric = groupedMetrics[0];
              const secondaryMetricName = primaryMetric.groupWith?.[0];

              if (secondaryMetricName == null || secondaryMetricName === '') return null;

              const secondaryMetric = throughputMetrics.find(m => m.name === secondaryMetricName);

              if (primaryMetric.data.length === 0 && (secondaryMetric?.data.length ?? 0) === 0)
                return null;

              return (
                <AreaChart
                  transmitData={primaryMetric.data}
                  receiveData={secondaryMetric?.data ?? []}
                  addLabelMatcher={() => {}}
                  setTimeRange={handleTimeRangeChange}
                  name={primaryMetric.name}
                  humanReadableName={primaryMetric.humanReadableName}
                  from={querySelection.from}
                  to={querySelection.to}
                  yAxisUnit={primaryMetric.yAxisUnit ?? 'bytes_per_second'}
                  utilizationMetricsLoading={utilizationMetrics.loading}
                  selectedSeries={undefined}
                  onSeriesClick={() => {
                    if (onUtilizationSeriesSelect != null) {
                      const globalSeriesIndex =
                        utilizationMetrics?.metrics.findIndex(
                          metric => metric.renderAs === 'throughput'
                        ) ?? 0;
                      onUtilizationSeriesSelect(globalSeriesIndex);
                    }
                  }}
                />
              );
            })()}
          </div>
        )}
      </div>
    </UtilizationProvider>
  );
};

export default UtilizationMetricsSelector;
