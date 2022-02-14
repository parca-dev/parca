import {QueryServiceClient} from '@parca/client';
import ProfileExplorer from 'components/ProfileExplorer';
import {NextRouter, withRouter} from 'next/router';

const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT;

interface ProfilesProps {
  router: NextRouter;
}

const Profiles = (_: ProfilesProps): JSX.Element => {
  const queryClient = new QueryServiceClient(
    apiEndpoint === undefined ? '/api' : `${apiEndpoint}/api`
  );
  return <ProfileExplorer queryClient={queryClient} />;
};

export default withRouter(Profiles);
