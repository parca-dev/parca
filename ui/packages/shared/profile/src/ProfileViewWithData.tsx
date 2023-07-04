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

import {useEffect, useMemo, useState} from 'react';

import {QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {useGrpcMetadata, useParcaContext, useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {saveAsBlob, type NavigateFunction} from '@parca/utilities';

import {ProfileSource} from './ProfileSource';
import {ProfileView} from './ProfileView';
import {useQuery} from './useQuery';
import {downloadPprof} from './utils';

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
  const [dashboardItems] = useURLState({param: 'dashboard_items', navigateTo});
  const [enableTrimming] = useUserPreference<boolean>(USER_PREFERENCES.ENABLE_GRAPH_TRIMMING.key);
  const [pprofDownloading, setPprofDownloading] = useState<boolean>(false);

  const nodeTrimThreshold = useMemo(() => {
    if (!enableTrimming) {
      return 0;
    }

    let width =
      // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
      window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    // subtract the padding
    width = width - 12 - 16 - 12;
    return (1 / width) * 100;
  }, [enableTrimming]);

  const {
    isLoading: flamegraphLoading,
    response: flamegraphResponse,
    error: flamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_TABLE, {
    skip: !dashboardItems.includes('icicle'),
    nodeTrimThreshold,
  });
  const {perf} = useParcaContext();

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

  useEffect(() => {
    if (!flamegraphLoading && flamegraphResponse?.report.oneofKind === 'flamegraph') {
      perf?.markInteraction('Flamegraph render', flamegraphResponse.report.flamegraph.total);
    }

    if (!topTableLoading && topTableResponse?.report.oneofKind === 'top') {
      perf?.markInteraction('Top table render', topTableResponse?.report?.top.total);
    }

    if (!callgraphLoading && callgraphResponse?.report.oneofKind === 'callgraph') {
      perf?.markInteraction('Callgraph render', callgraphResponse?.report?.callgraph.cumulative);
    }
  }, [
    flamegraphLoading,
    flamegraphResponse,
    callgraphResponse,
    callgraphLoading,
    topTableLoading,
    topTableResponse,
    perf,
  ]);

  const sampleUnit = profileSource.ProfileType().sampleUnit;

  const downloadPProfClick = async (): Promise<void> => {
    if (profileSource == null || queryClient == null) {
      return;
    }

    try {
      setPprofDownloading(true);
      const blob = await downloadPprof(profileSource.QueryRequest(), queryClient, metadata);
      saveAsBlob(blob, `profile.pb.gz`);
      setPprofDownloading(false);
    } catch (error) {
      setPprofDownloading(false);
      console.error('Error while querying', error);
    }
  };

  // TODO: Refactor how we get responses such that we have a single response,
  //  regardless of the report type.
  let total = BigInt(0);
  let filtered = BigInt(0);
  if (flamegraphResponse !== null) {
    total = BigInt(flamegraphResponse.total);
    filtered = BigInt(flamegraphResponse.filtered);
  } else if (topTableResponse !== null) {
    total = BigInt(topTableResponse.total);
    filtered = BigInt(topTableResponse.filtered);
  } else if (callgraphResponse !== null) {
    total = BigInt(callgraphResponse.total);
    filtered = BigInt(callgraphResponse.filtered);
  }

  return (
    <ProfileView
      total={total}
      filtered={filtered}
      flamegraphData={{
        loading: flamegraphLoading,
        data:
          flamegraphResponse?.report.oneofKind === 'flamegraph'
            ? flamegraphResponse?.report?.flamegraph
            : undefined,
        total: BigInt(flamegraphResponse?.total ?? '0'),
        filtered: BigInt(flamegraphResponse?.filtered ?? '0'),
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
      pprofDownloading={pprofDownloading}
    />
  );
};

export default ProfileViewWithData;
