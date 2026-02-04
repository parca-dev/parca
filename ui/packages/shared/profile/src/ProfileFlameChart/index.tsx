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

import {useEffect, useMemo, useRef} from 'react';

import {LabelSet, QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {
  useParcaContext,
  useURLState,
  useURLStateCustom,
  type OptionsCustom,
} from '@parca/components';
import {Matcher, MatcherTypes, ProfileType, Query} from '@parca/parser';

import ProfileFlameGraph, {validateFlameChartQuery} from '../ProfileFlameGraph';
import {FIELD_LABELS} from '../ProfileFlameGraph/FlameGraphArrow';
import {boundsFromProfileSource} from '../ProfileFlameGraph/FlameGraphArrow/utils';
import {MergedProfileSource, ProfileSource} from '../ProfileSource';
import type {SamplesData} from '../ProfileView/types/visualization';
import {useQuery} from '../useQuery';
import {NumberDuo} from '../utils';
import {SamplesStrip} from './SamplesStrips';

interface SelectedTimeframe {
  labels: LabelSet;
  bounds: NumberDuo;
}

const TimeframeStateSerializer: OptionsCustom<SelectedTimeframe | undefined> = {
  parse: (value: string) => {
    if (value == null || value === '' || value === 'undefined') {
      return undefined;
    }
    try {
      const [labelPart, boundsPart] = value.split('|');
      if (labelPart != null && boundsPart != null) {
        const labels = labelPart.split(',').map(labelStr => {
          const [name, ...rest] = labelStr.split(':');
          return {name, value: rest.join(':')};
        });
        const [startMs, endMs] = boundsPart.split(',').map(Number);
        if (labels.length > 0 && !isNaN(startMs) && !isNaN(endMs)) {
          return {
            labels: {labels},
            bounds: [startMs, endMs] as NumberDuo,
          };
        }
      }
    } catch {
      // Ignore parsing errors
    }
    return undefined;
  },
  stringify: (value: SelectedTimeframe | undefined) => {
    if (value == null) {
      return '';
    }
    const labelsStr = value.labels.labels.map(l => `${l.name}:${l.value}`).join(',');
    return `${labelsStr}|${value.bounds[0]},${value.bounds[1]}`;
  },
};

interface ProfileFlameChartProps {
  samplesData?: SamplesData;
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  width: number;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  isHalfScreen: boolean;
  metadataMappingFiles?: string[];
  metadataLoading?: boolean;
}

// Helper to create a filtered profile source with narrowed time bounds
// and dimension label matchers from the selected strip.
const createFilteredProfileSource = (
  profileSource: ProfileSource,
  selectedTimeframe: {labels: LabelSet; bounds: NumberDuo}
): ProfileSource | null => {
  if (!(profileSource instanceof MergedProfileSource)) {
    return null;
  }

  // The bounds are in milliseconds, convert to nanoseconds for the profile source
  const mergeFrom = BigInt(selectedTimeframe.bounds[0]) * 1_000_000n;
  const mergeTo = BigInt(selectedTimeframe.bounds[1]) * 1_000_000n;

  // Add dimension labels as additional matchers to the query
  const dimensionMatchers = selectedTimeframe.labels.labels.map(
    l => new Matcher(l.name, MatcherTypes.MatchEqual, l.value)
  );

  const query = new Query(
    profileSource.query.profType,
    [...profileSource.query.matchers, ...dimensionMatchers],
    ''
  );

  return new MergedProfileSource(mergeFrom, mergeTo, query);
};

export const ProfileFlameChart = ({
  samplesData,
  queryClient,
  profileSource,
  width,
  total,
  filtered,
  profileType,
  isHalfScreen,
  metadataMappingFiles,
  metadataLoading,
}: ProfileFlameChartProps): JSX.Element => {
  const {loader} = useParcaContext();

  const [selectedTimeframe, setSelectedTimeframe] = useURLStateCustom<
    SelectedTimeframe | undefined
  >('flamechart_timeframe', TimeframeStateSerializer);

  // Read flamechart dimension from URL state to detect changes
  const [flamechartDimension] = useURLState<string[]>('flamechart_dimension', {
    alwaysReturnArray: true,
  });

  // Reset selection when the parent time range (profileSource) changes
  const timeBoundsKey = boundsFromProfileSource(profileSource).join(',');
  const prevTimeBoundsKey = useRef(timeBoundsKey);
  useEffect(() => {
    if (prevTimeBoundsKey.current !== timeBoundsKey) {
      prevTimeBoundsKey.current = timeBoundsKey;
      setSelectedTimeframe(undefined);
    }
  }, [timeBoundsKey, setSelectedTimeframe]);

  // Reset selection when the dimension changes
  const dimensionKey = (flamechartDimension ?? []).join(',');
  const prevDimensionKey = useRef(dimensionKey);
  useEffect(() => {
    if (prevDimensionKey.current !== dimensionKey) {
      prevDimensionKey.current = dimensionKey;
      setSelectedTimeframe(undefined);
    }
  }, [dimensionKey, setSelectedTimeframe]);

  // Handle timeframe selection from strips
  const handleSelectedTimeframe = (labels: LabelSet, bounds: NumberDuo | undefined): void => {
    if (bounds === undefined) {
      setSelectedTimeframe(undefined);
    } else {
      setSelectedTimeframe({labels, bounds});
    }
  };

  // Create filtered profile source when selection exists
  const filteredProfileSource = useMemo(() => {
    if (selectedTimeframe == null) return null;
    return createFilteredProfileSource(profileSource, selectedTimeframe);
  }, [profileSource, selectedTimeframe]);

  // Query flamechart data only when a strip selection exists
  const {
    isLoading: flamechartLoading,
    response: flamechartResponse,
    error: flamechartError,
  } = useQuery(
    queryClient,
    filteredProfileSource ?? profileSource,
    QueryRequest_ReportType.FLAMECHART,
    {
      skip: selectedTimeframe == null || filteredProfileSource == null,
    }
  );

  const flamechartArrow =
    flamechartResponse?.report.oneofKind === 'flamegraphArrow'
      ? flamechartResponse.report.flamegraphArrow
      : undefined;
  const flamechartTotal = flamechartResponse != null ? BigInt(flamechartResponse.total) : total;
  const flamechartFiltered =
    flamechartResponse != null ? BigInt(flamechartResponse.filtered) : filtered;

  // Get time bounds from profile source for the strips
  const timeBounds = boundsFromProfileSource(profileSource);

  // Transform samples data for SamplesStrip component
  const stripsData = useMemo(() => {
    if (samplesData?.series == null) return {cpus: [], data: [], stepMs: 0};

    const cpus = samplesData.series.map(s => s.labelset);
    const data = samplesData.series.map(s => s.data);

    const stepMs = samplesData.stepMs ?? 0;

    return {cpus, data, stepMs};
  }, [samplesData?.series, samplesData?.stepMs]);

  const {isValid, isNonDelta, isDurationTooLong} = validateFlameChartQuery(
    profileSource as MergedProfileSource
  );

  if (!isValid) {
    const message = isNonDelta
      ? 'To use the Flame chart, please switch to a Delta profile.'
      : isDurationTooLong
      ? 'Flame chart is unavailable for queries longer than one minute. Try reducing the time range to one minute or selecting a point in the metrics graph.'
      : 'Flame chart is unavailable for this query.';
    return (
      <div className="flex flex-col justify-center p-10 text-center gap-6 text-sm">{message}</div>
    );
  }

  const hasDimension = (flamechartDimension ?? []).length > 0;

  if (!hasDimension) {
    return (
      <div className="flex justify-center items-center py-10 text-gray-500 dark:text-gray-400 text-sm">
        Select a label in the &quot;Samples group by&quot; dropdown above to view the samples
        strips.
      </div>
    );
  }

  if (samplesData?.loading === true) {
    return <>{loader}</>;
  }

  return (
    <div>
      {/* Samples Strips - rendered above flamechart */}
      {stripsData.cpus.length > 0 && stripsData.data.length > 0 && (
        <div className="mb-2">
          <SamplesStrip
            cpus={stripsData.cpus}
            data={stripsData.data}
            selectedTimeframe={selectedTimeframe}
            onSelectedTimeframe={handleSelectedTimeframe}
            width={width}
            bounds={[Number(timeBounds[0] / 1_000_000n), Number(timeBounds[1] / 1_000_000n)]}
            stepMs={stripsData.stepMs}
          />
        </div>
      )}

      {/* Flamegraph visualization - only shown when a time range is selected in the strips */}
      {selectedTimeframe != null && filteredProfileSource != null ? (
        <ProfileFlameGraph
          arrow={flamechartArrow}
          loading={flamechartLoading}
          error={flamechartError}
          profileSource={filteredProfileSource}
          width={width}
          total={flamechartTotal}
          filtered={flamechartFiltered}
          profileType={profileType}
          isHalfScreen={isHalfScreen}
          metadataMappingFiles={metadataMappingFiles}
          metadataLoading={metadataLoading}
          isFlameChart={true}
          curPathArrow={[]}
          setNewCurPathArrow={() => {}}
        />
      ) : (
        <div className="flex justify-center items-center py-10 text-gray-500 dark:text-gray-400 text-sm">
          Select a time range in the samples strips above to view the flamechart.
        </div>
      )}
    </div>
  );
};

export default ProfileFlameChart;
