import {QueryServiceClient} from '@parca/client';
import ProfileExplorer from '../components/ProfileExplorer';

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

const Profiles = () => {
  const queryClient = new QueryServiceClient(apiEndpoint === undefined ? '' : apiEndpoint);
  return <ProfileExplorer queryClient={queryClient} />;
};

export default Profiles;
