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

import React, {Fragment, useCallback, useId, useMemo, useRef, useState} from 'react';

import * as d3 from 'd3';
import {pointer} from 'd3-selection';
import throttle from 'lodash.throttle';
import {useContextMenu} from 'react-contexify';

import {Label, MetricsSample, MetricsSeries as MetricsSeriesPb} from '@parca/client';
import {DateTimeRange, useParcaContext} from '@parca/components';
import {
  formatDate,
  formatForTimespan,
  getPrecision,
  sanitizeHighlightedValues,
  valueFormatter,
} from '@parca/utilities';

import {MergedProfileSelection} from '..';
import MetricsCircle from '../MetricsCircle';
import MetricsSeries from '../MetricsSeries';
import MetricsContextMenu from './MetricsContextMenu';
import MetricsInfoPanel from './MetricsInfoPanel';
import MetricsTooltip from './MetricsTooltip';

interface Props {
  data: MetricsSeriesPb[];
  from: number;
  to: number;
  profile: MergedProfileSelection | null;
  onSampleClick: (timestamp: number, value: number, labels: Label[], duration: number) => void;
  addLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void;
  setTimeRange: (range: DateTimeRange) => void;
  sampleType: string;
  sampleUnit: string;
  width?: number;
  height?: number;
  margin?: number;
  sumBy?: string[];
}

export interface HighlightedSeries {
  seriesIndex: number;
  labels: Label[];
  timestamp: number;
  value: number;
  valuePerSecond: number;
  duration: number;
  x: number;
  y: number;
}

export interface Series {
  metric: Label[];
  values: number[][];
  labelset: string;
}

const MetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  addLabelMatcher,
  setTimeRange,
  sampleType,
  sampleUnit,
  width = 0,
  height = 0,
  margin = 0,
  sumBy,
}: Props): JSX.Element => {
  const [isInfoPanelOpen, setIsInfoPanelOpen] = useState<boolean>(false);
  return (
    <div className="relative" onClick={() => isInfoPanelOpen && setIsInfoPanelOpen(false)}>
      <div className="absolute right-0 top-0">
        <MetricsInfoPanel
          isInfoPanelOpen={isInfoPanelOpen}
          onInfoIconClick={() => setIsInfoPanelOpen(true)}
        />
      </div>
      <RawMetricsGraph
        data={data}
        from={from}
        to={to}
        profile={profile}
        onSampleClick={onSampleClick}
        addLabelMatcher={addLabelMatcher}
        setTimeRange={setTimeRange}
        sampleType={sampleType}
        sampleUnit={sampleUnit}
        width={width}
        height={height}
        margin={margin}
        sumBy={sumBy}
      />
    </div>
  );
};

export default MetricsGraph;

export const parseValue = (value: string): number | null => {
  const val = parseFloat(value);
  // "+Inf", "-Inf", "+Inf" will be parsed into NaN by parseFloat(). They
  // can't be graphed, so show them as gaps (null).
  return isNaN(val) ? null : val;
};

const lineStroke = '1px';
const lineStrokeHover = '2px';

export const RawMetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  addLabelMatcher,
  setTimeRange,
  sampleType,
  sampleUnit,
  width,
  height = 50,
  margin = 0,
  sumBy,
}: Props): JSX.Element => {
  const {timezone} = useParcaContext();
  const graph = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [hovering, setHovering] = useState(false);
  const [relPos, setRelPos] = useState(-1);
  const [pos, setPos] = useState([0, 0]);
  const [isContextMenuOpen, setIsContextMenuOpen] = useState<boolean>(false);
  const metricPointRef = useRef(null);
  const idForContextMenu = useId();

  // the time of the selected point is the start of the merge window
  const time: number = parseFloat(profile?.HistoryParams().merge_from);

  if (width === undefined || width == null) {
    width = 0;
  }

  const graphWidth = width - margin * 1.5 - margin / 2;

  const series: Series[] = data.reduce<Series[]>(function (agg: Series[], s: MetricsSeriesPb) {
    if (s.labelset !== undefined) {
      const metric = s.labelset.labels.sort((a, b) => a.name.localeCompare(b.name));
      agg.push({
        metric,
        values: s.samples.reduce<number[][]>(function (agg: number[][], d: MetricsSample) {
          if (d.timestamp !== undefined && d.valuePerSecond !== undefined) {
            const t = (Number(d.timestamp.seconds) * 1e9 + d.timestamp.nanos) / 1e6; // https://github.com/microsoft/TypeScript/issues/5710#issuecomment-157886246
            agg.push([t, d.valuePerSecond, Number(d.value), Number(d.duration)]);
          }
          return agg;
        }, []),
        labelset: metric.map(m => `${m.name}=${m.value}`).join(','),
      });
    }
    return agg;
  }, []);

  // Sort series by id to make sure the colors are consistent
  series.sort((a, b) => a.labelset.localeCompare(b.labelset));

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

  /* Scale */
  const xScale = d3.scaleUtc().domain([from, to]).range([0, graphWidth]);

  const yScale = d3
    .scaleLinear()
    // tslint:disable-next-line
    .domain([minY, maxY] as Iterable<d3.NumberValue>)
    .range([height - margin, 0])
    .nice();

  const color = d3.scaleOrdinal(d3.schemeCategory10);

  const l = d3.line(
    d => xScale(d[0]),
    d => yScale(d[1])
  );

  const highlighted = useMemo(() => {
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

  const openClosestProfile = (): void => {
    if (highlighted != null) {
      onSampleClick(
        Math.round(highlighted.timestamp),
        highlighted.value,
        sanitizeHighlightedValues(highlighted.labels), // When a user clicks on any sample in the graph, replace single `\` in the `labelValues` string with doubles `\\` if available.
        highlighted.duration
      );
    }
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
      openClosestProfile();
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

  const findSelectedProfile = (): HighlightedSeries | null => {
    if (profile == null) {
      return null;
    }

    let s: Series | null = null;
    let seriesIndex = -1;

    // if there are both query matchers and also a sumby value, we need to check if the sumby value is part of the query matchers.
    // if it is, then we should prioritize using the sumby label name and value to find the selected profile.
    const useSumBy =
      sumBy !== undefined &&
      sumBy.length > 0 &&
      profile.query.matchers.length > 0 &&
      profile.query.matchers.some(e => sumBy.includes(e.key));

    // get only the sumby keys and values from the profile query matchers
    const sumByMatchers =
      sumBy !== undefined ? profile.query.matchers.filter(e => sumBy.includes(e.key)) : [];

    const keysToMatch = useSumBy ? sumByMatchers : profile.query.matchers;

    outer: for (let i = 0; i < series.length; i++) {
      const keys = keysToMatch.map(e => e.key);
      for (let j = 0; j < keys.length; j++) {
        const matcherKey = keys[j];
        const label = series[i].metric.find(e => e.name === matcherKey);
        if (label === undefined) {
          continue outer; // label doesn't exist to begin with
        }
        if (keysToMatch[j].value !== label.value) {
          continue outer; // label values don't match
        }
      }
      seriesIndex = i;
      s = series[i];
    }

    if (s == null) {
      return null;
    }
    // Find the sample that matches the timestamp
    const sample = s.values.find(v => {
      return Math.round(v[0]) === time;
    });
    if (sample === undefined) {
      return null;
    }

    return {
      labels: [],
      seriesIndex,
      timestamp: sample[0],
      valuePerSecond: sample[1],
      value: sample[2],
      duration: sample[3],
      x: xScale(sample[0]),
      y: yScale(sample[1]),
    };
  };

  const selected = findSelectedProfile();

  const MENU_ID = `metrics-context-menu-${idForContextMenu}`;

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

  const trackVisibility = (isVisible: boolean): void => {
    setIsContextMenuOpen(isVisible);
  };

  const isDeltaType = profile !== null ? profile?.query.profType.delta : false;

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

  return (
    <>
      <MetricsContextMenu
        onAddLabelMatcher={addLabelMatcher}
        menuId={MENU_ID}
        highlighted={highlighted}
        trackVisibility={trackVisibility}
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
              highlighted={highlighted}
              contextElement={graph.current}
              sampleType={sampleType}
              sampleUnit={sampleUnit}
              delta={isDeltaType}
            />
          )}
        </div>
      )}
      <div
        ref={graph}
        onMouseEnter={function () {
          setHovering(true);
        }}
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
              {yScale.ticks(5).map((d, i, allTicks) => {
                let decimals = 2;
                const intervalBetweenTicks = allTicks[1] - allTicks[0];

                if (intervalBetweenTicks < 1) {
                  const precision = getPrecision(intervalBetweenTicks);
                  decimals = precision;
                }

                return (
                  <Fragment key={`${i.toString()}-${d.toString()}`}>
                    <g
                      key={`tick-${i}`}
                      className="tick"
                      /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                      transform={`translate(0, ${yScale(d)})`}
                    >
                      <line className="stroke-gray-300 dark:stroke-gray-500" x2={-6} />
                      <text fill="currentColor" x={-9} dy={'0.32em'}>
                        {valueFormatter(d, yAxisUnit, decimals)}
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
                  {yAxisLabel}
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
            <g className="lines fill-transparent">
              {series.map((s, i) => (
                <g key={i} className="line">
                  <MetricsSeries
                    data={s}
                    line={l}
                    color={color(i.toString())}
                    strokeWidth={
                      hovering && highlighted != null && i === highlighted.seriesIndex
                        ? lineStrokeHover
                        : lineStroke
                    }
                    xScale={xScale}
                    yScale={yScale}
                  />
                </g>
              ))}
            </g>
            {hovering && highlighted != null && (
              <g
                className="circle-group"
                ref={metricPointRef}
                style={{fill: color(highlighted.seriesIndex.toString())}}
              >
                <MetricsCircle cx={highlighted.x} cy={highlighted.y} />
              </g>
            )}
            {selected != null && (
              <g
                className="circle-group"
                style={
                  selected?.seriesIndex != null
                    ? {fill: color(selected.seriesIndex.toString())}
                    : {}
                }
              >
                <MetricsCircle cx={selected.x} cy={selected.y} radius={5} />
              </g>
            )}
          </g>
        </svg>
      </div>
    </>
  );
};
