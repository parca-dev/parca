import React, { useState, useEffect } from 'react'
import moment from 'moment'
import MetricsGraph from './MetricsGraph'
import { ProfileSelection, SingleProfileSelection } from '@parca/profile'
import { Alert, Card, Col, Row, Spinner } from 'react-bootstrap'
import {
  QueryRangeRequest,
  QueryRangeResponse,
  Label,
  QueryServiceClient,
  ServiceError
} from '@parca/client'
import { Timestamp } from 'google-protobuf/google/protobuf/timestamp_pb'

interface ProfileMetricsGraphProps {
  queryClient: QueryServiceClient
  queryExpression: string
  profile: ProfileSelection | null
  from: number
  to: number
  select: (source: ProfileSelection) => void
  setTimeRange: (from: number, to: number) => void
  addLabelMatcher: (key: string, value: string) => void
}

export interface IQueryRangeResult {
  response: QueryRangeResponse.AsObject | null
  error: ServiceError | null
}

export const useQueryRange = (
  client: QueryServiceClient,
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
      (error: ServiceError | null, responseMessage: QueryRangeResponse | null) => {
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
  const { response, error } = useQueryRange(queryClient, queryExpression, from, to)

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
          <div className='py-20 flex justify-center'>
            <p className='m-0'>No data found. Try a different query.</p>
          </div>
        </Col>
      </Row>
    )
  }

  const handleSampleClick = (timestamp: number, value: number, labels: Label.AsObject[]): void => {
    select(new SingleProfileSelection(labels, timestamp))
  }

  return (
      <div className="dark:bg-gray-700 rounded border-gray-300 dark:border-gray-500" style={{ borderWidth: 1 }}>
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
      </div>
  )
}

export default ProfileMetricsGraph
