import ProfileSelector, { QuerySelection } from './ProfileSelector'
import { Col, Row } from 'react-bootstrap'
import { ProfileSelection, ProfileView } from '@parca/profile'
import { QueryClient } from '@parca/client'

interface ProfileExplorerSingleProps {
  queryClient: QueryClient
  query: QuerySelection
  selectQuery: (query: QuerySelection) => void
  selectProfile: (source: ProfileSelection) => void
  profile: ProfileSelection | null
  compareProfile: () => void
}

const ProfileExplorerSingle = ({
  queryClient,
  query,
  selectQuery,
  selectProfile,
  profile,
  compareProfile
}: ProfileExplorerSingleProps): JSX.Element => {
  return (
    <>
      <Row style={{ marginTop: 10 }}>
        <Col xs={12}>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={query}
            selectQuery={selectQuery}
            selectProfile={selectProfile}
            profileSelection={profile}
            comparing={false}
            onCompareProfile={compareProfile}
            enforcedProfileName={''} // TODO
          />
        </Col>
      </Row>
      <Row>
        <Col xs={12}>
          {profile != null
            ? (
              <ProfileView
                queryClient={queryClient}
                profileSource={profile.ProfileSource()}
                allowComparing={false}
                startComparing={() => {}}
              />
              )
            : (
              <></>
              )}
        </Col>
      </Row>
    </>
  )
}

export default ProfileExplorerSingle
