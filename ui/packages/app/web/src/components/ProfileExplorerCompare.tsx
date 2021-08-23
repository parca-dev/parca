import { Alert, Col, Row } from 'react-bootstrap'
import ProfileSelector, { QuerySelection } from './ProfileSelector'
import { ProfileDiffSource, ProfileSelection, ProfileView } from '@parca/profile'
import { Query } from '@parca/parser'
import { QueryClient } from '@parca/client'

interface ProfileExplorerCompareProps {
  queryClient: QueryClient

  queryA: QuerySelection
  queryB: QuerySelection
  profileA: ProfileSelection | null
  profileB: ProfileSelection | null
  selectQueryA: (query: QuerySelection) => void
  selectQueryB: (query: QuerySelection) => void
  selectProfileA: (source: ProfileSelection) => void
  selectProfileB: (source: ProfileSelection) => void
}

const ProfileExplorerCompare = ({
  queryClient,
  queryA,
  queryB,
  profileA,
  profileB,
  selectQueryA,
  selectQueryB,
  selectProfileA,
  selectProfileB
}: ProfileExplorerCompareProps): JSX.Element => {
  return (
    <>
      <Row style={{ marginTop: 10 }}>
        <Col xs={6}>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryA}
            profileSelection={profileA}
            selectProfile={selectProfileA}
            selectQuery={selectQueryA}
            enforcedProfileName={''}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </Col>
        <Col xs={6}>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryB}
            profileSelection={profileB}
            selectProfile={selectProfileB}
            selectQuery={selectQueryB}
            enforcedProfileName={Query.parse(queryA.expression).profileName()}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </Col>
      </Row>
      <Row>
        <Col xs={12}>
          {profileA != null && profileB != null
            ? (
            <ProfileView
              queryClient={queryClient}
              profileSource={new ProfileDiffSource(
                profileA.ProfileSource(),
                profileB.ProfileSource()
              )}
              allowComparing={false}
              startComparing={() => {}}
            />
              )
            : (
            <div>
              <Alert variant="info">
                Select a profile on both sides.
              </Alert>
            </div>
              )}
        </Col>
      </Row>
    </>
  )
}

export default ProfileExplorerCompare
