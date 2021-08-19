import React, { useState, useEffect } from 'react'
import moment from 'moment'
import MetricsGraph from './MetricsGraph'
import { ProfileSelection, SingleProfileSelection } from '@parca/profile'
import { Alert, Card, Col, Row, Spinner } from 'react-bootstrap'
import { QueryRangeRequest, QueryRangeResponse, Label, QueryClient, ServiceError } from '@parca/client'
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb'

interface ProfileMetricsGraphProps {
  queryClient: QueryClient
  queryExpression: string
  profile: ProfileSelection | null
  from: number
  to: number
  select: (source: ProfileSelection) => void
  setTimeRange: (from: number, to: number) => void
  addLabelMatcher: (key: string, value: string) => void
}

export interface IQueryRangeResult {
  response: QueryRangeResponse.AsObject|null
  error: ServiceError|null
}

export const useQueryRange = (
  client: QueryClient,
  queryExpression: string,
  start: number,
  end: number
): IQueryRangeResult => {
  const [result, setResult] = useState<IQueryRangeResult>({
    response: null,
    error: null
  })

  useEffect(() => {
    const req = new QueryRangeRequest()
    req.setQuery(queryExpression)

    const startTimestamp = new Timestamp()
    startTimestamp.fromDate(moment(start).toDate())
    req.setStart(startTimestamp)

    const endTimestamp = new Timestamp()
    endTimestamp.fromDate(moment(end).toDate())
    req.setEnd(endTimestamp)

    client.queryRange(
      req,
      (error: ServiceError|null, responseMessage: QueryRangeResponse|null) => {
        const res = responseMessage == null ? null : responseMessage.toObject()

        setResult({
          response: res,
          error: error
        })
      }
    )
  }, [client, queryExpression, start, end])

  return result
}

// TODO(kakkoyun): !!
// const minStep = 15

const ProfileMetricsGraph = ({
  queryClient,
  queryExpression,
  profile,
  from,
  to,
  select,
  setTimeRange,
  addLabelMatcher
}: ProfileMetricsGraphProps): JSX.Element => {
  // const timeRangeSeconds = (to - from) / 1000

  // The SVG has 1200px width and we want 1 datapoint every 10px.
  // const calculatedStep = timeRangeSeconds / 120
  // const step = calculatedStep < minStep ? minStep : calculatedStep

  // TODO(kakkoyun): !!
  // `${queryRangeEndpoint}?query=${encodeURIComponent(queryExpression)}&start=${from / 1000}&end=${to / 1000}&step=${step}&dedup=true`, fetchJSON
  const { response, error } = useQueryRange(
    queryClient,
    queryExpression,
    from,
    to
  )

  if (error != null) {
    return (
      <Row>
        <Col xs='2'></Col>
        <Col xs='8'>
          <Alert variant='danger'>{error.message}</Alert>
        </Col>
      </Row>
    )
  }
  if (response == null) {
    return (
      <Row>
        <Col xs='2'></Col>
        <Col xs='8'>
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: 'inherit',
              marginTop: 100
            }}
          >
            <Spinner animation='border' role='status'>
              <span className='sr-only'>Loading...</span>
            </Spinner>
          </div>
        </Col>
      </Row>
    )
  }

  const series = response.seriesList
  if (series == null || series.length === 0) {
    return (
      <Row>
        <Col xs='2'></Col>
        <Col xs='8'>
          <div style={{ textAlign: 'center', paddingTop: 100 }}>
            <p>
              {/* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */}
              No data found. Try a different query.
            </p>
          </div>
        </Col>
      </Row>
    )
  }

  const handleSampleClick = (
    timestamp: number,
    value: number,
    labels: Label.AsObject[]
  ): void => {
    select(new SingleProfileSelection(labels, timestamp))
  }

  return (
    <Card>
      <MetricsGraph
        data={series}
        from={from}
        to={to}
        profile={profile as SingleProfileSelection}
        setTimeRange={setTimeRange}
        onSampleClick={handleSampleClick}
        onLabelClick={addLabelMatcher}
        width={0}
      />
    </Card>
  )
}

export default ProfileMetricsGraph
