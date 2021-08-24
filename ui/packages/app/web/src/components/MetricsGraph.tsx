import React, { useEffect, useRef, useState } from 'react'
import * as d3 from 'd3'
import moment from 'moment'
import MetricsSeries from './metrics/MetricsSeries'
import MetricsCircle from './metrics/MetricsCircle'
import { pointer } from 'd3-selection'
import { formatForTimespan } from '../libs/time'
import { nFormatter } from '../libs/unit'
import { Badge, Overlay, Popover } from 'react-bootstrap'
import { SingleProfileSelection, timeFormat } from '@parca/profile'
import { cutToMaxStringLength } from '../libs/utils'
import throttle from 'lodash.throttle'
import { CalcWidth } from '@parca/dynamicsize'
import { MetricsSeries as MetricsSeriesPb, MetricsSample, Label } from '@parca/client'

interface RawMetricsGraphProps {
  data: MetricsSeriesPb.AsObject[]
  from: number
  to: number
  profile: SingleProfileSelection | null
  onSampleClick: (timestamp: number, value: number, labels: Label.AsObject[]) => void
  onLabelClick: (labelName: string, labelValue: string) => void
  setTimeRange: (from: number, to: number) => void
  width?: number
}

interface HighlightedSeries {
  seriesIndex: number
  labels: Label.AsObject[]
  timestamp: number
  value: number
  x: number
  y: number
}

interface Series {
  metric: Label.AsObject[]
  values: number[][]
}

const MetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  onLabelClick,
  setTimeRange
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
)
export default MetricsGraph

export const parseValue = (value: string): number | null => {
  const val = parseFloat(value)
  // "+Inf", "-Inf", "+Inf" will be parsed into NaN by parseFloat(). They
  // can't be graphed, so show them as gaps (null).
  return isNaN(val) ? null : val
}

const lineStroke = '1px'
const lineStrokeHover = '2px'

const UpdatingPopover = React.forwardRef<any, any>(
  ({ popper, children, show: _, ...props }, ref) => {
    useEffect(() => {
      popper.scheduleUpdate()
    }, [children, popper])

    return (
      <Popover ref={ref} content {...props}>
        {children}
      </Popover>
    )
  }
)

export const RawMetricsGraph = ({
  data,
  from,
  to,
  profile,
  onSampleClick,
  onLabelClick,
  setTimeRange,
  width
}: RawMetricsGraphProps): JSX.Element => {
  const [dragging, setDragging] = useState(false)
  const [hovering, setHovering] = useState(false)
  const [selected, setSelected] = useState<HighlightedSeries | null>(null)
  const [relPos, setRelPos] = useState(-1)
  const [pos, setPos] = useState([0, 0])
  const metricPointRef = useRef(null)

  const time: number = profile?.HistoryParams().time

  if (width === undefined || width == null) {
    width = 0
  }

  const height = Math.min(width / 2.5, 400)
  const margin = 50
  const marginRight = 20

  const series: Series[] = data.reduce<Series[]>(function (agg: Series[], s: MetricsSeriesPb.AsObject) {
    if (s.labelset !== undefined) {
      agg.push({
        metric: s.labelset.labelsList,
        values: s.samplesList.reduce<number[][]>(function (agg: number[][], d: MetricsSample.AsObject) {
          if (d.timestamp !== undefined && d.value !== undefined) {
            const t = (d.timestamp.seconds * 1e9 + d.timestamp.nanos) / 1e6
            agg.push([t, d.value])
          }
          return agg
        }, [])
      })
    }
    return agg
  }, [])

  const extentsX = series.map(function (s) {
    return d3.extent(s.values, function (d) {
      return d[0]
    })
  })

  const minX = d3.min(extentsX, function (d) {
    return d[0]
  })
  const maxX = d3.max(extentsX, function (d) {
    return d[1]
  })

  const extentsY = series.map(function (s) {
    return d3.extent(s.values, function (d) {
      return d[1]
    })
  })

  const minY = d3.min(extentsY, function (d) {
    return d[0]
  })
  const maxY = d3.max(extentsY, function (d) {
    return d[1]
  })

  /* Scale */
  const xScale = d3
    .scaleUtc()
    .domain([minX, maxX])
    .range([0, width - margin - marginRight])

  const yScale = d3
    .scaleLinear()
    .domain([minY, maxY])
    .range([height - margin, 0])

  const color = d3.scaleOrdinal(d3.schemeCategory10)

  const l = d3.line(
    d => xScale(d[0]),
    d => yScale(d[1])
  )

  const getClosest = (): HighlightedSeries | null => {
    const closestPointPerSeries = series.map(function (s) {
      const distances = s.values.map(d => {
        const x = xScale(d[0])
        const y = yScale(d[1])

        return Math.sqrt(
          Math.pow(pos[0] - x, 2) +
          Math.pow(pos[1] - y, 2)
        )
      })

      const pointIndex = d3.minIndex(distances)
      const minDistance = distances[pointIndex]
      return {
        pointIndex: pointIndex,
        distance: minDistance
      }
    })

    const closestSeriesIndex = d3.minIndex(closestPointPerSeries, s => s.distance)
    const distance = closestPointPerSeries[closestSeriesIndex].distance
    if (distance > 15) {
      return null
    }

    const pointIndex = closestPointPerSeries[closestSeriesIndex].pointIndex
    const point = series[closestSeriesIndex].values[pointIndex]

    return {
      seriesIndex: closestSeriesIndex,
      labels: series[closestSeriesIndex].metric,
      timestamp: point[0],
      value: point[1],
      x: xScale(point[0]),
      y: yScale(point[1])
    }
  }

  const highlighted = getClosest()
  const nameLabel: Label.AsObject | undefined = highlighted?.labels.find((e) => e.name === '__name__')
  const highlightedNameLabel: Label.AsObject = nameLabel !== undefined ? (nameLabel) : ({ name: '', value: '' })

  const onMouseDown = (e): void => {
    // only left mouse button
    if (e.button !== 0) {
      return
    }

    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    const xCoordinate = rel[0]
    const xCoordinateWithoutMargin = xCoordinate - margin
    if (xCoordinateWithoutMargin >= 0) {
      setRelPos(xCoordinateWithoutMargin)
      setDragging(true)
    }

    e.stopPropagation()
    e.preventDefault()
  }

  const openClosestProfile = (): void => {
    if (highlighted != null) {
      setSelected({
        seriesIndex: highlighted.seriesIndex,
        labels: highlighted.labels,
        timestamp: highlighted.timestamp,
        value: highlighted.value,
        x: highlighted.x,
        y: highlighted.y
      })
      onSampleClick(
        highlighted.timestamp,
        highlighted.value,
        highlighted.labels
      )
    }
  }

  const onMouseUp = (e): void => {
    setDragging(false)

    if (relPos === -1) {
      // MouseDown happened outside of this element.
      return
    }

    // This is a normal click. We tolerate tiny movements to still be a
    // click as they can occur when clicking based on user feedback.
    if (Math.abs(relPos - pos[0]) <= 1) {
      openClosestProfile()
      setRelPos(-1)
      return
    }

    const firstTime = xScale.invert(relPos).valueOf()
    const secondTime = xScale.invert(pos[0]).valueOf()

    if (firstTime > secondTime) {
      setTimeRange(secondTime, firstTime)
    } else {
      setTimeRange(firstTime, secondTime)
    }
    setRelPos(-1)

    e.stopPropagation()
    e.preventDefault()
  }

  const throttledSetPos = throttle(setPos, 100)

  const onMouseMove = (e: React.MouseEvent<SVGSVGElement, MouseEvent>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    const xCoordinate = rel[0]
    const xCoordinateWithoutMargin = xCoordinate - margin
    const yCoordinate = rel[1]
    const yCoordinateWithoutMargin = yCoordinate - margin

    throttledSetPos([xCoordinateWithoutMargin, yCoordinateWithoutMargin])
  }

  const showTooltip = (): boolean => {
    if (highlighted == null) {
      return false
    }
    if (metricPointRef == null || metricPointRef.current == null) {
      return false
    }
    if (!hovering) {
      return false
    }
    // TODO Can probably be made even more understandable
    return !dragging || (dragging && Math.abs(relPos - pos[0]) <= 1)
  }

  useEffect(() => {
    if (profile == null) {
      return
    }

    let s: Series | null = null

    outer:
    for (let i = 0; i < series.length; i++) {
      const keys = Object.keys(profile.labels)
      for (let j = 0; j < keys.length; j++) {
        const key = keys[j]
        if (!(key in series[i].metric)) {
          continue outer // label doesn't exist to begin with
        }
        if (profile.labels[key] !== series[i].metric[key]) {
          continue outer // label values don't match
        }
      }
      s = series[i]
    }

    if (s == null) {
      return
    }
    // Find the sample that matches the timestamp
    const sample = s.values.find((v) => {
      return v[0] === time
    })
    if (sample === undefined) {
      return
    }

    setSelected({
      labels: [],
      seriesIndex: -1,
      timestamp: sample[0],
      value: sample[1],
      x: xScale(sample[0]),
      y: yScale(sample[1])
    })
  }, [time, width, profile, series, xScale, yScale])

  return (
    <div
      onMouseEnter={() => setHovering(true)}
      onMouseLeave={() => setHovering(false)}
    >
      <svg
        width={`${width}px`}
        height={`${height + margin}px`}
        onMouseDown={onMouseDown}
        onMouseUp={onMouseUp}
        onMouseMove={onMouseMove}
      >
        <g
          transform={`translate(${margin}, 0)`}
        >
          {dragging && (
            <g className="zoom-time-rect">
              <rect
                className="bar"
                x={(pos[0] - relPos) < 0 ? pos[0] : relPos}
                y={0}
                height={height}
                width={Math.abs(pos[0] - relPos)}
                fill={'rgba(0, 0, 0, 0.125)'}
              />
            </g>
          )}
        </g>
        <g
          transform={`translate(${margin}, ${margin})`}
        >
          <g className="lines">
            {series.map((s, i) => (
              <g key={i} className="line">
                <MetricsSeries
                  data={s}
                  line={l}
                  color={color(i)}
                  strokeWidth={(hovering && highlighted != null && i === highlighted.seriesIndex) ? lineStrokeHover : lineStroke}
                  xScale={xScale}
                  yScale={yScale}
                />
              </g>
            ))}
          </g>
          {hovering && highlighted != null && (
            <g className="circle-group" ref={metricPointRef} style={{ fill: color(highlighted.seriesIndex) }}>
              <MetricsCircle
                cx={highlighted.x}
                cy={highlighted.y}
              />
            </g>
          )}
          {selected != null
            ? (
            <g className="circle-group" style={{ fill: '#399' }}>
              <MetricsCircle
                cx={selected.x}
                cy={selected.y}
                radius={5}
              />
            </g>
              )
            : (
            <></>
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
                <line
                  y2={6}
                  stroke="currentColor"
                />
                <text
                  fill="currentColor"
                  dy=".71em"
                  y={9}
                >
                  {moment(d).utc().format(formatForTimespan(from, to))}
                </text>
              </g>
            ))}
          </g>
          <g
            className="y axis"
            textAnchor="end"
            fontSize="10"
            fill="none"
          >
            {yScale.ticks(3).map((d, i) => (
              <g
                key={i}
                className="tick"
                /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                transform={`translate(0, ${yScale(d)})`}
              >
                <line
                  stroke="currentColor"
                  x2={-6}
                />
                <text
                  fill="currentColor"
                  x={-9}
                  dy={'0.32em'}
                >
                  {nFormatter(d, 1)}
                </text>
              </g>
            ))}
          </g>
        </g>
      </svg>
      <Overlay
        show={showTooltip()}
        target={metricPointRef.current}
        placement="bottom"
      >
        <UpdatingPopover id="metrics-popover" style={{ opacity: '0.98' }}>
          {(showTooltip() && highlighted != null) && (
            <>
              <Popover.Title as="h3">
                <a>{highlightedNameLabel.value}</a>
              </Popover.Title>
              <Popover.Content>
                <p>Value: {nFormatter(highlighted.value, 1)}</p>
                <p>At: {moment(highlighted.timestamp).utc().format(timeFormat)}</p>
                  {highlighted.labels.filter((label: Label.AsObject) => (label.name !== '__name__')).map(function (label: Label.AsObject) {
                    return (
                      <React.Fragment key={label.name}>
                        <Badge
                          variant="light"
                          style={{
                            border: '1px solid rgba(0, 0, 0, 0.125)',
                            cursor: 'pointer'
                          }}
                          onClick={() => onLabelClick(label.name, label.value)}
                          /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                          title={`${label.name}="${label.value}"`.length > 37 ? `${label.name}="${label.value}"` : ''}
                        >
                          {/* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */}
                          {cutToMaxStringLength(`${label.name}="${label.value}"`, 37)}
                        </Badge>{'  '}
                      </React.Fragment>
                    )
                  })}
              </Popover.Content>
            </>
          )}
        </UpdatingPopover>
      </Overlay>
    </div>
  )
}
