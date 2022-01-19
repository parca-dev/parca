import React, {useEffect, useRef, useState} from 'react';
import * as d3 from 'd3';
import moment from 'moment';
import MetricsSeries from './metrics/MetricsSeries';
import MetricsCircle from './metrics/MetricsCircle';
import {pointer} from 'd3-selection';
import {formatForTimespan} from '../libs/time';
import {SingleProfileSelection, timeFormat} from '@parca/profile';
import {cutToMaxStringLength} from '../libs/utils';
import throttle from 'lodash.throttle';
import {CalcWidth} from '@parca/dynamicsize';
import {MetricsSeries as MetricsSeriesPb, MetricsSample, Label} from '@parca/client';
import {usePopper} from 'react-popper';
import type {VirtualElement} from '@popperjs/core';
import {valueFormatter} from '@parca/functions';

interface RawMetricsGraphProps {
  data: MetricsSeriesPb.AsObject[];
  from: number;
  to: number;
  profile: SingleProfileSelection | null;
  onSampleClick: (timestamp: number, value: number, labels: Label.AsObject[]) => void;
  onLabelClick: (labelName: string, labelValue: string) => void;
  setTimeRange: (from: number, to: number) => void;
  width?: number;
}

interface HighlightedSeries {
  seriesIndex: number;
  labels: Label.AsObject[];
  timestamp: number;
  value: number;
  x: number;
  y: number;
}

interface Series {
  metric: Label.AsObject[];
  values: number[][];
}

const MetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  onLabelClick,
  setTimeRange,
}: RawMetricsGraphProps): JSX.Element => (
  <CalcWidth throttle={300} delay={2000}>
    <RawMetricsGraph
      data={data}
      from={from}
      to={to}
      profile={profile}
      onSampleClick={onSampleClick}
      onLabelClick={onLabelClick}
      setTimeRange={setTimeRange}
    />
  </CalcWidth>
);
export default MetricsGraph;

export const parseValue = (value: string): number | null => {
  const val = parseFloat(value);
  // "+Inf", "-Inf", "+Inf" will be parsed into NaN by parseFloat(). They
  // can't be graphed, so show them as gaps (null).
  return isNaN(val) ? null : val;
};

const lineStroke = '1px';
const lineStrokeHover = '2px';

interface MetricsTooltipProps {
  x: number;
  y: number;
  highlighted: HighlightedSeries;
  onLabelClick: (labelName: string, labelValue: string) => void;
  contextElement: Element | null;
  sampleUnit: string;
}

function generateGetBoundingClientRect(contextElement: Element, x = 0, y = 0) {
  const domRect = contextElement.getBoundingClientRect();
  return () =>
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    ({
      width: 0,
      height: 0,
      top: domRect.y + y,
      left: domRect.x + x,
      right: domRect.x + x,
      bottom: domRect.y + y,
    } as ClientRect);
}

const virtualElement: VirtualElement = {
  getBoundingClientRect: () => {
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    return {
      width: 0,
      height: 0,
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
    } as ClientRect;
  },
};

export const MetricsTooltip = ({
  x,
  y,
  highlighted,
  onLabelClick,
  contextElement,
  sampleUnit,
}: MetricsTooltipProps): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(virtualElement, popperElement, {
    placement: 'auto-start',
    strategy: 'absolute',
    modifiers: [
      {
        name: 'preventOverflow',
        options: {
          tether: false,
          altAxis: true,
        },
      },
      {
        name: 'offset',
        options: {
          offset: [30, 30],
        },
      },
    ],
  });

  const update = popperProps.update;

  useEffect(() => {
    if (contextElement != null) {
      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(contextElement, x, y);
      update?.();
    }
  }, [x, y, contextElement, update]);

  const nameLabel: Label.AsObject | undefined = highlighted?.labels.find(
    e => e.name === '__name__'
  );
  const highlightedNameLabel: Label.AsObject =
    nameLabel !== undefined ? nameLabel : {name: '', value: ''};

  return (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper} className="z-10">
      <div className="flex max-w-md">
        <div className="m-auto">
          <div
            className="border-gray-300 dark:border-gray-500 bg-gray-50 dark:bg-gray-900 rounded-lg p-3 shadow-lg opacity-90"
            style={{borderWidth: 1}}
          >
            <div className="flex flex-row">
              <div className="ml-2 mr-6">
                <span className="font-semibold">{highlightedNameLabel.value}</span>
                <span className="block text-gray-700 dark:text-gray-300 my-2">
                  <table className="table-auto">
                    <tbody>
                      <tr>
                        <td className="w-1/4">Value</td>
                        <td className="w-3/4">
                          {valueFormatter(highlighted.value, sampleUnit, 1)}
                        </td>
                      </tr>
                      <tr>
                        <td className="w-1/4">At</td>
                        <td className="w-3/4">
                          {moment(highlighted.timestamp).utc().format(timeFormat)}
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </span>
                <span className="block text-gray-500 my-2">
                  {highlighted.labels
                    .filter((label: Label.AsObject) => label.name !== '__name__')
                    .map(function (label: Label.AsObject) {
                      return (
                        <button
                          key={label.name}
                          type="button"
                          className="inline-block rounded-lg text-gray-700 bg-gray-200 dark:bg-gray-700 dark:text-gray-400 px-2 py-1 text-xs font-bold mr-3"
                          onClick={() => onLabelClick(label.name, label.value)}
                        >
                          {cutToMaxStringLength(`${label.name}="${label.value}"`, 37)}
                        </button>
                      );
                    })}
                </span>
                <span className="block text-gray-500 text-xs">
                  Hold shift and click label to add to query.
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export const RawMetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  onLabelClick,
  setTimeRange,
  width,
}: RawMetricsGraphProps): JSX.Element => {
  const graph = useRef(null);
  const [dragging, setDragging] = useState(false);
  const [hovering, setHovering] = useState(false);
  const [relPos, setRelPos] = useState(-1);
  const [pos, setPos] = useState([0, 0]);
  const [freezeTooltip, setFreezeTooltip] = useState(false);
  const metricPointRef = useRef(null);

  useEffect(() => {
    const handleShiftDown = event => {
      if (event.keyCode === 16) {
        setFreezeTooltip(true);
      }
    };
    window.addEventListener('keydown', handleShiftDown);

    return () => {
      window.removeEventListener('keydown', handleShiftDown);
    };
  }, []);

  useEffect(() => {
    const handleShiftUp = event => {
      if (event.keyCode === 16) {
        setFreezeTooltip(false);
      }
    };
    window.addEventListener('keyup', handleShiftUp);

    return () => {
      window.removeEventListener('keyup', handleShiftUp);
    };
  }, []);

  const time: number = parseFloat(profile?.HistoryParams().time);

  if (width === undefined || width == null) {
    width = 0;
  }

  const height = Math.min(width / 2.5, 400);
  const margin = 50;
  const marginRight = 20;
  const sampleUnit = data[0].sampleType !== undefined ? data[0].sampleType.unit : '';

  const series: Series[] = data.reduce<Series[]>(function (
    agg: Series[],
    s: MetricsSeriesPb.AsObject
  ) {
    if (s.labelset !== undefined) {
      agg.push({
        metric: s.labelset.labelsList,
        values: s.samplesList.reduce<number[][]>(function (
          agg: number[][],
          d: MetricsSample.AsObject
        ) {
          if (d.timestamp !== undefined && d.value !== undefined) {
            const t = (d.timestamp.seconds * 1e9 + d.timestamp.nanos) / 1e6;
            agg.push([t, d.value]);
          }
          return agg;
        },
        []),
      });
    }
    return agg;
  },
  []);

  const extentsX = series.map(function (s) {
    return d3.extent(s.values, function (d) {
      return d[0];
    });
  });

  const minX = d3.min(extentsX, function (d) {
    return d[0];
  });
  const maxX = d3.max(extentsX, function (d) {
    return d[1];
  });

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
    .domain([minX, maxX])
    .range([0, width - margin - marginRight]);

  const yScale = d3
    .scaleLinear()
    .domain([minY, maxY])
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
        pointIndex: pointIndex,
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
      value: point[1],
      x: xScale(point[0]),
      y: yScale(point[1]),
    };
  };

  const highlighted = getClosest();

  const onMouseDown = (e): void => {
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
      onSampleClick(Math.round(highlighted.timestamp), highlighted.value, highlighted.labels);
    }
  };

  const onMouseUp = (e): void => {
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
      setTimeRange(secondTime, firstTime);
    } else {
      setTimeRange(firstTime, secondTime);
    }
    setRelPos(-1);

    e.stopPropagation();
    e.preventDefault();
  };

  const throttledSetPos = throttle(setPos, 20);

  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement, MouseEvent>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    const xCoordinate = rel[0];
    const xCoordinateWithoutMargin = xCoordinate - margin;
    const yCoordinate = rel[1];
    const yCoordinateWithoutMargin = yCoordinate - margin;

    if (!freezeTooltip) {
      throttledSetPos([xCoordinateWithoutMargin, yCoordinateWithoutMargin]);
    }
  };

  const findSelectedProfile = () => {
    if (profile == null) {
      return null;
    }

    let s: Series | null = null;
    let seriesIndex: number | null = null;

    outer: for (let i = 0; i < series.length; i++) {
      const keys = profile.labels.map(e => e.name);
      for (let j = 0; j < keys.length; j++) {
        const labelName = keys[j];
        const label = series[i].metric.find(e => e.name === labelName);
        if (label === undefined) {
          continue outer; // label doesn't exist to begin with
        }
        if (profile.labels[j].value !== label.value) {
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
      seriesIndex: seriesIndex,
      timestamp: sample[0],
      value: sample[1],
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
          />
        </div>
      )}
      <div
        ref={graph}
        onMouseEnter={function () {
          setHovering(true);
          setFreezeTooltip(false);
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
            <g className="lines">
              {series.map((s, i) => (
                <g key={i} className="line">
                  <MetricsSeries
                    data={s}
                    line={l}
                    color={color(i)}
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
                style={{fill: color(highlighted.seriesIndex)}}
              >
                <MetricsCircle cx={highlighted.x} cy={highlighted.y} />
              </g>
            )}
            {selected != null && (
              <g className="circle-group" style={{fill: color(selected.seriesIndex)}}>
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
                    {moment(d).utc().format(formatForTimespan(from, to))}
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
