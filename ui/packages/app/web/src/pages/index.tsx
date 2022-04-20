import {GrpcWebFetchTransport} from '@protobuf-ts/grpcweb-transport';
import {QueryServiceClient} from '@parca/client';
import {useLocation, useNavigate} from 'react-router-dom';
import {parseParams, convertToQueryParams} from '@parca/functions';
import {ProfileExplorer} from '@parca/components';

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

const Profiles = () => {
  const location = useLocation();
  const navigate = useNavigate();

  const navigateTo = (path: string, queryParams: any) => {
    navigate({
      pathname: path,
      search: `?${convertToQueryParams(queryParams)}`,
    });
  };

  const queryParams = parseParams(location.search);

  const queryClient = new QueryServiceClient(
    new GrpcWebFetchTransport({
      baseUrl: apiEndpoint === undefined ? '/api' : `${apiEndpoint}/api`,
    })
  );

  return (
    <ProfileExplorer queryClient={queryClient} queryParams={queryParams} navigateTo={navigateTo} />
  );
};

export default Profiles;
