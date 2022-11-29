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
import {NavigateOptions} from 'react-router-dom';

import {useQuery} from './useQuery';
import {ProfileView} from './ProfileView';
import {ProfileSource} from './ProfileSource';
import {downloadPprof} from './utils';
import {useGrpcMetadata, useParcaContext} from '@parca/components';
import {saveAsBlob, selectQueryParam} from '@parca/functions';
import {useEffect} from 'react';

export type NavigateFunction = (path: string, queryParams: any, options?: NavigateOptions) => void;

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
}: ProfileViewWithDataProps): JSX.Element => {
  const metadata = useGrpcMetadata();
  const dashboardItems = selectQueryParam('dashboard_items') as string[];
  const {
    isLoading: flamegraphLoading,
    response: flamegraphResponse,
    error: flamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_TABLE, {
    skip: !dashboardItems.includes('icicle'),
  });
  const {perf} = useParcaContext();

  useEffect(() => {
    if (flamegraphLoading) {
      return;
    }

    if (flamegraphResponse?.report.oneofKind !== 'flamegraph') {
      return;
    }

    perf?.markInteraction('Flamegraph Render', flamegraphResponse?.report?.flamegraph.total);
  }, [flamegraphLoading, flamegraphResponse, perf]);

  const {
    isLoading: topTableLoading,
    response: topTableResponse,
    error: topTableError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.TOP, {
    skip: !dashboardItems.includes('table'),
  });

  const {
    isLoading: callgraphLoading,
    response: callgraphResponse,
    error: callgraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.CALLGRAPH, {
    skip: !dashboardItems.includes('callgraph'),
  });

  const sampleUnit = profileSource.ProfileType().sampleUnit;

  const downloadPProfClick = async (): Promise<void> => {
    if (profileSource == null || queryClient == null) {
      return;
    }

    try {
      const blob = await downloadPprof(profileSource.QueryRequest(), queryClient, metadata);
      saveAsBlob(blob, `profile.pb.gz`);
    } catch (error) {
      console.error('Error while querying', error);
    }
  };

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
      callgraphData={{
        loading: callgraphLoading,
        data:
          callgraphResponse?.report.oneofKind === 'callgraph'
            ? callgraphResponse?.report?.callgraph
            : undefined,
        error: callgraphError,
      }}
      sampleUnit={sampleUnit}
      profileSource={profileSource}
      queryClient={queryClient}
      navigateTo={navigateTo}
      onDownloadPProf={() => void downloadPProfClick()}
      dashboardItems={dashboardItems}
    />
  );
};

export default ProfileViewWithData;
