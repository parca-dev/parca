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

import {ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {
  DateTimeRange,
  IconButton,
  useGrpcMetadata,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {CloseIcon} from '@parca/icons';
import {Query} from '@parca/parser';
import {type NavigateFunction} from '@parca/utilities';

import {ProfileSelection} from '..';
import {useLabelNames} from '../MatchersInput/index';
import {useMetricsGraphDimensions} from '../MetricsGraph/useMetricsGraphDimensions';
import {useDefaultSumBy, useSumBySelection} from '../useSumBy';
import {MetricsGraphSection} from './MetricsGraphSection';
import {QueryControls} from './QueryControls';
import {useAutoQuerySelector} from './useAutoQuerySelector';

export interface QuerySelection {
  expression: string;
  from: number;
  to: number;
  timeSelection: string;
  sumBy?: string[];
  mergeFrom?: number;
  mergeTo?: number;
}

interface ProfileSelectorFeatures {
  showMetricsGraph: boolean;
  showSumBySelector?: boolean;
  showProfileTypeSelector?: boolean;
  disableExplorativeQuerying?: boolean;
  disableProfileTypesDropdown?: boolean;
}

export interface UtilizationMetrics {
  timestamp: number;
  value: number;
  resource: {
    [key: string]: string;
  };
  attributes: {
    [key: string]: string;
  };
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
  setDisplayHideMetricsGraphButton: Dispatch<SetStateAction<boolean>>;
  suffix?: string;
  utilizationMetrics?: UtilizationMetrics[];
  utilizationMetricsLoading?: boolean;
}

export interface IProfileTypesResult {
  loading: boolean;
  data?: ProfileTypesResponse;
  error?: RpcError;
}

export const useProfileTypes = (client: QueryServiceClient): IProfileTypesResult => {
  const [result, setResult] = useState<ProfileTypesResponse | undefined>(undefined);
  const [error, setError] = useState<RpcError | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const metadata = useGrpcMetadata();

  useEffect(() => {
    if (!loading) {
      return;
    }
    const call = client.profileTypes({}, {meta: metadata});
    call.response
      .then(response => setResult(response))
      .catch(error => setError(error))
      .finally(() => setLoading(false));
  }, [client, metadata, loading]);

  return {loading, data: result, error};
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
  disableExplorativeQuerying = false,
  setDisplayHideMetricsGraphButton,
  utilizationMetrics,
  utilizationMetricsLoading,
}: ProfileSelectorProps): JSX.Element => {
  const {
    loading: profileTypesLoading,
    data: profileTypesData,
    error,
  } = useProfileTypes(queryClient);
  const {heightStyle} = useMetricsGraphDimensions(comparing);
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

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient, profileType.toString());
  const {loading: selectedLabelNamesLoading, result: selectedLabelNamesResult} = useLabelNames(
    queryClient,
    selectedProfileType.toString()
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
          mergeFrom: from,
          mergeTo: to,
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
          showProfileTypeSelector={showProfileTypeSelector}
          showSumBySelector={showSumBySelector}
          disableExplorativeQuerying={disableExplorativeQuerying}
          profileTypesData={profileTypesData}
          profileTypesLoading={profileTypesLoading}
          selectedProfileName={selectedProfileName}
          setProfileName={setProfileName}
          setMatchersString={setMatchersString}
          setQueryExpression={setQueryExpression}
          query={query}
          queryBrowserRef={queryBrowserRef}
          timeRangeSelection={timeRangeSelection}
          setTimeRangeSelection={setTimeRangeSelection}
          searchDisabled={searchDisabled}
          queryBrowserMode={queryBrowserMode as string}
          setQueryBrowserMode={setQueryBrowserMode}
          advancedModeForQueryBrowser={advancedModeForQueryBrowser}
          setAdvancedModeForQueryBrowser={setAdvancedModeForQueryBrowser}
          queryClient={queryClient}
          sumByRef={sumByRef}
          labels={labels}
          sumBySelection={sumBySelection ?? []}
          setUserSumBySelection={setUserSumBySelection}
          profileType={profileType}
          profileTypesError={error}
        />
        {comparing && (
          <div>
            <IconButton onClick={() => closeProfile()} icon={<CloseIcon />} />
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
        utilizationMetrics={utilizationMetrics}
        utilizationMetricsLoading={utilizationMetricsLoading}
      />
    </>
  );
};

export default ProfileSelector;
