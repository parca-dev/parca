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
import {
  NumberParser,
  NumberSerializer,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
  useURLStateCustom,
} from '@parca/components';
import {saveAsBlob} from '@parca/utilities';

import {validateFlameChartQuery} from './ProfileFlameGraph';
import {FIELD_FUNCTION_NAME, FIELD_LABELS} from './ProfileFlameGraph/FlameGraphArrow';
import {boundsFromProfileSource} from './ProfileFlameGraph/FlameGraphArrow/utils';
import {
  getStepCountFromScreenWidth,
  useQueryRange,
} from './ProfileMetricsGraph/hooks/useQueryRange';
import {MergedProfileSource, ProfileSource} from './ProfileSource';
import {ProfileView} from './ProfileView';
import {useProfileFilters} from './ProfileView/components/ProfileFilters/useProfileFilters';
import type {SamplesSeries} from './ProfileView/types/visualization';
import {useQuery} from './useQuery';
import {downloadPprof} from './utils';

interface ProfileViewWithDataProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  compare?: boolean;
  showVisualizationSelector?: boolean;
  onSwitchToOneMinute?: () => void;
}

export const ProfileViewWithData = ({
  queryClient,
  profileSource,
  showVisualizationSelector,
  onSwitchToOneMinute,
}: ProfileViewWithDataProps): JSX.Element => {
  const metadata = useGrpcMetadata();
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const [sourceBuildID] = useURLState<string>('source_buildid');
  const [sourceFilename] = useURLState<string>('source_filename');
  const [groupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });
  const [sandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const [flamechartDimension] = useURLState<string[]>('flamechart_dimension', {
    alwaysReturnArray: true,
  });

  const [invertStack] = useURLState('invert_call_stack');
  const invertCallStack = invertStack === 'true';

  const [pprofDownloading, setPprofDownloading] = useState<boolean>(false);

  const {protoFilters} = useProfileFilters();

  useEffect(() => {
    // If profile type is not delta, remove flamechart from the dashboard items
    // and set it to flame graph if no other items are selected.
    if (profileSource == null) {
      return;
    }
    const profileType = profileSource.ProfileType();
    let newDashboardItems = dashboardItems;
    if (dashboardItems.includes('flamechart') && !profileType.delta) {
      newDashboardItems = dashboardItems.filter(item => item !== 'flamechart');
    } else {
      return;
    }
    if (newDashboardItems.length === 0) {
      newDashboardItems = ['flamegraph'];
    }
    setDashboardItems(newDashboardItems);
  }, [profileSource, dashboardItems, setDashboardItems]);

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
    skip: !dashboardItems.includes('flamegraph'),
    nodeTrimThreshold,
    groupBy,
    invertCallStack,
    protoFilters,
  });

  const samplesEnabled = !!(
    dashboardItems.includes('flamechart') &&
    validateFlameChartQuery(profileSource as MergedProfileSource).isValid &&
    (flamechartDimension ?? []).length > 0
  );

  const [samplesFromMs, samplesToMs] = useMemo(() => {
    const bounds = boundsFromProfileSource(profileSource);
    return [Number(bounds[0] / 1_000_000n), Number(bounds[1] / 1_000_000n)];
  }, [profileSource]);

  const samplesSumBy = useMemo(
    () =>
      (flamechartDimension ?? []).map(f =>
        f.startsWith('labels.') ? f.slice('labels.'.length) : f
      ),
    [flamechartDimension]
  );

  // Samples step count: 2px per data point for finer granularity in strips
  const [samplesStepCount] = useURLStateCustom<number>('samples_step_count', {
    defaultValue: String(getStepCountFromScreenWidth(2)),
    parse: NumberParser,
    stringify: NumberSerializer,
  });

  const {
    isLoading: samplesLoading,
    response: samplesRangeResponse,
    error: samplesError,
    stepDurationMs: samplesStepMs,
  } = useQueryRange(
    queryClient,
    samplesEnabled ? (profileSource as MergedProfileSource).query.toString() : '',
    samplesFromMs,
    samplesToMs,
    samplesSumBy,
    samplesStepCount,
    !samplesEnabled
  );

  // Map QueryRange response to SamplesData
  // TODO (manoj): Check if we can skip this mapping and adapt the CPUSampleStrips to work directly with the QueryRange response format.
  const samplesData = useMemo(() => {
    if (samplesLoading) return {loading: true, error: null};
    if (samplesError != null) return {loading: false, error: samplesError};
    if (samplesRangeResponse?.series?.length == null) return {loading: false, error: null};

    const series: SamplesSeries[] = samplesRangeResponse.series.map(ms => ({
      labelset: ms.labelset ?? {labels: []},
      data: ms.samples.map(s => ({
        timestamp: Number(s.timestamp!.seconds) * 1000 + Math.floor(s.timestamp!.nanos / 1_000_000),
        value: Number(s.value),
        sampleCount: Number(s.count),
      })),
    }));

    return {loading: false, series, error: null, stepMs: samplesStepMs};
  }, [samplesLoading, samplesRangeResponse, samplesError, samplesStepMs]);

  const {
    isLoading: profileMetadataLoading,
    response: profileMetadataResponse,
    error: profileMetadataError,
    refetch: metadataRefetch,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.PROFILE_METADATA, {
    nodeTrimThreshold,
    groupBy,
    protoFilters,
    staleTime: 0,
  });

  const {perf} = useParcaContext();

  const {
    isLoading: tableLoading,
    response: tableResponse,
    error: tableError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.TABLE_ARROW, {
    skip: !dashboardItems.includes('table') && !dashboardItems.includes('sandwich'),
    protoFilters,
  });

  const {
    isLoading: sourceLoading,
    response: sourceResponse,
    error: sourceError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.SOURCE, {
    skip: !dashboardItems.includes('source'),
    sourceBuildID,
    sourceFilename,
    protoFilters,
  });

  const {
    isLoading: callersFlamegraphLoading,
    response: callersFlamegraphResponse,
    error: callersFlamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_ARROW, {
    nodeTrimThreshold,
    groupBy: [FIELD_FUNCTION_NAME],
    invertCallStack: true,
    sandwichByFunction: sandwichFunctionName,
    skip: sandwichFunctionName === undefined && !dashboardItems.includes('sandwich'),
    protoFilters,
  });

  const {
    isLoading: calleesFlamegraphLoading,
    response: calleesFlamegraphResponse,
    error: calleesFlamegraphError,
  } = useQuery(queryClient, profileSource, QueryRequest_ReportType.FLAMEGRAPH_ARROW, {
    nodeTrimThreshold,
    groupBy: [FIELD_FUNCTION_NAME],
    invertCallStack: false,
    sandwichByFunction: sandwichFunctionName,
    skip: sandwichFunctionName === undefined && !dashboardItems.includes('sandwich'),
    protoFilters,
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

    if (!sourceLoading && sourceResponse?.report.oneofKind === 'source') {
      perf?.markInteraction('Source render', sourceResponse.total);
    }
  }, [
    flamegraphLoading,
    flamegraphResponse,
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
      const req = profileSource.QueryRequest();
      req.groupBy = {fields: groupBy};
      const blob = await downloadPprof(req, queryClient, metadata);
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
  } else if (sourceResponse !== null) {
    total = BigInt(sourceResponse.total);
    filtered = BigInt(sourceResponse.filtered);
  } else if (callersFlamegraphResponse !== null) {
    total = BigInt(callersFlamegraphResponse.total);
    filtered = BigInt(callersFlamegraphResponse.filtered);
  } else if (calleesFlamegraphResponse !== null) {
    total = BigInt(calleesFlamegraphResponse.total);
    filtered = BigInt(calleesFlamegraphResponse.filtered);
  }

  return (
    <ProfileView
      total={total}
      filtered={filtered}
      flamegraphData={{
        loading: flamegraphLoading && profileMetadataLoading,
        arrow:
          flamegraphResponse?.report.oneofKind === 'flamegraphArrow'
            ? flamegraphResponse?.report?.flamegraphArrow
            : undefined,
        total: BigInt(flamegraphResponse?.total ?? '0'),
        filtered: BigInt(flamegraphResponse?.filtered ?? '0'),
        error: flamegraphError ?? profileMetadataError,
        metadataMappingFiles:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.mappingFiles
            : undefined,
        metadataLabels:
          profileMetadataResponse?.report.oneofKind === 'profileMetadata'
            ? profileMetadataResponse?.report?.profileMetadata?.labels
            : undefined,
        metadataLoading: profileMetadataLoading,
        metadataRefetch,
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
      sourceData={{
        loading: sourceLoading,
        data:
          sourceResponse?.report.oneofKind === 'source'
            ? sourceResponse?.report?.source
            : undefined,
        error: sourceError,
      }}
      sandwichData={{
        callees: {
          arrow:
            calleesFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
              ? calleesFlamegraphResponse?.report?.flamegraphArrow
              : undefined,
          loading: calleesFlamegraphLoading,
          error: calleesFlamegraphError,
          total: BigInt(calleesFlamegraphResponse?.total ?? '0'),
          filtered: BigInt(calleesFlamegraphResponse?.filtered ?? '0'),
          metadataMappingFiles:
            profileMetadataResponse?.report.oneofKind === 'profileMetadata'
              ? profileMetadataResponse?.report?.profileMetadata?.mappingFiles
              : undefined,
          metadataLoading: profileMetadataLoading,
        },
        callers: {
          arrow:
            callersFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
              ? callersFlamegraphResponse?.report?.flamegraphArrow
              : undefined,
          loading: callersFlamegraphLoading,
          error: callersFlamegraphError,
          total: BigInt(callersFlamegraphResponse?.total ?? '0'),
          filtered: BigInt(callersFlamegraphResponse?.filtered ?? '0'),
          metadataMappingFiles:
            profileMetadataResponse?.report.oneofKind === 'profileMetadata'
              ? profileMetadataResponse?.report?.profileMetadata?.mappingFiles
              : undefined,
          metadataLoading: profileMetadataLoading,
        },
      }}
      samplesData={samplesData}
      profileSource={profileSource}
      queryClient={queryClient}
      onDownloadPProf={() => void downloadPProfClick()}
      pprofDownloading={pprofDownloading}
      showVisualizationSelector={showVisualizationSelector}
      onSwitchToOneMinute={onSwitchToOneMinute}
    />
  );
};

export default ProfileViewWithData;
