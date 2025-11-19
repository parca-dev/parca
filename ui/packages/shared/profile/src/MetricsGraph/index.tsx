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

import {DateTimeRange, useParcaContext} from '@parca/components';
import {TEST_IDS, testId} from '@parca/test-utils';
import {formatDate, formatForTimespan, getPrecision, valueFormatter} from '@parca/utilities';

import MetricsCircle from '../MetricsCircle';
import MetricsSeries from '../MetricsSeries';
import MetricsContextMenu, {
  ContextMenuItem,
  ContextMenuItemOrSubmenu,
  ContextMenuSubmenu,
} from './MetricsContextMenu';
import MetricsInfoPanel from './MetricsInfoPanel';
import MetricsTooltip from './MetricsTooltip';

interface Props {
  data: Series[];
  from: number;
  to: number;
  onSampleClick: (closestPoint: SeriesPoint) => void;
  setTimeRange: (range: DateTimeRange) => void;
  yAxisLabel: string;
  yAxisUnit: string;
  width?: number;
  height?: number;
  margin?: number;
  selectedPoint?: SeriesPoint | null;
  contextMenuItems?: ContextMenuItemOrSubmenu[];
  renderTooltipContent?: (seriesIndex: number, pointIndex: number) => React.ReactNode;
}

export interface SeriesPoint {
  seriesIndex: number;
  pointIndex: number;
}

export interface HighlightedSeries {
  seriesIndex: number;
  pointIndex: number;
  x: number;
  y: number;
}

export interface Series {
  id: string; // opaque string used to determine line color
  values: Array<[number, number]>; // [timestamp_ms, value]
  highlighted?: boolean;
}

const MetricsGraph = ({
  data,
  from,
  to,
  onSampleClick,
  setTimeRange,
  yAxisLabel,
  yAxisUnit,
  width = 0,
  height = 0,
  margin = 0,
  selectedPoint,
  contextMenuItems,
  renderTooltipContent,
}: Props): JSX.Element => {
  const [isInfoPanelOpen, setIsInfoPanelOpen] = useState<boolean>(false);
  return (
    <div
      className="relative"
      {...testId(TEST_IDS.METRICS_GRAPH)}
      onClick={() => isInfoPanelOpen && setIsInfoPanelOpen(false)}
    >
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
        onSampleClick={onSampleClick}
        setTimeRange={setTimeRange}
        yAxisLabel={yAxisLabel}
        yAxisUnit={yAxisUnit}
        width={width}
        height={height}
        margin={margin}
        selectedPoint={selectedPoint}
        contextMenuItems={contextMenuItems}
        renderTooltipContent={renderTooltipContent}
      />
    </div>
  );
};

export default MetricsGraph;
export type {ContextMenuItemOrSubmenu, ContextMenuItem, ContextMenuSubmenu};

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
  onSampleClick,
  setTimeRange,
  yAxisLabel,
  yAxisUnit,
  width,
  height = 50,
  margin = 0,
  selectedPoint,
  contextMenuItems,
  renderTooltipContent,
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

  if (width === undefined || width == null) {
    width = 0;
  }

  const graphWidth = useMemo(() => width - margin * 1.5 - margin / 2, [width, margin]);
  const graphTransform = useMemo(() => {
    // Adds 6px padding which aligns the graph on the grid
    return `translate(6, 0) scale(${(graphWidth - 6) / graphWidth}, 1)`;
  }, [graphWidth]);

  const series = data;

  const extentsY = series.map(function (s) {
    return d3.extent(s.values, function (d) {
      return d[1]; // d[1] is the value
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

  // Create deterministic color mapping based on series IDs
  const color = useMemo(() => {
    const scale = d3.scaleOrdinal(d3.schemeCategory10);
    // Pre-populate the scale with sorted series IDs to ensure consistent colors
    const sortedIds = [...new Set(series.map(s => s.id))].sort();
    sortedIds.forEach(id => scale(id));
    return scale;
  }, [series]);

  const l = d3.line<[number, number]>(
    d => xScale(d[0]),
    d => yScale(d[1])
  );

  const closestPoint = useMemo(() => {
    // Guard against empty series
    if (series.length === 0) {
      return null;
    }

    const closestPointPerSeries = series.map(function (s) {
      const distances = s.values.map(d => {
        const x = xScale(d[0]) + margin / 2; // d[0] is timestamp_ms
        const y = yScale(d[1]) - margin / 3; // d[1] is value

        // Cartesian distance from the mouse position to the point
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
    return {
      seriesIndex: closestSeriesIndex,
      pointIndex,
    };
  }, [pos, series, xScale, yScale, margin]);

  const highlighted = useMemo(() => {
    if (series.length === 0 || closestPoint == null) {
      return null;
    }

    const point = series[closestPoint.seriesIndex].values[closestPoint.pointIndex];
    return {
      seriesIndex: closestPoint.seriesIndex,
      pointIndex: closestPoint.pointIndex,
      x: xScale(point[0]),
      y: yScale(point[1]),
    };
  }, [closestPoint, series, xScale, yScale]);

  const selected = useMemo(() => {
    if (series.length === 0 || selectedPoint == null) {
      return null;
    }

    const point = series[selectedPoint.seriesIndex].values[selectedPoint.pointIndex];
    return {
      seriesIndex: selectedPoint.seriesIndex,
      pointIndex: selectedPoint.pointIndex,
      x: xScale(point[0]),
      y: yScale(point[1]),
    };
  }, [selectedPoint, series, xScale, yScale]);

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

  const handleClosestPointClick = (): void => {
    if (closestPoint != null) {
      onSampleClick(closestPoint);
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
      handleClosestPointClick();
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

  return (
    <>
      {contextMenuItems != null && (
        <MetricsContextMenu
          menuId={MENU_ID}
          closestPoint={closestPoint}
          series={series}
          trackVisibility={trackVisibility}
          menuItems={contextMenuItems}
        />
      )}
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
              contextElement={graph.current}
              content={renderTooltipContent?.(highlighted.seriesIndex, highlighted.pointIndex)}
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
            <g
              className="lines fill-transparent"
              transform={graphTransform}
              width={graphWidth - 100}
            >
              {series.map((s, i) => (
                <g key={s.id} className="line">
                  <MetricsSeries
                    data={s}
                    line={l}
                    color={color(s.id)}
                    strokeWidth={
                      (hovering && highlighted != null && i === highlighted.seriesIndex) ||
                      s.highlighted === true
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
                style={{fill: color(series[highlighted.seriesIndex]?.id ?? '0')}}
                transform={graphTransform}
              >
                <MetricsCircle cx={highlighted.x} cy={highlighted.y} />
              </g>
            )}
            {selected != null && (
              <g
                className="circle-group"
                style={
                  selected?.seriesIndex != null
                    ? {fill: color(series[selected.seriesIndex]?.id ?? '0')}
                    : {}
                }
                transform={graphTransform}
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
