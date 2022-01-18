import {QueryServiceClient} from '@parcaui/client';
import ProfileExplorer from 'components/ProfileExplorer';
import {NextRouter, withRouter} from 'next/router';

const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT;

interface ProfilesProps {
  router: NextRouter;
}

const Profiles = (_: ProfilesProps): JSX.Element => {
  const queryClient = new QueryServiceClient(apiEndpoint === undefined ? '' : apiEndpoint);
  return <ProfileExplorer queryClient={queryClient} />;
};

export default withRouter(Profiles);
