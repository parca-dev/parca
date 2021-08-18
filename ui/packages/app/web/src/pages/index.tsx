import * as React from 'react'
import { NextPage } from 'next'
import { Button } from 'components/Button'
import { capitalize } from '@parca/functions'
import { grpc } from '@improbable-eng/grpc-web'
import { NodeHttpTransport } from '@improbable-eng/grpc-web-node-http-transport'
import { Query } from '@parca/client/src/query/query_pb_service'
import { QueryRequest } from '@parca/client/src/query/query_pb'

const host = 'http://localhost:9090'

function query (): void {
  const instantQuery = new QueryRequest()

  grpc.unary(Query.Query, {
    request: instantQuery,
    host: host, // process.env.HOST,
    transport: NodeHttpTransport(),
    onEnd: res => {
      const { status, statusMessage, headers, message, trailers } = res
      console.log('query.onEnd.status', status, statusMessage)
      console.log('query.onEnd.headers', headers)
      if (status === grpc.Code.OK && message != null) {
        console.log('query.onEnd.message', message.toObject())
      }
      console.log('query.onEnd.trailers', trailers)
    }
  })
}

const Index: NextPage = () => {
  const handleClick = (): void => {
    query()
  }

  return (
    <div>
      <Button label={capitalize('hello web')} onClick={handleClick} />
    </div>
  )
}

export default Index
