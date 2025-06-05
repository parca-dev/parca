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

import {Fragment, useCallback, useId, useMemo, useRef, useState} from 'react';

import * as d3 from 'd3';
import {pointer} from 'd3-selection';
import {AnimatePresence, motion} from 'framer-motion';
import throttle from 'lodash.throttle';
import {useContextMenu} from 'react-contexify';

import {DateTimeRange, MetricsGraphSkeleton, useParcaContext, useURLState} from '@parca/components';
import {Matcher} from '@parca/parser';
import {formatDate, formatForTimespan, getPrecision, valueFormatter} from '@parca/utilities';

import {type UtilizationMetrics as MetricSeries} from '../../ProfileSelector';
import MetricsContextMenu from '../MetricsContextMenu';
import MetricsTooltip from '../MetricsTooltip';
import {useMetricsGraphDimensions} from '../useMetricsGraphDimensions';
import {getSeriesColor} from '../utils/colorMapping';

interface NetworkLabel {
  name: string;
  value: string;
}

interface NetworkSeries {
  metric: NetworkLabel[];
  values: number[][];
  labelset: string;
  isReceive?: boolean;
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
}

type RawAreaChartProps = CommonProps & {
  width: number;
  height: number;
  margin: number;
};

type Props = CommonProps & {
  utilizationMetricsLoading?: boolean;
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
      });
    }
    return agg;
  },
  []);

  // Sort values by timestamp for each series
  return series.map(series => ({
    ...series,
    values: series.values.sort((a, b) => a[0] - b[0]),
  }));
}

const RawAreaChart = ({
  transmitData,
  receiveData,
  addLabelMatcher,
  setTimeRange,
  width,
  height,
  margin,
  humanReadableName,
  from,
  to,
}: RawAreaChartProps): JSX.Element => {
  const {timezone} = useParcaContext();
  const graph = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [hovering, setHovering] = useState(false);
  const [relPos, setRelPos] = useState(-1);
  const [pos, setPos] = useState([0, 0]);
  const [isContextMenuOpen, setIsContextMenuOpen] = useState<boolean>(false);
  const idForContextMenu = useId();
  const [selectedSeries, setSelectedSeries] = useURLState<string>('selectedSeries');
  const [_, setSelectedTimeframe] = useURLState('gpu_selected_timeframe');

  const parsedSelectedSeries: Matcher[] = useMemo(() => {
    if (selectedSeries === undefined) {
      return [];
    }

    return JSON.parse(decodeURIComponent(selectedSeries));
  }, [selectedSeries]);

  const lineStroke = '1px';
  const lineStrokeHover = '2px';
  const lineStrokeSelected = '3px';

  const graphWidth = width - margin * 1.5 - margin / 2;

  const paddedFrom = from;
  const paddedTo = to;

  const series = useMemo(() => {
    const transmitSeries = transformToSeries(transmitData);
    const receiveSeries = transformToSeries(receiveData, true);
    return [...transmitSeries, ...receiveSeries];
  }, [transmitData, receiveData]);

  const extentsY = series.map(function (s) {
    return d3.extent(s.values, function (d) {
      return d[1];
    });
  });

  const minY = d3.min(extentsY, function (d) {
    return d[0];
  });

  const maxY = d3.max(extentsY, function (d) {
    return d[1];
  });

  // Setup scales with padded time range
  const xScale = d3.scaleUtc().domain([paddedFrom, paddedTo]).range([0, graphWidth]);

  const yScale = d3
    .scaleLinear()
    // Ensure domain is symmetric around 0 for balanced visualization
    .domain([minY ?? 0, maxY ?? 0])
    .range([height - margin, 0])
    .nice();

  const throttledSetPos = throttle(setPos, 20);

  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    if (isContextMenuOpen) {
      return;
    }

    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    const xCoordinate = rel[0];
    const xCoordinateWithoutMargin = xCoordinate - margin;
    const yCoordinate = rel[1];
    const yCoordinateWithoutMargin = yCoordinate - margin;

    throttledSetPos([xCoordinateWithoutMargin, yCoordinateWithoutMargin]);
  };

  const trackVisibility = (isVisible: boolean): void => {
    setIsContextMenuOpen(isVisible);
  };

  const MENU_ID = `areachart-context-menu-${idForContextMenu}`;

  const {show} = useContextMenu({
    id: MENU_ID,
  });

  const displayMenu = useCallback(
    (e: React.MouseEvent): void => {
      show({
        event: e,
      });
    },
    [show]
  );

  // Create line generator for both transmit and receive
  const lineGenerator = d3
    .line<number[]>()
    .x(d => xScale(d[0]))
    .y(d => yScale(d[1]));

  const highlighted = useMemo(() => {
    if (series.length === 0) {
      return null;
    }

    // Return the closest point as the highlighted point
    const closestPointPerSeries = series.map(function (s) {
      const distances = s.values.map(d => {
        const x = xScale(d[0]) + margin / 2;
        const y = yScale(d[1]) - margin / 3;

        return Math.sqrt(Math.pow(pos[0] - x, 2) + Math.pow(pos[1] - y, 2));
      });

      const pointIndex = d3.minIndex(distances);
      const minDistance = distances[pointIndex];

      return {
        pointIndex,
        distance: minDistance,
      };
    });

    const closestSeriesIndex = d3.minIndex(closestPointPerSeries, s => s.distance);
    const pointIndex = closestPointPerSeries[closestSeriesIndex].pointIndex;
    const point = series[closestSeriesIndex].values[pointIndex];
    return {
      seriesIndex: closestSeriesIndex,
      labels: series[closestSeriesIndex].metric,
      timestamp: point[0],
      valuePerSecond: point[1],
      value: point[2],
      duration: point[3],
      x: xScale(point[0]),
      y: yScale(point[1]),
    };
  }, [pos, series, xScale, yScale, margin]);

  const onMouseDown = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    // only left mouse button
    if (e.button !== 0) {
      return;
    }

    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    const xCoordinate = rel[0];
    const xCoordinateWithoutMargin = xCoordinate - margin;
    if (xCoordinateWithoutMargin >= 0) {
      setRelPos(xCoordinateWithoutMargin);
      setDragging(true);
    }

    e.stopPropagation();
    e.preventDefault();
  };

  const onMouseUp = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    setDragging(false);

    if (relPos === -1) {
      // MouseDown happened outside of this element.
      return;
    }

    // This is a normal click. We tolerate tiny movements to still be a
    // click as they can occur when clicking based on user feedback.
    if (Math.abs(relPos - pos[0]) <= 1) {
      setRelPos(-1);
      return;
    }

    let startPos = relPos;
    let endPos = pos[0];

    if (startPos > endPos) {
      startPos = pos[0];
      endPos = relPos;
    }

    const startCorrection = 10;
    const endCorrection = 30;

    const firstTime = xScale.invert(startPos - startCorrection).valueOf();
    const secondTime = xScale.invert(endPos - endCorrection).valueOf();

    setTimeRange(DateTimeRange.fromAbsoluteDates(firstTime, secondTime));

    setRelPos(-1);

    e.stopPropagation();
    e.preventDefault();
  };

  return (
    <>
      <MetricsContextMenu
        onAddLabelMatcher={addLabelMatcher}
        menuId={MENU_ID}
        highlighted={highlighted}
        trackVisibility={trackVisibility}
        utilizationMetrics={true}
      />

      {highlighted != null && hovering && !dragging && pos[0] !== 0 && pos[1] !== 0 && (
        <div
          onMouseMove={onMouseMove}
          onMouseEnter={() => setHovering(true)}
          onMouseLeave={() => setHovering(false)}
        >
          {!isContextMenuOpen && (
            <MetricsTooltip
              x={pos[0] + margin}
              y={pos[1] + margin}
              highlighted={{
                ...highlighted,
                valuePerSecond: Math.abs(highlighted.valuePerSecond),
              }}
              contextElement={graph.current}
              sampleType={'throughput'}
              sampleUnit={'bytes_per_second'}
              delta={false}
              utilizationMetrics={true}
              valuePrefix={
                highlighted.seriesIndex >= transmitData.length ? 'Receive ' : 'Transmit '
              }
            />
          )}
        </div>
      )}
      <div
        ref={graph}
        onMouseEnter={() => setHovering(true)}
        onMouseLeave={() => setHovering(false)}
        onContextMenu={displayMenu}
      >
        <svg
          width={`${width}px`}
          height={`${height + margin}px`}
          onMouseDown={onMouseDown}
          onMouseUp={onMouseUp}
          onMouseMove={onMouseMove}
        >
          <g transform={`translate(${margin}, 0)`}>
            {dragging && (
              <g className="zoom-time-rect">
                <rect
                  className="bar"
                  x={pos[0] - relPos < 0 ? pos[0] : relPos}
                  y={0}
                  height={height}
                  width={Math.abs(pos[0] - relPos)}
                  fill={'rgba(0, 0, 0, 0.125)'}
                />
              </g>
            )}
          </g>
          <g transform={`translate(${margin * 1.5}, ${margin / 1.5})`}>
            <g className="y axis" textAnchor="end" fontSize="10" fill="none">
              {yScale.ticks(6).map((d, i, allTicks) => {
                let decimals = 2;
                const intervalBetweenTicks = allTicks[1] - allTicks[0];

                if (intervalBetweenTicks < 1) {
                  const precision = getPrecision(intervalBetweenTicks);
                  decimals = precision;
                }

                return (
                  <Fragment key={`${i.toString()}-${d.toString()}`}>
                    <g key={`tick-${i}`} className="tick" transform={`translate(0, ${yScale(d)})`}>
                      <line className="stroke-gray-300 dark:stroke-gray-500" x2={-6} />
                      <text fill="currentColor" x={-9} dy={'0.32em'}>
                        {d < 0 ? '-' : ''}
                        {valueFormatter(Math.abs(d), 'bytes_per_second', decimals)}
                      </text>
                    </g>
                    <g key={`grid-${i}`}>
                      <line
                        className="stroke-gray-300 dark:stroke-gray-500"
                        x1={xScale(from)}
                        x2={xScale(to)}
                        y1={yScale(d)}
                        y2={yScale(d)}
                      />
                    </g>
                  </Fragment>
                );
              })}
              <line
                className="stroke-gray-300 dark:stroke-gray-500"
                x1={0}
                x2={0}
                y1={0}
                y2={height - margin}
              />
              <line
                className="stroke-gray-300 dark:stroke-gray-500"
                x1={xScale(to)}
                x2={xScale(to)}
                y1={0}
                y2={height - margin}
              />
              <g transform={`translate(${-margin}, ${(height - margin) / 2}) rotate(270)`}>
                <text
                  fill="currentColor"
                  dy="-0.7em"
                  className="text-sm capitalize"
                  textAnchor="middle"
                >
                  {humanReadableName}
                </text>
              </g>
            </g>
            <g
              className="x axis"
              fill="none"
              fontSize="10"
              textAnchor="middle"
              transform={`translate(0,${height - margin})`}
            >
              {xScale.ticks(5).map((d, i) => (
                <Fragment key={`${i.toString()}-${d.toString()}`}>
                  <g
                    key={`tick-${i}`}
                    className="tick"
                    /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                    transform={`translate(${xScale(d)}, 0)`}
                  >
                    <line y2={6} className="stroke-gray-300 dark:stroke-gray-500" />
                    <text fill="currentColor" dy=".71em" y={9}>
                      {formatDate(d, formatForTimespan(from, to), timezone)}
                    </text>
                  </g>
                  <g key={`grid-${i}`}>
                    <line
                      className="stroke-gray-300 dark:stroke-gray-500"
                      x1={xScale(d)}
                      x2={xScale(d)}
                      y1={0}
                      y2={-height + margin}
                    />
                  </g>
                </Fragment>
              ))}
              <line
                className="stroke-gray-300 dark:stroke-gray-500"
                x1={0}
                x2={graphWidth}
                y1={0}
                y2={0}
              />
              <g transform={`translate(${(width - 2.5 * margin) / 2}, ${margin / 2})`}>
                <text fill="currentColor" dy=".71em" y={5} className="text-sm">
                  Time
                </text>
              </g>
            </g>
            <g className="areas">
              {/* Draw baseline at y=0 */}
              <line
                x1={xScale(from)}
                x2={xScale(to)}
                y1={yScale(0)}
                y2={yScale(0)}
                stroke="#64748b"
                strokeDasharray="4 2"
                strokeWidth={1}
                opacity={0.7}
              />
              {series.map((s, i) => {
                let isSelected = false;
                if (parsedSelectedSeries != null && parsedSelectedSeries.length > 0) {
                  isSelected = parsedSelectedSeries.every(m => {
                    for (let i = 0; i < s.metric.length; i++) {
                      if (s.metric[i].name === m.key && s.metric[i].value === m.value) {
                        return true;
                      }
                    }
                    return false;
                  });
                }

                return (
                  <g key={i} className="line cursor-pointer">
                    <path
                      d={lineGenerator(s.values) ?? ''}
                      fill="none"
                      stroke={getSeriesColor(s.metric)}
                      strokeWidth={
                        isSelected
                          ? lineStrokeSelected
                          : hovering && highlighted != null && i === highlighted.seriesIndex
                          ? lineStrokeHover
                          : lineStroke
                      }
                      strokeOpacity={isSelected ? 1 : 0.8}
                      onClick={() => {
                        if (highlighted != null) {
                          setSelectedSeries(
                            JSON.stringify(
                              highlighted.labels.map(l => ({
                                key: l.name,
                                value: l.value,
                              }))
                            )
                          );
                          setSelectedTimeframe(undefined);
                        }
                      }}
                    />
                  </g>
                );
              })}
            </g>
          </g>
        </svg>
      </div>
    </>
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
}: Props): JSX.Element => {
  const {isDarkMode} = useParcaContext();
  const {width, height, margin, heightStyle} = useMetricsGraphDimensions(false, true);

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
            addLabelMatcher={addLabelMatcher}
            setTimeRange={setTimeRange}
            width={width}
            height={height}
            margin={margin}
            name={name}
            humanReadableName={humanReadableName}
            from={from}
            to={to}
          />
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default AreaChart;
