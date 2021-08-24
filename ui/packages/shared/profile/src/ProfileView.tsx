import { useState, useEffect } from 'react'
//import ProfileSVG from './ProfileSVG'
//import ProfileTop from './ProfileTop'
import ProfileIcicleGraph from './ProfileIcicleGraph'
import { ProfileSource } from './ProfileSource'
import { QueryClient, QueryResponse, QueryRequest, ServiceError } from '@parca/client'
import {
  Button,
  Card,
  Col,
  Dropdown,
  DropdownButton,
  Row,
  Spinner
} from 'react-bootstrap'

interface ProfileViewProps {
  title?: string
  queryClient: QueryClient
  profileSource: ProfileSource
  startComparing: () => void
  allowComparing: boolean
}

export interface IQueryResult {
  response: QueryResponse|null
  error: ServiceError|null
}

export const useQuery = (
  client: QueryClient,
  profileSource: ProfileSource,
): IQueryResult => {
  const [result, setResult] = useState<IQueryResult>({
    response: null,
    error: null
  })

  useEffect(() => {
    const req = profileSource.QueryRequest()
    req.setReportType(QueryRequest.ReportType.FLAMEGRAPH)

    client.query(
      req,
      (error: ServiceError|null, responseMessage: QueryResponse|null) => {
        setResult({
          response: responseMessage,
          error: error
        })
      }
    )
  }, [client, profileSource])

  return result
}

export const ProfileView = ({
  title,
  queryClient,
  profileSource,
  startComparing,
  allowComparing,
}: ProfileViewProps): JSX.Element => {
  const { response, error } = useQuery(
    queryClient,
    profileSource
  )

  const [showModal, setShowModal] = useState(false)
  const [reportType, setReportType] = useState('iciclegraph')

  if (error != null) {
    return <div>An error occurred: {error.message}</div>
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
  console.log("profileResponse", queryResponse)

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
      element: (
        <ProfileIcicleGraph
          graph={response.getFlamegraph()?.toObject()}
        />
      )
    }
  }

  //TODO
  const downloadURL = ''

  return (
    <>
      <Row style={{ marginBottom: 20 }} className='profile-view'>
        <Col xs='12'>
          <Card style={{ width: '100%', marginTop: 10 }}>
            <Card.Body style={{ width: '100%' }}>
              <Row>
                <Col>{title !== undefined && title.length > 0 && <p>{title}</p>}</Col>
                <Col md='6' style={{ textAlign: 'right' }}>
                  {allowComparing && (
                    <Button
                      style={{ display: 'inline-block' }}
                      variant='light'
                      onClick={() => startComparing()}
                    >
                      Compare
                    </Button>
                  )}
                  <DropdownButton
                    style={{ display: 'inline-block' }}
                    title='View'
                    variant='light'
                    alignRight
                  >
                    {Object.keys(reportTypes).map((k: string) => (
                      <Dropdown.Item key={k} onSelect={() => setReportType(k)}>
                        {reportTypes[k].name}
                      </Dropdown.Item>
                    ))}
                  </DropdownButton>
                  <Button style={{ display: 'inline-block' }} variant='light' href={downloadURL}>
                    Download
                  </Button>
                </Col>
              </Row>
              {reportTypes[reportType].element}
            </Card.Body>
          </Card>
        </Col>
      </Row>
    </>
  )
}
