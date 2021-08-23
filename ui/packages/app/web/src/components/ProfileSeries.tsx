import { MouseEvent, useRef, useState } from 'react'
import { Badge, Card, Col, Row } from 'react-bootstrap'
import { ProfileSelection, SingleProfileSelection } from '@parca/profile'
import { pointer } from 'd3-selection'
import binarySearchClosest from '../libs/binary-search-closest'
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome'
import { faCaretDown, faCaretUp } from '@fortawesome/free-solid-svg-icons'

interface ProfileSeriesProps {
  profileSeries: any
  xScale: any
  height: number
  width: number
  marginLeft: number
  marginTop: number
  select: (source: ProfileSelection) => void
  setHighlightedTimestampIndex: (timestamp: number) => void
  highlightedSeries: boolean
  highlightedTimestampIndex: number
  color: string
  setTooltip: (e: Element, timestamp: number, labels: { [key: string]: string }) => void
  setShowTooltip: (on: boolean) => void
  addLabelMatcher: (key: string, value: string) => void
  setTimeRange: (from: number, to: number) => void
}

export default ({
  profileSeries,
  xScale,
  height,
  width,
  marginLeft,
  marginTop,
  select,
  setHighlightedTimestampIndex,
  highlightedSeries,
  highlightedTimestampIndex,
  color,
  setTooltip,
  setShowTooltip,
  addLabelMatcher,
  setTimeRange
}: ProfileSeriesProps): JSX.Element => {
  const [expandLabels, setExpandLabels] = useState(false)
  const [dragging, setDragging] = useState(false)
  const [relPos, setRelPos] = useState(0)
  const [pos, setPos] = useState(0)
  const svgGroupRef = useRef(null)

  const closestProfile = (e: any): number => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    const xCoordinate = rel[0]
    const xCoordinateWithoutMargin = xCoordinate - marginLeft
    const xTime = xScale.invert(xCoordinateWithoutMargin)
    const timestamps = profileSeries.timestamps
    const closestTimestampIndex = binarySearchClosest(timestamps, xTime.valueOf())

    return closestTimestampIndex
  }

  const selectClosestProfile = (e: any): void => {
    const closestTimestampIndex = closestProfile(e)
    const timestamp = profileSeries.timestamps[closestTimestampIndex]

    select(new SingleProfileSelection(profileSeries.labels, timestamp))
  }

  const highlightClosestProfile = (e: MouseEvent): void => {
    const closestTimestampIndex = closestProfile(e)
    setHighlightedTimestampIndex(closestTimestampIndex)
    if (svgGroupRef?.current != null) {
      // @ts-expect-error
      const el = svgGroupRef.current.childNodes[closestTimestampIndex]
      setTooltip(el, profileSeries.timestamps[closestTimestampIndex], profileSeries.labels)
      setShowTooltip(true)
    }
  }

  const onMouseDown = (e): void => {
    e.stopPropagation()
    e.preventDefault()

    // only left mouse button
    if (e.button !== 0) {
      return
    }

    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    const xCoordinate = rel[0]
    const xCoordinateWithoutMargin = xCoordinate - marginLeft
    setRelPos(xCoordinateWithoutMargin)
    setPos(xCoordinateWithoutMargin)
    setDragging(true)
  }

  const onMouseUp = (e): void => {
    e.stopPropagation()
    e.preventDefault()

    setDragging(false)

    // This is a normal click.
    if (relPos === pos) {
      selectClosestProfile(e)
      return
    }

    const fromTime = xScale.invert(relPos).valueOf()
    const toTime = xScale.invert(pos).valueOf()

    setTimeRange(fromTime, toTime)
  }

  const onMouseMove = (e: MouseEvent): void => {
    e.stopPropagation()
    e.preventDefault()

    if (!dragging) {
      highlightClosestProfile(e)
      return
    }

    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    const xCoordinate = rel[0]
    const xCoordinateWithoutMargin = xCoordinate - marginLeft

    setPos(xCoordinateWithoutMargin)
  }

  const displayLabels = expandLabels
    ? (
        Object.keys(profileSeries.labels).filter(key => (key !== '__name__' && key !== 'polarsignals_project_id'))
      )
    : (
        Object.keys(profileSeries.labels).filter(key => (key !== '__name__' && key !== 'polarsignals_project_id')).slice(0, 2)
      )

  return (
    <Row style={{ paddingTop: 10, paddingBottom: 10 }}>
      <Col xs="2">
        {displayLabels.map((key: string) => (
          <div key={key}>
            <Badge
              variant="light"
              onClick={() => addLabelMatcher(key, profileSeries.labels[key])}
              style={{ border: '1px solid rgba(0, 0, 0, 0.125)', color: '#505050', cursor: 'pointer' }}
            >
              {key}={profileSeries.labels[key]}
            </Badge>{'  '}
          </div>
        ))}
        <b>
          <a style={{ fontSize: 10, marginLeft: 5, color: '#505050', cursor: 'pointer' }}
            onClick={() => setExpandLabels(!expandLabels)}>
            {expandLabels
              ? (
              <><FontAwesomeIcon style={{ marginTop: -2 }} width={10} icon={faCaretDown}/>{' '}COLLAPSE LABELS</>
                )
              : (
              <><FontAwesomeIcon style={{ marginTop: -2 }} width={10} icon={faCaretUp}/>{' '}EXPAND LABELS</>
                )}
          </a>
        </b>
      </Col>
      <Col xs="8">
        <Card style={{ padding: 10 }}>
          <svg
            width={'100%'}
            height={height}
            onMouseDown={(e: MouseEvent<SVGSVGElement>) => onMouseDown(e)}
            onMouseUp={(e: MouseEvent<SVGSVGElement>) => onMouseUp(e)}
            onMouseMove={(e: MouseEvent<SVGSVGElement>) => onMouseMove(e)}
            onMouseOut={() => setShowTooltip(false)}
            viewBox={'0 0 ' + width.toString() + ' ' + height.toString()}
            preserveAspectRatio="xMidYMid meet"
          >
            <g ref={svgGroupRef} transform={`translate(${marginLeft},${marginTop})`}>
              {dragging && (
                <rect
                  className="bar"
                  x={relPos}
                  y={0}
                  height={height}
                  width={pos - relPos}
                  fill={'rgba(0, 0, 0, 0.125)'}
                />
              )}
              {profileSeries.timestamps.map((timestamp, j) => (
                <rect
                  key={j}
                  className="bar"
                  x={xScale(new Date(timestamp))}
                  y={0}
                  height={height}
                  width={(highlightedSeries && highlightedTimestampIndex === j) ? '2' : '1'}
                  fill={color}
                />
              ))}
            </g>
          </svg>
        </Card>
      </Col>
    </Row>
  )
}
