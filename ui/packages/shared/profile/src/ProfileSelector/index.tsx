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

import {Dispatch, SetStateAction, useEffect, useMemo, useRef, useState} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {ProfileTypesRequest, ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {
  DateTimeRange,
  IconButton,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {CloseIcon} from '@parca/icons';
import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import {millisToProtoTimestamp, type NavigateFunction} from '@parca/utilities';

import {ProfileSelection} from '..';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import {QueryControls} from '../QueryControls';
import {useLabelNames} from '../hooks/useLabels';
import useGrpcQuery from '../useGrpcQuery';
import {useDefaultSumBy, useSumBySelection} from '../useSumBy';
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
  disableExplorativeQuerying?: boolean;
  disableProfileTypesDropdown?: boolean;
}

interface ProfileSelectorProps extends ProfileSelectorFeatures {
  queryClient: QueryServiceClient;
  querySelection: QuerySelection;
  selectProfile: (source: ProfileSelection) => void;
  selectQuery: (query: QuerySelection) => void;
  closeProfile: () => void;
  enforcedProfileName: string;
  profileSelection: ProfileSelection | null;
  comparing: boolean;
  navigateTo: NavigateFunction;
  setDisplayHideMetricsGraphButton?: Dispatch<SetStateAction<boolean>>;
  suffix?: string;
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
  querySelection,
  selectProfile,
  selectQuery,
  closeProfile,
  enforcedProfileName,
  profileSelection,
  comparing,
  navigateTo,
  showMetricsGraph = true,
  showSumBySelector = true,
  showProfileTypeSelector = true,
  setDisplayHideMetricsGraphButton,
}: ProfileSelectorProps): JSX.Element => {
  const {heightStyle} = useMetricsGraphDimensions(comparing, false);
  const {viewComponent} = useParcaContext();
  const [queryBrowserMode, setQueryBrowserMode] = useURLState('query_browser_mode');

  const [timeRangeSelection, setTimeRangeSelection] = useState(
    DateTimeRange.fromRangeKey(querySelection.timeSelection, querySelection.from, querySelection.to)
  );

  const [queryExpressionString, setQueryExpressionString] = useState(querySelection.expression);

  const [advancedModeForQueryBrowser, setAdvancedModeForQueryBrowser] = useState(
    queryBrowserMode === 'advanced'
  );

  const profileType = useMemo(() => {
    return Query.parse(queryExpressionString).profileType();
  }, [queryExpressionString]);

  const selectedProfileType = useMemo(() => {
    return Query.parse(querySelection.expression).profileType();
  }, [querySelection.expression]);

  const from = timeRangeSelection.getFromMs();
  const to = timeRangeSelection.getToMs();

  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error,
  } = useProfileTypes(queryClient, from, to);

  const {loading: labelNamesLoading, result} = useLabelNames(
    queryClient,
    profileType.toString(),
    from,
    to
  );
  const {loading: selectedLabelNamesLoading, result: selectedLabelNamesResult} = useLabelNames(
    queryClient,
    selectedProfileType.toString(),
    from,
    to
  );

  const labels = useMemo(() => {
    return result.response?.labelNames === undefined ? [] : result.response.labelNames;
  }, [result]);

  const selectedLabels = useMemo(() => {
    return selectedLabelNamesResult.response?.labelNames === undefined
      ? []
      : selectedLabelNamesResult.response.labelNames;
  }, [selectedLabelNamesResult]);

  const [sumBySelection, setUserSumBySelection, {isLoading: sumBySelectionLoading}] =
    useSumBySelection(profileType, labelNamesLoading, labels, {
      defaultValue: querySelection.sumBy,
    });

  const {defaultSumBy, isLoading: defaultSumByLoading} = useDefaultSumBy(
    selectedProfileType,
    selectedLabelNamesLoading,
    selectedLabels
  );

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

  const setNewQueryExpression = (expr: string, updateTs = false): void => {
    const query = enforcedProfileName !== '' ? enforcedProfileNameQuery() : Query.parse(expr);
    const delta = query.profileType().delta;
    const from = timeRangeSelection.getFromMs(updateTs);
    const to = timeRangeSelection.getToMs(updateTs);
    const mergeParams = delta
      ? {
          mergeFrom: (BigInt(from) * 1_000_000n).toString(),
          mergeTo: (BigInt(to) * 1_000_000n).toString(),
        }
      : {};

    selectQuery({
      expression: expr,
      from,
      to,
      timeSelection: timeRangeSelection.getRangeKey(),
      sumBy: sumBySelection,
      ...mergeParams,
    });
  };

  const setQueryExpression = (updateTs = false): void => {
    setNewQueryExpression(query.toString(), updateTs);
  };

  const setMatchersString = (matchers: string): void => {
    const newExpressionString = `${selectedProfileName}{${matchers}}`;
    setQueryExpressionString(newExpressionString);
  };

  const setProfileName = (profileName: string | undefined): void => {
    if (profileName === undefined) {
      return;
    }
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
    querySelection: {...querySelection, sumBy: sumBySelection},
    navigateTo,
    loading: sumBySelectionLoading,
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
        <QueryControls
          queryClient={queryClient}
          query={query}
          profileType={profileType}
          timeRangeSelection={timeRangeSelection}
          setTimeRangeSelection={setTimeRangeSelection}
          setMatchersString={setMatchersString}
          setQueryExpression={setQueryExpression}
          searchDisabled={searchDisabled}
          showProfileTypeSelector={showProfileTypeSelector}
          showSumBySelector={showSumBySelector}
          showAdvancedMode={true}
          profileTypesData={profileTypesData}
          profileTypesLoading={profileTypesLoading}
          selectedProfileName={selectedProfileName}
          setProfileName={setProfileName}
          profileTypesError={error}
          viewComponent={viewComponent}
          setQueryBrowserMode={setQueryBrowserMode}
          advancedModeForQueryBrowser={advancedModeForQueryBrowser}
          setAdvancedModeForQueryBrowser={setAdvancedModeForQueryBrowser}
          queryBrowserRef={queryBrowserRef}
          labels={labels}
          sumBySelection={sumBySelection ?? []}
          sumBySelectionLoading={sumBySelectionLoading}
          setUserSumBySelection={setUserSumBySelection}
          sumByRef={sumByRef}
        />
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
        sumBy={querySelection.sumBy ?? defaultSumBy ?? []}
        defaultSumByLoading={defaultSumByLoading}
        queryClient={queryClient}
        queryExpressionString={queryExpressionString}
        setTimeRangeSelection={setTimeRangeSelection}
        selectQuery={selectQuery}
        selectProfile={selectProfile}
        query={query}
        setQueryExpression={setQueryExpression}
        setNewQueryExpression={setNewQueryExpression}
      />
    </>
  );
};

export default ProfileSelector;
