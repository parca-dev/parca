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

import {Dispatch, SetStateAction, useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {ProfileTypesRequest, ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {
  DateTimeRange,
  IconButton,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
  useURLStateBatch,
} from '@parca/components';
import {CloseIcon} from '@parca/icons';
import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import {millisToProtoTimestamp, type NavigateFunction} from '@parca/utilities';

import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import {QueryControls} from '../QueryControls';
import {LabelsQueryProvider, useLabelsQueryProvider} from '../contexts/LabelsQueryProvider';
import {UnifiedLabelsProvider} from '../contexts/UnifiedLabelsContext';
import {useLabelNames} from '../hooks/useLabels';
import {useQueryState} from '../hooks/useQueryState';
import useGrpcQuery from '../useGrpcQuery';
import {MetricsGraphSection} from './MetricsGraphSection';
import {useAutoQuerySelector} from './useAutoQuerySelector';

export interface QuerySelection {
  expression: string;
  from: number;
  to: number;
  timeSelection: string;
  sumBy?: string[];
  mergeFrom?: string;
  mergeTo?: string;
}

interface ProfileSelectorFeatures {
  showMetricsGraph: boolean;
  showSumBySelector?: boolean;
  showProfileTypeSelector?: boolean;
  disableProfileTypesDropdown?: boolean;
}

interface ProfileSelectorProps extends ProfileSelectorFeatures {
  queryClient: QueryServiceClient;
  closeProfile: () => void;
  enforcedProfileName: string;
  comparing: boolean;
  navigateTo: NavigateFunction;
  setDisplayHideMetricsGraphButton?: Dispatch<SetStateAction<boolean>>;
  suffix?: '_a' | '_b'; // For comparison mode
  onSearchHook?: () => void;
}

export interface IProfileTypesResult {
  loading: boolean;
  data?: ProfileTypesResponse;
  error?: RpcError;
}

export const useProfileTypes = (
  client: QueryServiceClient,
  start?: number,
  end?: number
): IProfileTypesResult => {
  const metadata = useGrpcMetadata();
  const metadataString = useMemo(() => JSON.stringify(metadata), [metadata]);
  const request: ProfileTypesRequest = {};

  if (start != null && end != null) {
    request.start = millisToProtoTimestamp(start);
    request.end = millisToProtoTimestamp(end);
  }

  const {isLoading, data, error} = useGrpcQuery({
    key: ['profileTypes', metadataString, start, end],
    queryFn: async abort => {
      const {response} = await client.profileTypes(request, {
        meta: metadata,
        abort,
      });
      return response;
    },
  });

  return {loading: isLoading, data, error: error as RpcError};
};

const ProfileSelector = ({
  queryClient,
  closeProfile,
  enforcedProfileName,
  comparing,
  navigateTo,
  showMetricsGraph = true,
  showSumBySelector = true,
  showProfileTypeSelector = true,
  setDisplayHideMetricsGraphButton,
  suffix,
  onSearchHook,
}: ProfileSelectorProps): JSX.Element => {
  const {heightStyle} = useMetricsGraphDimensions(comparing, false);
  const {viewComponent} = useParcaContext();
  const [queryBrowserMode, setQueryBrowserMode] = useURLState('query_browser_mode');
  const batchUpdates = useURLStateBatch();

  // Use the new useQueryState hook - reads directly from URL params
  const {
    querySelection,
    draftSelection,
    setDraftExpression,
    setDraftTimeRange,
    setDraftSumBy,
    setDraftProfileName,
    setDraftMatchers,
    commitDraft,
    profileSelection,
    setProfileSelection,
    sumByLoading,
  } = useQueryState({suffix});

  // Use draft state for local state instead of committed state
  const [timeRangeSelection, setTimeRangeSelection] = useState(
    DateTimeRange.fromRangeKey(draftSelection.timeSelection, draftSelection.from, draftSelection.to)
  );

  const [queryExpressionString, setQueryExpressionString] = useState(draftSelection.expression);

  const [advancedModeForQueryBrowser, setAdvancedModeForQueryBrowser] = useState(
    queryBrowserMode === 'advanced'
  );

  // Handler to update draft when time range changes
  const handleTimeRangeChange = useCallback(
    (range: DateTimeRange) => {
      setTimeRangeSelection(range);
      setDraftTimeRange(range.getFromMs(), range.getToMs(), range.getRangeKey());
    },
    [setDraftTimeRange]
  );

  const profileType = useMemo(() => {
    return Query.parse(queryExpressionString).profileType();
  }, [queryExpressionString]);

  const from = timeRangeSelection.getFromMs();
  const to = timeRangeSelection.getToMs();

  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error,
  } = useProfileTypes(queryClient, from, to);

  const {result} = useLabelNames(queryClient, profileType.toString(), from, to);

  const labels = useMemo(() => {
    return result.response?.labelNames === undefined ? [] : result.response.labelNames;
  }, [result]);

  useEffect(() => {
    if (enforcedProfileName !== '') {
      const [q, changed] = Query.parse(querySelection.expression).setProfileName(
        enforcedProfileName
      );
      if (changed) {
        setQueryExpressionString(q.toString());
        return;
      }
    }
    setQueryExpressionString(querySelection.expression);
  }, [enforcedProfileName, querySelection.expression]);

  const enforcedProfileNameQuery = (): Query => {
    const pq = Query.parse(queryExpressionString);
    const [q] = pq.setProfileName(enforcedProfileName);
    return q;
  };

  const query =
    enforcedProfileName !== '' ? enforcedProfileNameQuery() : Query.parse(queryExpressionString);
  const selectedProfileName = query.profileName();

  const setQueryExpression = (updateTs = false): void => {
    batchUpdates(() => {
      if (onSearchHook != null) {
        onSearchHook();
      }
      // When updateTs is true, re-evaluate the time range to current values
      if (updateTs) {
        // Force re-evaluation of time range (important for relative ranges like "last 15 minutes")
        const currentFrom = timeRangeSelection.getFromMs(true);
        const currentTo = timeRangeSelection.getToMs(true);
        const currentRangeKey = timeRangeSelection.getRangeKey();
        // Commit with refreshed time range
        commitDraft({
          from: currentFrom,
          to: currentTo,
          timeSelection: currentRangeKey,
        });
      } else {
        // Commit the draft with existing values
        commitDraft();
      }
    });
  };

  const setMatchersString = (matchers: string): void => {
    // Update draft state only
    setDraftMatchers(matchers);
    setQueryExpressionString(`${selectedProfileName}{${matchers}}`);
  };

  const setProfileName = (profileName: string | undefined): void => {
    if (profileName === undefined) {
      return;
    }
    // Update draft state only
    setDraftProfileName(profileName);
    const [newQuery, changed] = query.setProfileName(profileName);
    if (changed) {
      const q = newQuery.toString();
      setQueryExpressionString(q);
    }
  };

  useEffect(() => {
    if (viewComponent !== undefined) {
      viewComponent.emitQuery(query.toString());
    }
  }, [query, viewComponent]);

  useAutoQuerySelector({
    selectedProfileName,
    profileTypesData,
    setProfileName,
    setQueryExpression,
    querySelection,
    navigateTo,
    loading: sumByLoading,
  });

  const searchDisabled =
    queryExpressionString === undefined ||
    queryExpressionString === '' ||
    queryExpressionString === '{}';

  const queryBrowserRef = useRef<HTMLDivElement>(null);
  const sumByRef = useRef(null);

  return (
    <>
      <div className="mb-2 flex">
        <LabelsQueryProvider
          setMatchersString={setMatchersString}
          runQuery={setQueryExpression}
          currentQuery={query}
          profileType={selectedProfileName ?? profileType.toString()}
          queryClient={queryClient}
          start={timeRangeSelection.getFromMs()}
          end={timeRangeSelection.getToMs()}
        >
          <LabelsSource>
            <QueryControls
              showProfileTypeSelector={showProfileTypeSelector}
              showSumBySelector={showSumBySelector}
              profileTypesData={profileTypesData}
              profileTypesLoading={profileTypesLoading}
              selectedProfileName={selectedProfileName}
              setProfileName={setProfileName}
              setMatchersString={setMatchersString}
              setQueryExpression={setQueryExpression}
              query={query}
              queryBrowserRef={queryBrowserRef}
              timeRangeSelection={timeRangeSelection}
              setTimeRangeSelection={handleTimeRangeChange}
              searchDisabled={searchDisabled}
              setQueryBrowserMode={setQueryBrowserMode}
              advancedModeForQueryBrowser={advancedModeForQueryBrowser}
              setAdvancedModeForQueryBrowser={setAdvancedModeForQueryBrowser}
              queryClient={queryClient}
              sumByRef={sumByRef}
              labels={labels}
              sumBySelection={draftSelection.sumBy ?? []}
              sumBySelectionLoading={sumByLoading}
              setUserSumBySelection={setDraftSumBy}
              profileType={profileType}
              profileTypesError={error}
              viewComponent={viewComponent}
            />
          </LabelsSource>
        </LabelsQueryProvider>
        {comparing && (
          <div>
            <IconButton
              onClick={() => closeProfile()}
              icon={<CloseIcon />}
              {...testId(TEST_IDS.COMPARE_CLOSE_BUTTON)}
            />
          </div>
        )}
      </div>
      <MetricsGraphSection
        showMetricsGraph={showMetricsGraph}
        setDisplayHideMetricsGraphButton={setDisplayHideMetricsGraphButton}
        heightStyle={heightStyle}
        querySelection={querySelection}
        profileSelection={profileSelection}
        comparing={comparing}
        sumBy={querySelection.sumBy ?? []}
        defaultSumByLoading={sumByLoading}
        queryClient={queryClient}
        queryExpressionString={queryExpressionString}
        setTimeRangeSelection={handleTimeRangeChange}
        selectQuery={commitDraft}
        setProfileSelection={setProfileSelection}
        query={query}
        setQueryExpression={setQueryExpression}
        setNewQueryExpression={setDraftExpression}
      />
    </>
  );
};

export default ProfileSelector;

const LabelsSource = ({children}: {children: React.ReactNode}): JSX.Element => {
  const {
    labelNames,
    labelValues,
    isLabelNamesLoading,
    isLabelValuesLoading,
    refetchLabelValues,
    refetchLabelNames,
    queryClient,
    setMatchersString,
    runQuery,
    start,
    end,
    profileType,
    currentQuery,
    currentLabelName,
    setCurrentLabelName,
  } = useLabelsQueryProvider();

  return (
    <UnifiedLabelsProvider
      setMatchersString={setMatchersString}
      runQuery={runQuery}
      currentQuery={currentQuery}
      profileType={profileType.toString()}
      queryClient={queryClient}
      start={start}
      end={end}
      labelNames={labelNames}
      labelValues={labelValues}
      isLabelNamesLoading={isLabelNamesLoading}
      isLabelValuesLoading={isLabelValuesLoading}
      refetchLabelValues={refetchLabelValues}
      refetchLabelNames={refetchLabelNames}
      currentLabelName={currentLabelName}
      setCurrentLabelName={setCurrentLabelName}
    >
      {children}
    </UnifiedLabelsProvider>
  );
};
