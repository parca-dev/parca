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

import {Icon} from '@iconify/react';
import {AnimatePresence, motion} from 'framer-motion';

import {
  Label,
  MetricsSample,
  MetricsSeries as MetricsSeriesPb,
  QueryServiceClient,
} from '@parca/client';
import {
  DateTimeRange,
  MetricsGraphSkeleton,
  TextWithTooltip,
  useParcaContext,
} from '@parca/components';
import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import {capitalizeOnlyFirstLetter, formatDate, timePattern, valueFormatter} from '@parca/utilities';

import {MergedProfileSelection, ProfileSelection} from '..';
import MetricsGraph, {ContextMenuItemOrSubmenu, Series, SeriesPoint} from '../MetricsGraph';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import {useQueryRange} from './hooks/useQueryRange';

const transformUtilizationLabels = (label: string, utilizationMetrics: boolean): string => {
  if (utilizationMetrics) {
    return label.replace('attributes.', '').replace('attributes_resource.', '');
  }
  return label;
};

const createProfileContextMenuItems = (
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void,
  data: MetricsSeriesPb[], // The original MetricsSeriesPb[] data
  utilizationMetrics = false
): ContextMenuItemOrSubmenu[] => {
  return [
    {
      id: 'focus-on-single-series',
      label: 'Focus only on this series',
      icon: 'ph:star',
      onClick: (closestPoint, _series) => {
        if (closestPoint != null && data.length > 0 && data[closestPoint.seriesIndex] != null) {
          const originalSeriesData = data[closestPoint.seriesIndex];
          if (originalSeriesData.labelset?.labels != null) {
            const labels = originalSeriesData.labelset.labels.filter(
              (label: Label) => label.name !== '__name__'
            );
            const labelsToAdd = labels.map((label: Label) => ({
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
        if (closestPoint == null || data.length === 0 || data[closestPoint.seriesIndex] == null) {
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

        const originalSeriesData = data[closestPoint.seriesIndex];
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
          (label: Label) => label.name !== '__name__'
        );

        return labels.map((label: Label) => ({
          id: `add-label-${label.name}`,
          label: (
            <div className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-300">
              {`${transformUtilizationLabels(label.name, utilizationMetrics)}="${label.value}"`}
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

const transformMetricsData = (data: MetricsSeriesPb[]): Series[] => {
  const series = data.reduce<Series[]>((agg: Series[], s: MetricsSeriesPb) => {
    if (s.labelset !== undefined) {
      // Generate ID from sorted labelsets
      const labels = s.labelset.labels ?? [];
      const sortedLabels = labels
        .filter(label => label.name !== '__name__') // Exclude __name__ from ID generation
        .sort((a, b) => a.name.localeCompare(b.name));
      const id = sortedLabels.map(label => `${label.name}=${label.value}`).join(',');

      agg.push({
        id: id !== '' ? id : 'default', // fallback to 'default' if no labels
        values: s.samples.reduce<Array<[number, number]>>((agg, d: MetricsSample) => {
          if (d.timestamp !== undefined && d.valuePerSecond !== undefined) {
            const timestampMs = Number(d.timestamp.seconds) * 1000 + d.timestamp.nanos / 1_000_000;
            agg.push([timestampMs, d.valuePerSecond]);
          }
          return agg;
        }, []),
      });
    }
    return agg;
  }, []);

  return series;
};

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
    timestamp: bigint,
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
  const {onError, perf, authenticationErrorMessage, isDarkMode, timezone} = useParcaContext();
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

  const [originalSeries, {isTrimmed, beforeTrim, afterTrim}] = useMemo(() => {
    if (response?.series != null) {
      // Limit the number of series to 100 to avoid performance issues
      if (response.series.length > 100) {
        return [
          response.series.slice(0, 100),
          {isTrimmed: true, beforeTrim: response.series.length, afterTrim: 100},
        ];
      }
      return [response.series, {isTrimmed: false, beforeTrim: 0, afterTrim: 0}];
    }
    return [null, {isTrimmed: false, beforeTrim: 0, afterTrim: 0}];
  }, [response?.series]);

  const selectedPoint = useMemo((): SeriesPoint | null => {
    if (profile !== null && profile instanceof MergedProfileSelection) {
      // Iterate over the series and find the series index that matches all
      // labels of the profile selection. We specifically need the index
      // because that's what the SeriesPoint interface expects.
      const seriesIndex = originalSeries?.findIndex(s => {
        return s.labelset?.labels?.every(label => {
          return profile.query.matchers.some(matcher => {
            return matcher.key === label.name && matcher.value === label.value;
          });
        });
      });

      // if we found a series, return the point that matches the from/to timestamp exactly (in millisecond precision)
      if (
        seriesIndex !== undefined &&
        seriesIndex !== -1 &&
        originalSeries != null &&
        originalSeries[seriesIndex] != null
      ) {
        const series = originalSeries[seriesIndex];
        const pointIndex = series.samples.findIndex(sample => {
          return (
            sample.timestamp?.seconds === BigInt(profile.mergeFrom / 1_000_000_000n) &&
            sample.timestamp?.nanos === Number(profile.mergeFrom % 1_000_000_000n)
          );
        });

        if (pointIndex !== -1) {
          return {
            seriesIndex,
            pointIndex,
          };
        }
      }
      return null;
    }

    return null;
  }, [profile, originalSeries]);

  const transformedSeries = useMemo(() => {
    return originalSeries != null ? transformMetricsData(originalSeries) : [];
  }, [originalSeries]);

  const contextMenuItems = useMemo(() => {
    return originalSeries != null
      ? createProfileContextMenuItems(addLabelMatcher, originalSeries)
      : [];
  }, [originalSeries, addLabelMatcher]);

  const dataAvailable =
    originalSeries !== null && originalSeries !== undefined && originalSeries?.length > 0;

  const {sampleUnit, sampleType, yAxisLabel, yAxisUnit} = useMemo(() => {
    let sampleUnit = '';
    let sampleType = '';

    if (dataAvailable) {
      if (
        originalSeries?.every((val, i, arr) => val?.sampleType?.unit === arr[0]?.sampleType?.unit)
      ) {
        sampleUnit = originalSeries[0]?.sampleType?.unit ?? '';
        sampleType = originalSeries[0]?.sampleType?.type ?? '';
      }
      if (sampleUnit === '') {
        const profileType = Query.parse(queryExpression).profileType();
        sampleUnit = profileType.sampleUnit;
        sampleType = profileType.sampleType;
      }
    }

    // Calculate axis labels based on profile data
    const isDeltaType =
      profile !== null ? (profile as MergedProfileSelection)?.query.profType.delta : false;
    let yAxisLabel = sampleUnit;
    let yAxisUnit = sampleUnit;

    if (isDeltaType) {
      if (sampleUnit === 'nanoseconds') {
        if (sampleType === 'cpu') {
          yAxisLabel = 'CPU Cores';
          yAxisUnit = '';
        }
        if (sampleType === 'cuda') {
          yAxisLabel = 'GPU Time';
        }
      }
      if (sampleUnit === 'bytes') {
        yAxisLabel = 'Bytes per Second';
      }
    }

    return {sampleUnit, sampleType, yAxisLabel, yAxisUnit};
  }, [dataAvailable, originalSeries, queryExpression, profile]);

  const loading = metricsGraphLoading;

  // Handle errors after all hooks have been called
  if (!metricsGraphLoading && error !== null) {
    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
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
        {isTrimmed ? (
          <div className="flex justify-center">
            <span className="text-sm text-amber-800 dark:text-amber-200 bg-amber-100 dark:bg-amber-900 text-center px-2 rounded">
              Note: Showing only {afterTrim} of {new Intl.NumberFormat().format(beforeTrim)} series
              for performance reasons. Please narrow your query to view more.
            </span>
          </div>
        ) : null}
        {loading ? (
          <MetricsGraphSkeleton heightStyle={heightStyle} isDarkMode={isDarkMode} />
        ) : dataAvailable ? (
          <MetricsGraph
            data={transformedSeries}
            from={from}
            to={to}
            setTimeRange={setTimeRange}
            selectedPoint={selectedPoint}
            onSampleClick={(closestPoint: SeriesPoint): void => {
              // Use original data for both series and point
              if (originalSeries?.[closestPoint.seriesIndex] != null) {
                const originalSeriesData = originalSeries[closestPoint.seriesIndex];
                const originalPoint = originalSeriesData.samples[closestPoint.pointIndex];
                if (originalPoint.timestamp != null && originalPoint.valuePerSecond !== undefined) {
                  const timestampNanos =
                    originalPoint.timestamp.seconds * 1_000_000_000n +
                    BigInt(originalPoint.timestamp.nanos);
                  onPointClick(
                    timestampNanos, // Convert to number to match interface
                    originalSeriesData.labelset?.labels ?? [],
                    queryExpression,
                    Number(originalPoint.duration ?? 0) // Convert bigint to number
                  );
                }
              }
            }}
            renderTooltipContent={(seriesIndex: number, pointIndex: number) => {
              if (originalSeries?.[seriesIndex]?.samples?.[pointIndex] != null) {
                const originalSeriesData = originalSeries[seriesIndex];
                const originalPoint = originalSeriesData.samples[pointIndex];

                if (originalPoint.timestamp != null && originalPoint.valuePerSecond !== undefined) {
                  const timestampMs =
                    Number(originalPoint.timestamp.seconds) * 1000 +
                    originalPoint.timestamp.nanos / 1_000_000;
                  const labels = originalSeriesData.labelset?.labels ?? [];
                  const nameLabel = labels.find(e => e.name === '__name__');
                  const highlightedNameLabel = nameLabel ?? {name: '', value: ''};

                  // Calculate attributes maps for utilization metrics
                  const utilizationMetrics = false; // This is for profile metrics, not utilization
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

                  const isDeltaType =
                    profile !== null
                      ? (profile as MergedProfileSelection)?.query.profType.delta
                      : false;

                  return (
                    <div className="flex flex-row">
                      <div className="ml-2 mr-6">
                        <span className="font-semibold">{highlightedNameLabel.value}</span>
                        <span className="my-2 block text-gray-700 dark:text-gray-300">
                          <table className="table-auto">
                            <tbody>
                              {isDeltaType ? (
                                <>
                                  <tr>
                                    <td className="w-1/4 pr-3">Per&nbsp;Second</td>
                                    <td className="w-3/4">
                                      {valueFormatter(
                                        originalPoint.valuePerSecond,
                                        sampleUnit === 'nanoseconds' && sampleType === 'cpu'
                                          ? 'CPU Cores'
                                          : sampleUnit,
                                        5
                                      )}
                                    </td>
                                  </tr>
                                  <tr>
                                    <td className="w-1/4">Total</td>
                                    <td className="w-3/4">
                                      {valueFormatter(originalPoint.value ?? 0, sampleUnit, 2)}
                                    </td>
                                  </tr>
                                </>
                              ) : (
                                <tr>
                                  <td className="w-1/4">Value</td>
                                  <td className="w-3/4">
                                    {valueFormatter(originalPoint.valuePerSecond, sampleUnit, 5)}
                                  </td>
                                </tr>
                              )}
                              {originalPoint.duration != null &&
                                Number(originalPoint.duration) > 0 && (
                                  <tr>
                                    <td className="w-1/4">Duration</td>
                                    <td className="w-3/4">
                                      {valueFormatter(
                                        Number(originalPoint.duration.toString()),
                                        'nanoseconds',
                                        2
                                      )}
                                    </td>
                                  </tr>
                                )}
                              <tr>
                                <td className="w-1/4">At</td>
                                <td className="w-3/4">
                                  {formatDate(
                                    new Date(timestampMs),
                                    timePattern(timezone as string),
                                    timezone
                                  )}
                                </td>
                              </tr>
                            </tbody>
                          </table>
                        </span>
                        <span className="my-2 block text-gray-500">
                          {utilizationMetrics ? (
                            <>
                              {Object.keys(attributesResourceMap).length > 0 && (
                                <span className="text-sm font-bold text-gray-700 dark:text-white">
                                  Resource Attributes
                                </span>
                              )}
                              <span className="my-2 block text-gray-500">
                                {Object.keys(attributesResourceMap).map(name => (
                                  <div
                                    key={name}
                                    className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                                    {...testId(TEST_IDS.TOOLTIP_LABEL)}
                                  >
                                    <TextWithTooltip
                                      text={`${name.replace('attributes.', '')}="${
                                        attributesResourceMap[name]
                                      }"`}
                                      maxTextLength={48}
                                      id={`tooltip-${name}-${attributesResourceMap[name]}`}
                                    />
                                  </div>
                                ))}
                              </span>
                              {Object.keys(attributesMap).length > 0 && (
                                <span className="text-sm font-bold text-gray-700 dark:text-white">
                                  Attributes
                                </span>
                              )}
                              <span className="my-2 block text-gray-500">
                                {Object.keys(attributesMap).map(name => (
                                  <div
                                    key={name}
                                    className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                                    {...testId(TEST_IDS.TOOLTIP_LABEL)}
                                  >
                                    <TextWithTooltip
                                      text={`${name.replace('attributes.', '')}="${
                                        attributesMap[name]
                                      }"`}
                                      maxTextLength={48}
                                      id={`tooltip-${name}-${attributesMap[name]}`}
                                    />
                                  </div>
                                ))}
                              </span>
                            </>
                          ) : (
                            <>
                              {labels
                                .filter((label: Label) => label.name !== '__name__')
                                .map((label: Label) => (
                                  <div
                                    key={label.name}
                                    className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                                    {...testId(TEST_IDS.TOOLTIP_LABEL)}
                                  >
                                    <TextWithTooltip
                                      text={`${label.name}="${label.value}"`}
                                      maxTextLength={37}
                                      id={`tooltip-${label.name}`}
                                    />
                                  </div>
                                ))}
                            </>
                          )}
                        </span>
                        <div className="flex w-full items-center gap-1 text-xs text-gray-500">
                          <Icon icon="iconoir:mouse-button-right" />
                          <div>Right click to add labels to query.</div>
                        </div>
                      </div>
                    </div>
                  );
                }
              }
              return null;
            }}
            yAxisLabel={yAxisLabel}
            yAxisUnit={yAxisUnit}
            height={height}
            width={width}
            margin={margin}
            contextMenuItems={contextMenuItems}
          />
        ) : (
          <ProfileMetricsEmptyState message="No data found. Try a different query." />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileMetricsGraph;
