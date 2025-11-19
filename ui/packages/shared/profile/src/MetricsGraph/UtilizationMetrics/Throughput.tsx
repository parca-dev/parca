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

import {useMemo} from 'react';

import {Icon} from '@iconify/react';
import {AnimatePresence, motion} from 'framer-motion';

import {
  DateTimeRange,
  MetricsGraphSkeleton,
  TextWithTooltip,
  useParcaContext,
} from '@parca/components';
import {formatDate, timePattern, valueFormatter} from '@parca/utilities';

import {type UtilizationMetrics as MetricSeries} from '../../ProfileSelector';
import MetricsGraph, {type ContextMenuItemOrSubmenu, type Series} from '../index';
import {useMetricsGraphDimensions} from '../useMetricsGraphDimensions';

interface NetworkLabel {
  name: string;
  value: string;
}

interface NetworkSeries {
  metric: NetworkLabel[];
  values: number[][];
  labelset: string;
  isReceive?: boolean;
  isSelected?: boolean;
}

interface CommonProps {
  transmitData: MetricSeries[];
  receiveData: MetricSeries[];
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void;
  setTimeRange: (range: DateTimeRange) => void;
  name: string;
  humanReadableName: string;
  from: number;
  to: number;
  selectedSeries?: Array<{key: string; value: string}>;
  onSeriesClick?: (name: string, seriesIndex: number) => void;
}

type RawAreaChartProps = CommonProps & {
  transformedData: Series[];
  width: number;
  height: number;
  margin: number;
  contextMenuItems?: ContextMenuItemOrSubmenu[];
};

type Props = CommonProps & {
  utilizationMetricsLoading?: boolean;
};

const transformUtilizationLabels = (label: string): string => {
  return label.replace('attributes.', '').replace('attributes_resource.', '');
};

const createThroughputContextMenuItems = (
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void,
  transmitData: MetricSeries[],
  receiveData: MetricSeries[]
): ContextMenuItemOrSubmenu[] => {
  const allData = [...transmitData, ...receiveData];

  return [
    {
      id: 'focus-on-single-series',
      label: 'Focus only on this series',
      icon: 'ph:star',
      onClick: (closestPoint, _series) => {
        if (
          closestPoint != null &&
          allData.length > 0 &&
          allData[closestPoint.seriesIndex] != null
        ) {
          const originalSeriesData = allData[closestPoint.seriesIndex];
          if (originalSeriesData.labelset?.labels != null) {
            const labels = originalSeriesData.labelset.labels.filter(
              label => label.name !== '__name__'
            );
            const labelsToAdd = labels.map(label => ({
              key: label.name,
              value: label.value,
            }));
            addLabelMatcher(labelsToAdd);
          }
        }
      },
    },
    {
      id: 'add-to-query',
      label: 'Add to query',
      icon: 'material-symbols:add',
      createDynamicItems: (closestPoint, _series) => {
        if (
          closestPoint == null ||
          allData.length === 0 ||
          allData[closestPoint.seriesIndex] == null
        ) {
          return [
            {
              id: 'no-labels-available',
              label: 'No labels available',
              icon: 'ph:warning',
              disabled: () => true,
              onClick: () => {}, // No-op for disabled item
            },
          ];
        }

        const originalSeriesData = allData[closestPoint.seriesIndex];
        if (originalSeriesData.labelset?.labels == null) {
          return [
            {
              id: 'no-labels-available',
              label: 'No labels available',
              icon: 'ph:warning',
              disabled: () => true,
              onClick: () => {}, // No-op for disabled item
            },
          ];
        }

        const labels = originalSeriesData.labelset.labels.filter(
          label => label.name !== '__name__'
        );

        return labels.map(label => ({
          id: `add-label-${label.name}`,
          label: (
            <div className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-300">
              {`${transformUtilizationLabels(label.name)}="${label.value}"`}
            </div>
          ),
          onClick: () => {
            addLabelMatcher({
              key: label.name,
              value: label.value,
            });
          },
        }));
      },
    },
  ];
};

interface MetricsSample {
  timestamp: number;
  value: number;
}

function transformToSeries(data: MetricSeries[], isReceive = false): NetworkSeries[] {
  const series: NetworkSeries[] = data.reduce<NetworkSeries[]>(function (
    agg: NetworkSeries[],
    s: MetricSeries
  ) {
    if (s.labelset !== undefined) {
      const metric = s.labelset.labels.sort((a, b) => a.name.localeCompare(b.name));
      agg.push({
        metric,
        values: s.samples.reduce<number[][]>(function (agg: number[][], d: MetricsSample) {
          if (d.timestamp !== undefined && d.value !== undefined) {
            // Multiply receive values by -1 to display below zero
            const value = isReceive ? -1 * d.value : d.value;
            agg.push([d.timestamp, value]);
          }
          return agg;
        }, []),
        labelset: metric.map(m => `${m.name}=${m.value}`).join(','),
        isReceive,
        isSelected: s.isSelected,
      });
    }
    return agg;
  }, []);

  // Sort values by timestamp for each series
  return series.map(series => ({
    ...series,
    values: series.values.sort((a, b) => a[0] - b[0]),
  }));
}

function transformNetworkSeriesToSeries(
  transmitData: MetricSeries[],
  receiveData: MetricSeries[]
): Series[] {
  const transmitSeries = transformToSeries(transmitData);
  const receiveSeries = transformToSeries(receiveData, true);
  const allSeries = [...transmitSeries, ...receiveSeries];

  return allSeries.map(networkSeries => {
    const labels = networkSeries.metric ?? [];
    const sortedLabels = labels
      .filter(label => label.name !== '__name__')
      .sort((a, b) => a.name.localeCompare(b.name));
    const labelString = sortedLabels.map(label => `${label.name}=${label.value}`).join(',');
    const id =
      (networkSeries.isReceive === true ? 'receive-' : 'transmit-') +
      (labelString !== '' ? labelString : 'default');

    return {
      id,
      values: networkSeries.values.map(([timestamp, value]): [number, number] => [
        timestamp,
        value,
      ]),
      highlighted: networkSeries.isSelected ?? false,
    };
  });
}

const RawAreaChart = ({
  transmitData,
  receiveData,
  transformedData,
  addLabelMatcher: _addLabelMatcher,
  setTimeRange,
  width,
  height,
  margin,
  humanReadableName,
  from,
  to,
  selectedSeries: _selectedSeries,
  onSeriesClick,
  contextMenuItems,
}: RawAreaChartProps): JSX.Element => {
  const {timezone} = useParcaContext();

  // Compute original series data for rich tooltip
  const allOriginalData = useMemo(
    () => [...transmitData, ...receiveData],
    [transmitData, receiveData]
  );

  return (
    <MetricsGraph
      data={transformedData}
      from={from}
      to={to}
      setTimeRange={setTimeRange}
      onSampleClick={closestPoint => {
        if (onSeriesClick != null) {
          onSeriesClick(humanReadableName, closestPoint.seriesIndex);
        }
      }}
      yAxisLabel={humanReadableName}
      yAxisUnit="bytes_per_second"
      width={width}
      height={height}
      margin={margin}
      contextMenuItems={contextMenuItems}
      renderTooltipContent={(seriesIndex: number, pointIndex: number) => {
        if (allOriginalData?.[seriesIndex]?.samples?.[pointIndex] != null) {
          const originalSeriesData = allOriginalData[seriesIndex];
          const originalPoint = allOriginalData[seriesIndex].samples[pointIndex];

          const labels = originalSeriesData.labelset?.labels ?? [];
          const nameLabel = labels.find(e => e.name === '__name__');
          const highlightedNameLabel = nameLabel ?? {name: '', value: ''};

          // Determine if this is receive data (negative values)
          const isReceive = seriesIndex >= transmitData.length;
          const valuePrefix = isReceive ? 'Receive ' : 'Transmit ';

          return (
            <div className="flex flex-row">
              <div className="ml-2 mr-6">
                <span className="font-semibold">{highlightedNameLabel.value}</span>
                <span className="my-2 block text-gray-700 dark:text-gray-300">
                  <table className="table-auto">
                    <tbody>
                      <tr>
                        <td className="w-1/4">{valuePrefix}Value</td>
                        <td className="w-3/4">
                          {valueFormatter(Math.abs(originalPoint.value), 'bytes_per_second', 2)}
                        </td>
                      </tr>
                      <tr>
                        <td className="w-1/4">At</td>
                        <td className="w-3/4">
                          {formatDate(
                            new Date(originalPoint.timestamp),
                            timePattern(timezone as string),
                            timezone
                          )}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </span>
                <span className="my-2 block text-gray-500">
                  {labels
                    .filter(label => label.name !== '__name__')
                    .map(label => (
                      <div
                        key={`${seriesIndex.toString()}-${pointIndex.toString()}-${label.name}`}
                        className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                      >
                        <TextWithTooltip
                          text={`${transformUtilizationLabels(label.name)}="${label.value}"`}
                          maxTextLength={37}
                          id={`${seriesIndex.toString()}-${pointIndex.toString()}-tooltip-${
                            label.name
                          }`}
                        />
                      </div>
                    ))}
                </span>
                <div className="flex w-full items-center gap-1 text-xs text-gray-500">
                  <Icon icon="iconoir:mouse-button-right" />
                  <div>Right click to add labels to query.</div>
                </div>
              </div>
            </div>
          );
        }
        return null;
      }}
    />
  );
};

const AreaChart = ({
  transmitData,
  receiveData,
  addLabelMatcher,
  setTimeRange,
  utilizationMetricsLoading,
  name,
  humanReadableName,
  from,
  to,
  selectedSeries,
  onSeriesClick,
}: Props): JSX.Element => {
  const {isDarkMode} = useParcaContext();
  const {width, height, margin, heightStyle} = useMetricsGraphDimensions(false, true);

  const transformedData = useMemo(
    () => transformNetworkSeriesToSeries(transmitData, receiveData),
    [transmitData, receiveData]
  );

  const contextMenuItems = useMemo(() => {
    return createThroughputContextMenuItems(addLabelMatcher, transmitData, receiveData);
  }, [addLabelMatcher, transmitData, receiveData]);

  return (
    <AnimatePresence>
      <motion.div
        className="w-full relative"
        key="area-chart-graph-loaded"
        initial={{display: 'none', opacity: 0}}
        animate={{display: 'block', opacity: 1}}
        transition={{duration: 0.5}}
      >
        {utilizationMetricsLoading === true ? (
          <MetricsGraphSkeleton heightStyle={heightStyle} isDarkMode={isDarkMode} isMini={true} />
        ) : (
          <RawAreaChart
            transmitData={transmitData}
            receiveData={receiveData}
            transformedData={transformedData}
            addLabelMatcher={addLabelMatcher}
            setTimeRange={setTimeRange}
            width={width}
            height={height}
            margin={margin}
            name={name}
            humanReadableName={humanReadableName}
            from={from}
            to={to}
            selectedSeries={selectedSeries}
            onSeriesClick={onSeriesClick}
            contextMenuItems={contextMenuItems}
          />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default AreaChart;
