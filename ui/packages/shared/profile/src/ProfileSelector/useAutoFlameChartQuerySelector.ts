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

import {Dispatch, SetStateAction, useEffect, useMemo, useRef} from 'react';

import {QueryServiceClient} from '@parca/client';
import {DateTimeRange} from '@parca/components';
import {ProfileType} from '@parca/parser';

import {useLabelNames} from '../hooks/useLabels';

interface UseAutoFlameChartQuerySelectorProps {
  queryClient: QueryServiceClient;
  dashboardItems: string[];
  profileType: ProfileType;
  timeRange: DateTimeRange;
  comparing: boolean;
  loading: boolean;
  setTimeRangeSelection: Dispatch<SetStateAction<DateTimeRange>>;
  setDraftTimeRange: (from: number, to: number, timeSelection: string) => void;
  setDraftSumBy: (sumBy: string[] | undefined) => void;
  commitDraft: (refreshedTimeRange?: {from: number; to: number; timeSelection: string}) => void;
}

export const useAutoFlameChartQuerySelector = ({
  queryClient,
  dashboardItems,
  profileType,
  timeRange,
  comparing,
  loading,
  setTimeRangeSelection,
  setDraftTimeRange,
  setDraftSumBy,
  commitDraft,
}: UseAutoFlameChartQuerySelectorProps): void => {
  const previousDashboardItems = useRef<string[]>([]);
  const hasAutoConfigured = useRef(false);

  const wasFlameChartAdded =
    dashboardItems.includes('flamechart') &&
    !previousDashboardItems.current.includes('flamechart') &&
    !hasAutoConfigured.current;

  const wasFlameChartRemoved =
    !dashboardItems.includes('flamechart') && previousDashboardItems.current.includes('flamechart');

  if (wasFlameChartRemoved) {
    hasAutoConfigured.current = false;
    previousDashboardItems.current = dashboardItems;
  }

  const {result, loading: labelsLoading} = useLabelNames(
    queryClient,
    profileType.toString(),
    timeRange.getFromMs(),
    timeRange.getToMs()
  );

  const availableLabels = useMemo(() => {
    return result.response?.labelNames ?? [];
  }, [result]);

  const optimalSumBy = useMemo(() => {
    const preferredLabels = ['node', 'cpu', 'thread_id'];
    const availablePreferred = preferredLabels.filter(label => availableLabels.includes(label));

    if (availablePreferred.length > 0) {
      return availablePreferred;
    }

    return undefined;
  }, [availableLabels]);

  const oneMinuteRange = useMemo(() => {
    return DateTimeRange.fromRangeKey('relative:minute|1');
  }, []);

  useEffect(() => {
    if (!wasFlameChartAdded || loading || labelsLoading) {
      return;
    }

    hasAutoConfigured.current = true;

    const timeSelection = oneMinuteRange.getRangeKey();
    const fromMs = oneMinuteRange.getFromMs(true);
    const toMs = oneMinuteRange.getToMs(true);

    setTimeRangeSelection(oneMinuteRange);

    setDraftTimeRange(fromMs, toMs, timeSelection);
    setDraftSumBy(optimalSumBy);

    commitDraft({
      from: fromMs,
      to: toMs,
      timeSelection,
    });

    previousDashboardItems.current = dashboardItems;
  }, [
    wasFlameChartAdded,
    loading,
    labelsLoading,
    comparing,
    optimalSumBy,
    oneMinuteRange,
    dashboardItems,
    setTimeRangeSelection,
    setDraftTimeRange,
    setDraftSumBy,
    commitDraft,
  ]);
};
