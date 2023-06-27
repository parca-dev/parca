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

import React, {useRef, useState} from 'react';

import * as d3 from 'd3';
import {pointer} from 'd3-selection';
import throttle from 'lodash.throttle';

import {Label, MetricsSample, MetricsSeries as MetricsSeriesPb} from '@parca/client';
import {DateTimeRange, useKeyDown} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {
  formatDate,
  formatForTimespan,
  sanitizeHighlightedValues,
  valueFormatter,
} from '@parca/utilities';

import {MergedProfileSelection} from '..';
import MetricsCircle from '../MetricsCircle';
import MetricsSeries from '../MetricsSeries';
import MetricsTooltip from './MetricsTooltip';

interface Props {
  data: MetricsSeriesPb[];
  from: number;
  to: number;
  profile: MergedProfileSelection | null;
  onSampleClick: (timestamp: number, value: number, labels: Label[]) => void;
  onLabelClick: (labelName: string, labelValue: string) => void;
  setTimeRange: (range: DateTimeRange) => void;
  sampleUnit: string;
  width?: number;
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

interface Series {
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
  onLabelClick,
  setTimeRange,
  sampleUnit,
}: Props): JSX.Element => {
  const {ref, dimensions} = useContainerDimensions();

  return (
    <div ref={ref}>
      <RawMetricsGraph
        data={data}
        from={from}
        to={to}
        profile={profile}
        onSampleClick={onSampleClick}
        onLabelClick={onLabelClick}
        setTimeRange={setTimeRange}
        sampleUnit={sampleUnit}
        width={dimensions?.width}
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
  onLabelClick,
  setTimeRange,
  width,
  sampleUnit,
}: Props): JSX.Element => {
  const graph = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [hovering, setHovering] = useState(false);
  const [relPos, setRelPos] = useState(-1);
  const [pos, setPos] = useState([0, 0]);
  const metricPointRef = useRef(null);
  const {isShiftDown} = useKeyDown();

  // the time of the selected point is the start of the merge window
  const time: number = parseFloat(profile?.HistoryParams().merge_from);

  if (width === undefined || width == null) {
    width = 0;
  }

  const height = Math.min(width / 2.5, 400);
  const margin = 50;
  const marginRight = 20;

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
  const xScale = d3
    .scaleUtc()
    .domain([from, to])
    .range([0, width - margin - marginRight]);

  const yScale = d3
    .scaleLinear()
    // tslint:disable-next-line
    .domain([minY, maxY] as Iterable<d3.NumberValue>)
    .range([height - margin, 0]);

  const color = d3.scaleOrdinal(d3.schemeCategory10);

  const l = d3.line(
    d => xScale(d[0]),
    d => yScale(d[1])
  );

  const getClosest = (): HighlightedSeries | null => {
    const closestPointPerSeries = series.map(function (s) {
      const distances = s.values.map(d => {
        const x = xScale(d[0]);
        const y = yScale(d[1]);

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
  };

  const highlighted = getClosest();

  const onMouseDown = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    // if shift is down, disable mouse behavior
    if (isShiftDown) {
      return;
    }

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
        sanitizeHighlightedValues(highlighted.labels) // When a user clicks on any sample in the graph, replace single `\` in the `labelValues` string with doubles `\\` if available.
      );
    }
  };

  const onMouseUp = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    if (isShiftDown) {
      return;
    }

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

    const firstTime = xScale.invert(relPos).valueOf();
    const secondTime = xScale.invert(pos[0]).valueOf();

    if (firstTime > secondTime) {
      setTimeRange(DateTimeRange.fromAbsoluteDates(secondTime, firstTime));
    } else {
      setTimeRange(DateTimeRange.fromAbsoluteDates(firstTime, secondTime));
    }
    setRelPos(-1);

    e.stopPropagation();
    e.preventDefault();
  };

  const throttledSetPos = throttle(setPos, 20);

  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    // do not update position if shift is down because this means the user is locking the tooltip
    if (isShiftDown) {
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

    outer: for (let i = 0; i < series.length; i++) {
      const keys = profile.query.matchers.map(e => e.key);
      for (let j = 0; j < keys.length; j++) {
        const matcherKey = keys[j];
        const label = series[i].metric.find(e => e.name === matcherKey);
        if (label === undefined) {
          continue outer; // label doesn't exist to begin with
        }
        if (profile.query.matchers[j].value !== label.value) {
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
  return (
    <>
      {highlighted != null && hovering && !dragging && pos[0] !== 0 && pos[1] !== 0 && (
        <div
          onMouseMove={onMouseMove}
          onMouseEnter={() => setHovering(true)}
          onMouseLeave={() => setHovering(false)}
        >
          <MetricsTooltip
            x={pos[0] + margin}
            y={pos[1] + margin}
            highlighted={highlighted}
            onLabelClick={onLabelClick}
            contextElement={graph.current}
            sampleUnit={sampleUnit}
            delta={profile !== null ? profile?.query.profType.delta : false}
          />
        </div>
      )}
      <div
        ref={graph}
        onMouseEnter={function () {
          setHovering(true);
        }}
        onMouseLeave={() => setHovering(false)}
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
          <g transform={`translate(${margin}, ${margin})`}>
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
            <g
              className="x axis"
              fill="none"
              fontSize="10"
              textAnchor="middle"
              transform={`translate(0,${height - margin})`}
            >
              {xScale.ticks(5).map((d, i) => (
                <g
                  key={i}
                  className="tick"
                  /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                  transform={`translate(${xScale(d)}, 0)`}
                >
                  <line y2={6} stroke="currentColor" />
                  <text fill="currentColor" dy=".71em" y={9}>
                    {formatDate(d, formatForTimespan(from, to))}
                  </text>
                </g>
              ))}
            </g>
            <g className="y axis" textAnchor="end" fontSize="10" fill="none">
              {yScale.ticks(3).map((d, i) => (
                <g
                  key={i}
                  className="tick"
                  /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                  transform={`translate(0, ${yScale(d)})`}
                >
                  <line stroke="currentColor" x2={-6} />
                  <text fill="currentColor" x={-9} dy={'0.32em'}>
                    {valueFormatter(d, sampleUnit, 1)}
                  </text>
                </g>
              ))}
            </g>
          </g>
        </svg>
      </div>
    </>
  );
};
