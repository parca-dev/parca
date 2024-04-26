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
import {saveAsBlob, type NavigateFunction} from '@parca/utilities';

import {FIELD_FUNCTION_NAME} from './ProfileIcicleGraph/IcicleGraphArrow';
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
  const [dashboardItems = ['icicle']] = useURLState({param: 'dashboard_items', navigateTo});
  const [sourceBuildID] = useURLState({param: 'source_buildid', navigateTo}) as unknown as [string];
  const [sourceFilename] = useURLState({param: 'source_filename', navigateTo}) as unknown as [
    string,
  ];
  const [groupBy = [FIELD_FUNCTION_NAME]] = useURLState({param: 'group_by', navigateTo});

  const [showRuntimeRubyStr] = useURLState({param: 'show_runtime_ruby', navigateTo});
  const showRuntimeRuby = showRuntimeRubyStr === 'true';
  const [showRuntimePythonStr] = useURLState({param: 'show_runtime_python', navigateTo});
  const showRuntimePython = showRuntimePythonStr === 'true';
  const [showInterpretedOnlyStr] = useURLState({param: 'show_interpreted_only', navigateTo});
  const showInterpretedOnly = showInterpretedOnlyStr === 'true';

  const [pprofDownloading, setPprofDownloading] = useState<boolean>(false);

  const nodeTrimThreshold = useMemo(() => {
    let width =
      // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
      window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    // subtract the padding
    width = width - 12 - 16 - 12;
    return (1 / width) * 100;
  }, []);

  // make sure we get a string[]
  const groupByParam: string[] = typeof groupBy === 'string' ? [groupBy] : groupBy;

  const {
    isLoading: flamegraphLoading,
    response: flamegraphResponse,
    error: flamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_ARROW, {
    skip: !dashboardItems.includes('icicle'),
    nodeTrimThreshold,
    groupBy: groupByParam,
    showRuntimeRuby,
    showRuntimePython,
    showInterpretedOnly,
  });
  const {perf} = useParcaContext();

  const {
    isLoading: tableLoading,
    response: tableResponse,
    error: tableError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.TABLE_ARROW, {
    skip: !dashboardItems.includes('table'),
  });

  const {
    isLoading: callgraphLoading,
    response: callgraphResponse,
    error: callgraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.CALLGRAPH, {
    skip: !dashboardItems.includes('callgraph'),
  });

  const {
    isLoading: sourceLoading,
    response: sourceResponse,
    error: sourceError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.SOURCE, {
    skip: !dashboardItems.includes('source'),
    sourceBuildID,
    sourceFilename,
  });

  useEffect(() => {
    if (
      (!flamegraphLoading && flamegraphResponse?.report.oneofKind === 'flamegraph') ||
      flamegraphResponse?.report.oneofKind === 'flamegraphArrow'
    ) {
      perf?.markInteraction('Flamegraph render', flamegraphResponse.total);
    }

    if (!tableLoading && tableResponse?.report.oneofKind === 'tableArrow') {
      perf?.markInteraction('table render', tableResponse.total);
    }

    if (!callgraphLoading && callgraphResponse?.report.oneofKind === 'callgraph') {
      perf?.markInteraction('Callgraph render', callgraphResponse.total);
    }

    if (!sourceLoading && sourceResponse?.report.oneofKind === 'source') {
      perf?.markInteraction('Source render', sourceResponse.total);
    }
  }, [
    flamegraphLoading,
    flamegraphResponse,
    callgraphResponse,
    callgraphLoading,
    tableLoading,
    tableResponse,
    sourceLoading,
    sourceResponse,
    perf,
  ]);

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
  } else if (tableResponse !== null) {
    total = BigInt(tableResponse.total);
    filtered = BigInt(tableResponse.filtered);
  } else if (callgraphResponse !== null) {
    total = BigInt(callgraphResponse.total);
    filtered = BigInt(callgraphResponse.filtered);
  } else if (sourceResponse !== null) {
    total = BigInt(sourceResponse.total);
    filtered = BigInt(sourceResponse.filtered);
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
        arrow:
          flamegraphResponse?.report.oneofKind === 'flamegraphArrow'
            ? flamegraphResponse?.report?.flamegraphArrow
            : undefined,
        total: BigInt(flamegraphResponse?.total ?? '0'),
        filtered: BigInt(flamegraphResponse?.filtered ?? '0'),
        error: flamegraphError,
      }}
      topTableData={{
        loading: tableLoading,
        arrow:
          tableResponse?.report.oneofKind === 'tableArrow'
            ? tableResponse.report.tableArrow
            : undefined,
        error: tableError,
      }}
      callgraphData={{
        loading: callgraphLoading,
        data:
          callgraphResponse?.report.oneofKind === 'callgraph'
            ? callgraphResponse?.report?.callgraph
            : undefined,
        error: callgraphError,
      }}
      sourceData={{
        loading: sourceLoading,
        data:
          sourceResponse?.report.oneofKind === 'source'
            ? sourceResponse?.report?.source
            : undefined,
        error: sourceError,
      }}
      profileSource={profileSource}
      queryClient={queryClient}
      navigateTo={navigateTo}
      onDownloadPProf={() => void downloadPProfClick()}
      pprofDownloading={pprofDownloading}
    />
  );
};

export default ProfileViewWithData;
