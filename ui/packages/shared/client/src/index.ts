export * from '@parca/client/src/query/query_pb';
export * from '@parca/client/src/query/query_pb_service';
export * from '@parca/client/src/profilestore/profilestore_pb';

import { grpc } from '@improbable-eng/grpc-web'
import { NodeHttpTransport } from '@improbable-eng/grpc-web-node-http-transport'
import { Query } from '@parca/client/src/query/query_pb_service'
import {
  LabelsRequest,
  QueryRangeRequest,
  QueryRequest,
  ValuesRequest
} from '@parca/client/src/query/query_pb'
import { Timestamp } from './google/api/timestamp_pb'

// TODO(kakkoyun): !!
// const host = 'http://localhost:9090' // process.env.HOST

export function query(host: string): Promise<any> {
  const instantQuery = new QueryRequest()

  const promise = new Promise<any>((resolve, reject) => {
    grpc.unary(Query.Query, {
      request: instantQuery,
      host: host,
      transport: NodeHttpTransport(),
      onEnd: res => {
        const { status, statusMessage, headers, message, trailers } = res
        console.log('query.onEnd.status', status, statusMessage)
        console.log('query.onEnd.headers', headers)
        if (status === grpc.Code.OK && message != null) {
          console.log('query.onEnd.message', message.toObject())
          resolve(message.toObject())
        }
        console.log('query.onEnd.trailers', trailers)
        reject(statusMessage)
      }
    })
  })

  return promise
}

export function queryRange(
  host: string,
  query: string,
  from: number,
  to: number,
  limit: number
): Promise<any> {
  const rangeQuery = new QueryRangeRequest()
  const start = new Timestamp()
  start.setNanos(from)
  const end = new Timestamp()
  start.setNanos(to)

  //rangeQuery.setQuery(query)
  //rangeQuery.setStart(start)
  //rangeQuery.setEnd(end)
  //rangeQuery.setLimit(limit)

  const promise = new Promise<any>((resolve, reject) => {
    grpc.unary(Query.QueryRange, {
      request: rangeQuery,
      host: host,
      transport: NodeHttpTransport(),
      onEnd: res => {
        const { status, statusMessage, headers, message, trailers } = res
        console.log('query.onEnd.status', status, statusMessage)
        console.log('query.onEnd.headers', headers)
        if (status === grpc.Code.OK && message != null) {
          console.log('query.onEnd.message', message.toObject())
          resolve(message.toObject())
        }
        console.log('query.onEnd.trailers', trailers)
        reject(statusMessage)
      }
    })
  })

  return promise
}

export function labels(host: string): Promise<any> {
  const rangeQuery = new LabelsRequest()

  const promise = new Promise<any>((resolve, reject) => {
    grpc.unary(Query.Labels, {
      request: rangeQuery,
      host: host,
      transport: NodeHttpTransport(),
      onEnd: res => {
        const { status, statusMessage, headers, message, trailers } = res
        console.log('query.onEnd.status', status, statusMessage)
        console.log('query.onEnd.headers', headers)
        if (status === grpc.Code.OK && message != null) {
          console.log('query.onEnd.message', message.toObject())
          resolve(message.toObject())
        }
        console.log('query.onEnd.trailers', trailers)
        reject(statusMessage)
      }
    })
  })

  return promise
}

export function values(host: string): Promise<any> {
  const rangeQuery = new ValuesRequest()

  const promise = new Promise<any>((resolve, reject) => {
    grpc.unary(Query.Values, {
      request: rangeQuery,
      host: host,
      transport: NodeHttpTransport(),
      onEnd: res => {
        const { status, statusMessage, headers, message, trailers } = res
        console.log('query.onEnd.status', status, statusMessage)
        console.log('query.onEnd.headers', headers)
        if (status === grpc.Code.OK && message != null) {
          console.log('query.onEnd.message', message.toObject())
          resolve(message.toObject())
        }
        console.log('query.onEnd.trailers', trailers)
        reject(statusMessage)
      }
    })
  })

  return promise
}
