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

interface CommonProps {
  setTimeRange: (range: DateTimeRange) => void;
  humanReadableName: string;
  from: number;
  to: number;
  onSeriesClick?: (seriesIndex: number) => void;
}

type RawUtilizationMetricsProps = CommonProps & {
  data: Series[];
  originalData: MetricSeries[];
  width: number;
  height: number;
  margin: number;
  yAxisUnit: string;
  contextMenuItems?: ContextMenuItemOrSubmenu[];
};

type Props = CommonProps & {
  data: MetricSeries[];
  yAxisUnit: string;
  utilizationMetricsLoading?: boolean;
  addLabelMatcher?: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void;
  onSelectedSeriesChange?: (series: Array<{key: string; value: string}>) => void;
};

const transformUtilizationLabels = (label: string): string => {
  return label.replace('attributes.', '').replace('attributes_resource.', '');
};

const createUtilizationContextMenuItems = (
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void,
  originalData: MetricSeries[]
): ContextMenuItemOrSubmenu[] => {
  return [
    {
      id: 'focus-on-single-series',
      label: 'Focus only on this series',
      icon: 'ph:star',
      onClick: (closestPoint, _series) => {
        if (
          closestPoint != null &&
          originalData.length > 0 &&
          originalData[closestPoint.seriesIndex] != null
        ) {
          const originalSeriesData = originalData[closestPoint.seriesIndex];
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
          originalData.length === 0 ||
          originalData[closestPoint.seriesIndex] == null
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

        const originalSeriesData = originalData[closestPoint.seriesIndex];
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

const transformMetricSeriesToSeries = (data: MetricSeries[]): Series[] => {
  return data.map(metricSeries => {
    if (metricSeries.labelset != null) {
      const labels = metricSeries.labelset.labels ?? [];
      const sortedLabels = labels.sort((a, b) => a.name.localeCompare(b.name));
      const id = sortedLabels.map(label => `${label.name}=${label.value}`).join(',');

      return {
        id: id !== '' ? id : 'default',
        values: metricSeries.samples.map((sample): [number, number] => [
          sample.timestamp,
          sample.value,
        ]),
      };
    }
    return {
      id: 'default',
      values: [],
    };
  });
};

const _getYAxisUnit = (name: string): string => {
  switch (name) {
    case 'gpu_power_watt':
      return 'watts';
    case 'gpu_temperature_celsius':
      return 'celsius';
    case 'gpu_clock_hertz':
      return 'hertz';
    default:
      return 'percent';
  }
};

const RawUtilizationMetrics = ({
  data,
  originalData,
  setTimeRange,
  width,
  height,
  margin,
  humanReadableName,
  from,
  to,
  yAxisUnit,
  contextMenuItems,
  onSeriesClick,
}: RawUtilizationMetricsProps): JSX.Element => {
  const {timezone} = useParcaContext();

  return (
    <MetricsGraph
      data={data.map((val, idx) => ({
        ...val,
        highlighted: originalData?.[idx]?.isSelected ?? false,
      }))}
      from={from}
      to={to}
      setTimeRange={setTimeRange}
      onSampleClick={closestPoint => {
        if (onSeriesClick != null) {
          onSeriesClick(closestPoint.seriesIndex);
        }
      }}
      yAxisLabel={humanReadableName}
      yAxisUnit={yAxisUnit}
      width={width}
      height={height}
      margin={margin}
      contextMenuItems={contextMenuItems}
      renderTooltipContent={(seriesIndex: number, pointIndex: number) => {
        if (originalData?.[seriesIndex]?.samples?.[pointIndex] != null) {
          const originalSeriesData = originalData[seriesIndex];
          const originalPoint = originalData[seriesIndex].samples[pointIndex];

          const labels = originalSeriesData.labelset?.labels ?? [];
          const nameLabel = labels.find(e => e.name === '__name__');
          const highlightedNameLabel = nameLabel ?? {name: '', value: ''};

          // Calculate attributes maps for utilization metrics
          const attributesMap = labels
            .filter(
              label =>
                label.name.startsWith('attributes.') &&
                !label.name.startsWith('attributes_resource.')
            )
            .reduce<Record<string, string>>((acc, label) => {
              const key = label.name.replace('attributes.', '');
              acc[key] = label.value;
              return acc;
            }, {});

          const attributesResourceMap = labels
            .filter(label => label.name.startsWith('attributes_resource.'))
            .reduce<Record<string, string>>((acc, label) => {
              const key = label.name.replace('attributes_resource.', '');
              acc[key] = label.value;
              return acc;
            }, {});

          return (
            <div className="flex flex-row">
              <div className="ml-2 mr-6">
                <span className="font-semibold">{highlightedNameLabel.value}</span>
                <span className="my-2 block text-gray-700 dark:text-gray-300">
                  <table className="table-auto">
                    <tbody>
                      <tr>
                        <td className="w-1/4">Value</td>
                        <td className="w-3/4">
                          {valueFormatter(originalPoint.value, yAxisUnit, 2)}
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
                  {Object.keys(attributesResourceMap).length > 0 ? (
                    <span className="text-sm font-bold text-gray-700 dark:text-white">
                      Resource Attributes
                    </span>
                  ) : null}
                  <span className="my-2 block text-gray-500">
                    {Object.keys(attributesResourceMap).map(name => (
                      <div
                        key={
                          'resourceattribute-' +
                          seriesIndex.toString() +
                          '-' +
                          pointIndex.toString() +
                          '-' +
                          name
                        }
                        className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                      >
                        <TextWithTooltip
                          text={`${transformUtilizationLabels(name)}="${
                            attributesResourceMap[name] ?? ''
                          }"`}
                          maxTextLength={48}
                          id={`tooltip-${name}-${attributesResourceMap[name] ?? ''}`}
                        />
                      </div>
                    ))}
                  </span>
                  {Object.keys(attributesMap).length > 0 ? (
                    <span className="text-sm font-bold text-gray-700 dark:text-white">
                      Attributes
                    </span>
                  ) : null}
                  <span className="my-2 block text-gray-500">
                    {Object.keys(attributesMap).map(name => (
                      <div
                        key={
                          'attribute-' +
                          seriesIndex.toString() +
                          '-' +
                          pointIndex.toString() +
                          '-' +
                          name
                        }
                        className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                      >
                        <TextWithTooltip
                          text={`${transformUtilizationLabels(name)}="${
                            attributesMap[name] ?? ''
                          }"`}
                          maxTextLength={48}
                          id={`tooltip-${name}-${attributesMap[name] ?? ''}`}
                        />
                      </div>
                    ))}
                  </span>
                  {labels
                    .filter(
                      label => label.name !== '__name__' && !label.name.startsWith('attributes')
                    )
                    .map(label => (
                      <div
                        key={
                          'attribute-' +
                          seriesIndex.toString() +
                          '-' +
                          pointIndex.toString() +
                          '-label-' +
                          label.name
                        }
                        className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                      >
                        <TextWithTooltip
                          text={`${transformUtilizationLabels(label.name)}="${label.value}"`}
                          maxTextLength={37}
                          id={`tooltip-${label.name}`}
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

const UtilizationMetrics = ({
  data,
  setTimeRange,
  utilizationMetricsLoading,
  humanReadableName,
  from,
  to,
  yAxisUnit,
  addLabelMatcher,
  onSeriesClick,
  onSelectedSeriesChange: _onSelectedSeriesChange,
}: Props): JSX.Element => {
  const {isDarkMode} = useParcaContext();
  const {width, height, margin, heightStyle} = useMetricsGraphDimensions(false, true);

  const transformedData = useMemo(() => transformMetricSeriesToSeries(data), [data]);

  const contextMenuItems = useMemo(() => {
    return addLabelMatcher != null ? createUtilizationContextMenuItems(addLabelMatcher, data) : [];
  }, [addLabelMatcher, data]);

  return (
    <AnimatePresence>
      <motion.div
        className="w-full relative"
        key="utilization-metrics-graph-loaded"
        initial={{display: 'none', opacity: 0}}
        animate={{display: 'block', opacity: 1}}
        transition={{duration: 0.5}}
      >
        {utilizationMetricsLoading === true ? (
          <MetricsGraphSkeleton heightStyle={heightStyle} isDarkMode={isDarkMode} isMini={true} />
        ) : (
          <RawUtilizationMetrics
            data={transformedData}
            originalData={data}
            setTimeRange={setTimeRange}
            width={width}
            height={height}
            margin={margin}
            humanReadableName={humanReadableName}
            from={from}
            to={to}
            yAxisUnit={yAxisUnit}
            contextMenuItems={contextMenuItems}
            onSeriesClick={onSeriesClick}
          />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default UtilizationMetrics;
