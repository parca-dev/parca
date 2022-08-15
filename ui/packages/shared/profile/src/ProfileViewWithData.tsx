// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
