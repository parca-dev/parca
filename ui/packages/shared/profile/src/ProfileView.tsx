import React, { useEffect, useState } from 'react'
// import ProfileSVG from './ProfileSVG'
// import ProfileTop from './ProfileTop'
import { CalcWidth } from '@parca/dynamicsize'
import ProfileIcicleGraph from './ProfileIcicleGraph'
import { ProfileSource } from './ProfileSource'
import { QueryRequest, QueryResponse, QueryServiceClient, ServiceError } from '@parca/client'
import Card from '../../../app/web/src/components/ui/Card'
import Button from '@parca/web/src/components/ui/Button'
import * as parca_query_v1alpha1_query_pb from '@parca/client/src/parca/query/v1alpha1/query_pb'

interface ProfileViewProps {
  queryClient: QueryServiceClient
  profileSource: ProfileSource
}

export interface IQueryResult {
  response: QueryResponse | null
  error: ServiceError | null
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource
): IQueryResult => {
  const [result, setResult] = useState<IQueryResult>({
    response: null,
    error: null
  })

  useEffect(() => {
    const req = profileSource.QueryRequest()
    req.setReportType(QueryRequest.ReportType.REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED)

    client.query(req, (error: ServiceError | null, responseMessage: QueryResponse | null) => {
      setResult({
        response: responseMessage,
        error: error
      })
    })
  }, [client, profileSource])

  return result
}

export const ProfileView = ({
                              queryClient,
                              profileSource
                            }: ProfileViewProps): JSX.Element => {
  const { response, error } = useQuery(queryClient, profileSource)

  if (error != null) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>
  }

  if (response == null) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: 'inherit',
          marginTop: 100
        }}
      >
        <svg className="animate-spin -ml-1 mr-3 h-5 w-5" xmlns="http://www.w3.org/2000/svg" fill="none"
             viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span>Loading...</span>
      </div>
    )
  }

  return (
    <>
      <div className="py-3">
        <Card>
          <Card.Body>
            <Button
              onClick={(e: React.MouseEvent<HTMLElement>) => {
                e.preventDefault()
                const req = profileSource.QueryRequest()
                queryClient.queryPprof(req, (error: ServiceError | null, responseMessage: parca_query_v1alpha1_query_pb.QueryPprofResponse | null) => {
                  if (responseMessage !== null) {
                    const bytes = responseMessage.getProfile()
                    const blob = new Blob([bytes], { type: 'application/octet-stream' })

                    const link = document.createElement('a')
                    link.href = window.URL.createObjectURL(blob)
                    link.download = 'profile.pb.gz'
                    link.click()
                  }
                })
              }}
            >Download</Button>
            <CalcWidth throttle={300} delay={2000}>
              <ProfileIcicleGraph
                graph={response.getFlamegraph()?.toObject()}
              />
            </CalcWidth>
          </Card.Body>
        </Card>
      </div>
    </>
  )
}
