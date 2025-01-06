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
import {saveAsBlob} from '@parca/utilities';

import {FIELD_FUNCTION_NAME} from './ProfileIcicleGraph/IcicleGraphArrow';
import {ProfileSource} from './ProfileSource';
import {ProfileView} from './ProfileView';
import {useQuery} from './useQuery';
import {downloadPprof} from './utils';

interface ProfileViewWithDataProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  compare?: boolean;
  showVisualizationSelector?: boolean;
}

export const ProfileViewWithData = ({
  queryClient,
  profileSource,
  showVisualizationSelector,
}: ProfileViewWithDataProps): JSX.Element => {
  const metadata = useGrpcMetadata();
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const [sourceBuildID] = useURLState<string>('source_buildid');
  const [sourceFilename] = useURLState<string>('source_filename');
  const [groupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });

  const [invertStack] = useURLState('invert_call_stack');
  const invertCallStack = invertStack === 'true';
  const [binaryFrameFilterStr] = useURLState<string[] | string>('binary_frame_filter');

  const binaryFrameFilter: string[] =
    typeof binaryFrameFilterStr === 'string'
      ? binaryFrameFilterStr.split(',')
      : binaryFrameFilterStr;

  const [pprofDownloading, setPprofDownloading] = useState<boolean>(false);

  const nodeTrimThreshold = useMemo(() => {
    let width =
      // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
      window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    // subtract the padding
    width = width - 12 - 16 - 12;
    return (1 / width) * 100;
  }, []);

  const {
    isLoading: flamegraphLoading,
    response: flamegraphResponse,
    error: flamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_ARROW, {
    skip: !dashboardItems.includes('icicle'),
    nodeTrimThreshold,
    groupBy,
    invertCallStack,
    binaryFrameFilter,
  });

  const {
    isLoading: flamechartLoading,
    response: flamechartResponse,
    error: flamechartError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMECHART, {
    skip: !dashboardItems.includes('iciclechart'),
    nodeTrimThreshold,
    groupBy,
    invertCallStack,
    binaryFrameFilter,
  });

  const {isLoading: profileMetadataLoading, response: profileMetadataResponse} = useQuery(
    queryClient,
    profileSource,
    QueryRequest_ReportType.PROFILE_METADATA,
    {
      nodeTrimThreshold,
      groupBy,
    }
  );

  const {perf} = useParcaContext();

  const {
    isLoading: tableLoading,
    response: tableResponse,
    error: tableError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.TABLE_ARROW, {
    skip: !dashboardItems.includes('table'),
    binaryFrameFilter,
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
  } else if (flamechartResponse !== null) {
    total = BigInt(flamechartResponse.total);
    filtered = BigInt(flamechartResponse.filtered);
  }

  return (
    <ProfileView
      total={total}
      filtered={filtered}
      flamegraphData={{
        loading: flamegraphLoading && profileMetadataLoading,
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
        metadataMappingFiles:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.mappingFiles
            : undefined,
        metadataLabels:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.labels
            : undefined,
        metadataLoading: profileMetadataLoading,
      }}
      flamechartData={{
        loading: flamechartLoading && profileMetadataLoading,
        arrow:
          flamechartResponse?.report.oneofKind === 'flamegraphArrow'
            ? flamechartResponse?.report?.flamegraphArrow
            : undefined,
        total: BigInt(flamechartResponse?.total ?? '0'),
        filtered: BigInt(flamechartResponse?.filtered ?? '0'),
        error: flamechartError,
        metadataMappingFiles:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.mappingFiles
            : undefined,
        metadataLabels:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.labels
            : undefined,
        metadataLoading: profileMetadataLoading,
      }}
      topTableData={{
        loading: tableLoading,
        arrow:
          tableResponse?.report.oneofKind === 'tableArrow'
            ? tableResponse.report.tableArrow
            : undefined,
        error: tableError,
        unit:
          tableResponse?.report.oneofKind === 'tableArrow'
            ? tableResponse.report.tableArrow.unit
            : '',
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
      onDownloadPProf={() => void downloadPProfClick()}
      pprofDownloading={pprofDownloading}
      showVisualizationSelector={showVisualizationSelector}
    />
  );
};

export default ProfileViewWithData;
