import {QueryServiceClient} from '@parca/client';
import {useLocation, useNavigate} from 'react-router-dom';
import ProfileExplorer from '../components/ProfileExplorer';

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

const transformToArray = params => params.split(',');

const parseParams = (querystring: string) => {
  const params = new URLSearchParams(querystring);

  const obj: any = {};
  for (const key of params.keys()) {
    if (params.getAll(key).length > 1) {
      obj[key] = params.getAll(key);
    } else {
      if (params.get(key).includes(',')) {
        obj[key] = transformToArray(params.get(key));
      } else {
        obj[key] = params.get(key);
      }
    }
  }

  return obj;
};

const convertToQueryParams = params =>
  Object.keys(params)
    .map(key => key + '=' + params[key])
    .join('&');

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
    apiEndpoint === undefined ? '/api' : `${apiEndpoint}/api`
  );

  return (
    <ProfileExplorer queryClient={queryClient} queryParams={queryParams} navigateTo={navigateTo} />
  );
};

export default Profiles;
