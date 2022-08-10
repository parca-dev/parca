import {QueryServiceClient, QueryRequest_ReportType} from '@parca/client';

import {useQuery} from './useQuery';
import {ProfileView, useProfileVisState} from './ProfileView';
import {ProfileSource} from './ProfileSource';

type NavigateFunction = (path: string, queryParams: any) => void;

interface ProfileViewWithDataProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  navigateTo?: NavigateFunction;
  compare?: boolean;
}

export const ProfileViewWithData = ({
  queryClient,
  profileSource,
  navigateTo,
}: ProfileViewWithDataProps) => {
  const profileVisState = useProfileVisState();
  const {currentView} = profileVisState;
  const {
    isLoading: flamegraphLoading,
    response: flamegraphResponse,
    error: flamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED, {
    skip: currentView != 'icicle' && currentView != 'both',
  });

  const {
    isLoading: topTableLoading,
    response: topTableResponse,
    error: topTableError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.TOP, {
    skip: currentView != 'table' && currentView != 'both',
  });

  const sampleUnit = profileSource.ProfileType().sampleUnit;

  return (
    <ProfileView
      flamegraphData={{
        loading: flamegraphLoading,
        data:
          flamegraphResponse?.report.oneofKind === 'flamegraph'
            ? flamegraphResponse?.report?.flamegraph
            : undefined,
        error: flamegraphError,
      }}
      topTableData={{
        loading: topTableLoading,
        data:
          topTableResponse?.report.oneofKind === 'top' ? topTableResponse.report.top : undefined,
        error: topTableError,
      }}
      profileVisState={profileVisState}
      sampleUnit={sampleUnit}
      profileSource={profileSource}
      queryClient={queryClient}
      navigateTo={navigateTo}
    />
  );
};

export default ProfileViewWithData;
