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

import {memo, useEffect, useState} from 'react';
import {QueryServiceClient, QueryRequest_ReportType} from '@parca/client';
import {useQuery} from './useQuery';
import {ProfileView} from './ProfileView';
import {ProfileSource} from './ProfileSource';
import {downloadPprof} from './utils';
import {useGrpcMetadata, useParcaContext, useURLState} from '@parca/components';
import {saveAsBlob} from '@parca/functions';
import type {NavigateFunction} from '@parca/functions';
import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';

interface ProfileViewWithDataProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  navigateTo?: NavigateFunction;
  compare?: boolean;
}

export const ProfileViewWithData = memo(
  ({queryClient, profileSource, navigateTo}: ProfileViewWithDataProps): JSX.Element => {
    const metadata = useGrpcMetadata();
    const [dashboardItems] = useURLState({param: 'dashboard_items', navigateTo});
    const [nodeTrimThreshold, setNodeTrimThreshold] = useState<number>(0);
    const [enableTrimming] = useUserPreference<boolean>(USER_PREFERENCES.ENABLE_GRAPH_TRIMMING.key);

    useEffect(() => {
      if (!enableTrimming) {
        setNodeTrimThreshold(0);
      }
    }, [enableTrimming]);

    const onFlamegraphContainerResize = (width: number): void => {
      if (!enableTrimming || width === 0) {
        return;
      }
      const threshold = (1 / width) * 100;
      if (threshold === nodeTrimThreshold) {
        return;
      }
      setNodeTrimThreshold(threshold);
    };

    const {
      isLoading: flamegraphLoading,
      response: flamegraphResponse,
      error: flamegraphError,
    } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_TABLE, {
      skip: !dashboardItems.includes('icicle'),
      nodeTrimThreshold,
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
        onFlamegraphContainerResize={onFlamegraphContainerResize}
      />
    );
  }
);

export default ProfileViewWithData;
