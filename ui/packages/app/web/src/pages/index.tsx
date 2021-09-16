import ProfileExplorer from 'components/ProfileExplorer'
import { NextRouter, withRouter } from 'next/router'
import { QueryServiceClient } from '@parca/client'
import Cookies from 'universal-cookie'

const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT

interface ProfilesProps {
  router: NextRouter
}

const Profiles = (_: ProfilesProps): JSX.Element => {
  const queryClient = new QueryServiceClient(apiEndpoint === undefined ? '' : apiEndpoint)
  return <ProfileExplorer queryClient={queryClient} />
}

export default withRouter(Profiles)

export function getServerSideProps({ req }) {
  const cookies = new Cookies(req ? req.headers.cookie : undefined)
  const persistedState = cookies.get('parca') || {}
  return {
    props: { persistedState }
  }
}
