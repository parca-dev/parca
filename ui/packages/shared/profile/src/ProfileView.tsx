import { useState, useEffect } from 'react'
//import ProfileSVG from './ProfileSVG'
//import ProfileTop from './ProfileTop'
import ProfileIcicleGraph from './ProfileIcicleGraph'
import { ProfileSource } from './ProfileSource'
import { QueryServiceClient, QueryResponse, QueryRequest, ServiceError } from '@parca/client'
import { Spinner } from 'react-bootstrap'
import Button from '../../../app/web/src/components/ui/Button'
import Card from '../../../app/web/src/components/ui/Card'
import Dropdown from '../../../app/web/src/components/ui/Dropdown'

interface ProfileViewProps {
  title?: string
  queryClient: QueryServiceClient
  profileSource: ProfileSource
  startComparing: () => void
  allowComparing: boolean
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
  title,
  queryClient,
  profileSource,
  startComparing,
  allowComparing
}: ProfileViewProps): JSX.Element => {
  const { response, error } = useQuery(queryClient, profileSource)

  const [showModal, setShowModal] = useState(false)
  const [reportType, setReportType] = useState('iciclegraph')

  if (error != null) {
    return <div className='p-10 flex justify-center'>An error occurred: {error.message}</div>
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
        <Spinner animation='border' role='status'>
          <span className='sr-only'>Loading...</span>
        </Spinner>
      </div>
    )
  }

  const queryResponse = response.toObject()

  const reportTypes = {
    //svg: {
    //  name: 'Graph',
    //  element: (
    //    <ProfileSVG
    //      queryEndpoint={apiEndpoint}
    //      profileSource={profileSource}
    //      sampleIndex={sampleIndex}
    //    />
    //  )
    //},
    //top: {
    //  name: 'Top',
    //  element: (
    //    <ProfileTop
    //      queryEndpoint={apiEndpoint}
    //      profileSource={profileSource}
    //      sampleIndex={sampleIndex}
    //    />
    //  )
    //},
    iciclegraph: {
      name: 'Icicle Graph',
      element: <ProfileIcicleGraph graph={response.getFlamegraph()?.toObject()} />
    }
  }

  //TODO
  const downloadURL = ''

  return (
    <>
      <div className='my-4'>
        <Card>
          <Card.Body>
            <div className='flex justify-between'>
              <div className='flex'>{title}</div>
              <div className='flex space-x-4'>
                {allowComparing && (
                  <Button color='neutral' onClick={() => startComparing()}>
                    Compare
                  </Button>
                )}
                <Dropdown text='View'>
                  {Object.keys(reportTypes).map((k: string) => (
                    <Dropdown.Item key={k} onSelect={() => setReportType(k)}>
                      {reportTypes[k].name}
                    </Dropdown.Item>
                  ))}
                </Dropdown>
                <Button
                  color='neutral'
                  // href={downloadURL}
                >
                  Download
                </Button>
              </div>
            </div>

            {reportTypes[reportType].element}
          </Card.Body>
        </Card>
      </div>
    </>
  )
}
