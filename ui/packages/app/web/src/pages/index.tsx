import ProfileExplorer from 'components/ProfileExplorer'
import { NextRouter, withRouter } from 'next/router'
import { QueryClient } from '@parca/client'

const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT

interface ProfilesProps {
  router: NextRouter
}

const Profiles = (_: ProfilesProps): JSX.Element => {
  const queryClient = new QueryClient(apiEndpoint === undefined ? '' : apiEndpoint)
  return (
    <ProfileExplorer
      queryClient={queryClient}
    />
  )
}

export default withRouter(Profiles)
