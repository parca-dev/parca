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

import {useEffect, useRef} from 'react';

import {useQueryStates} from 'nuqs';

import {ProfileTypesResponse} from '@parca/client';
import {selectAutoQuery, setAutoQuery, useAppDispatch, useAppSelector} from '@parca/store';

import {ProfileSelectionFromParams} from '..';
import {QuerySelection} from '../ProfileSelector';
import {constructProfileName} from '../ProfileTypeSelector';
import {boolParam, stringParam} from '../hooks/urlParsers';
import {useDashboardItems} from '../hooks/useDashboardItems';

interface Props {
  selectedProfileName: string;
  profileTypesData: ProfileTypesResponse | undefined;
  setProfileName: (name: string) => void;
  setQueryExpression: () => void;
  querySelection: QuerySelection;
  loading: boolean;
  defaultProfileType?: string;
}

export const useAutoQuerySelector = ({
  selectedProfileName,
  profileTypesData,
  setProfileName,
  setQueryExpression,
  querySelection,
  loading,
  defaultProfileType,
}: Props): void => {
  const autoQuery = useAppSelector(selectAutoQuery);
  const dispatch = useAppDispatch();

  const {setDashboardItems} = useDashboardItems();

  const [compareState, setCompareParams] = useQueryStates(
    {
      compare_a: boolParam,
      compare_b: boolParam,
      expression_a: stringParam,
      from_a: stringParam,
      to_a: stringParam,
      time_selection_a: stringParam,
      sum_by_a: stringParam,
      merge_from_a: stringParam,
      merge_to_a: stringParam,
      selection_a: stringParam,
      expression_b: stringParam,
      from_b: stringParam,
      to_b: stringParam,
      time_selection_b: stringParam,
      sum_by_b: stringParam,
      search_string: stringParam,
    },
    {history: 'replace'}
  );

  // Read compare params through nuqs (not location.search) to stay in sync
  const comparing = compareState.compare_a === true || compareState.compare_b === true;
  const expressionA = compareState.expression_a;
  const expressionB = compareState.expression_b;

  // Track if we've already set up compare mode to prevent infinite loops
  const hasSetupCompareMode = useRef(false);

  useEffect(() => {
    if (loading) {
      return;
    }

    // Only run this effect if:
    // 1. We're in compare mode
    // 2. expressionA exists
    // 3. expressionB doesn't exist yet (meaning we need to set it up)
    // 4. We haven't already set it up in this session
    if (comparing && expressionA !== null && expressionB === null && !hasSetupCompareMode.current) {
      if (querySelection.expression === undefined) {
        return;
      }
      const profileA = ProfileSelectionFromParams(
        querySelection.mergeFrom?.toString(),
        querySelection.mergeTo?.toString(),
        querySelection.expression
      );
      const queryA = {
        expression: querySelection.expression,
        from: querySelection.from,
        to: querySelection.to,
        timeSelection: querySelection.timeSelection,
        sumBy: querySelection.sumBy,
      };

      const sumBy = queryA.sumBy?.join(',') ?? null;

      const mergeFromA = profileA != null ? profileA.HistoryParams().merge_from?.toString() : null;
      const mergeToA = profileA != null ? profileA.HistoryParams().merge_to?.toString() : null;
      const selectionA = profileA != null ? profileA.HistoryParams().selection?.toString() : null;

      hasSetupCompareMode.current = true;

      // Set all compare params atomically via nuqs
      void setCompareParams({
        compare_a: true,
        compare_b: true,
        expression_a: queryA.expression,
        from_a: queryA.from.toString(),
        to_a: queryA.to.toString(),
        time_selection_a: queryA.timeSelection,
        sum_by_a: sumBy,
        merge_from_a: mergeFromA,
        merge_to_a: mergeToA,
        selection_a: selectionA,
        expression_b: queryA.expression,
        from_b: queryA.from.toString(),
        to_b: queryA.to.toString(),
        time_selection_b: queryA.timeSelection,
        sum_by_b: sumBy,
        search_string: null,
      });

      setDashboardItems(['flamegraph']);
    }
  }, [
    comparing,
    querySelection,
    expressionA,
    expressionB,
    dispatch,
    loading,
    setCompareParams,
    setDashboardItems,
  ]);

  // Effect to load some initial data on load when is no selection
  useEffect(() => {
    void (async () => {
      if (selectedProfileName.length > 0) {
        return;
      }
      if (profileTypesData?.types == null || profileTypesData.types.length < 1) {
        return;
      }
      if (autoQuery === 'true') {
        // Autoquery already enabled.
        return;
      }
      dispatch(setAutoQuery('true'));

      if (defaultProfileType != null && defaultProfileType.length > 0) {
        setProfileName(defaultProfileType);
        return;
      }

      let profileType = profileTypesData.types.find(
        type => type.name === 'parca_agent' && type.sampleType === 'samples' && type.delta
      );
      if (profileType == null) {
        profileType = profileTypesData.types.find(
          type => type.name === 'go_opentelemetry_io_ebpf_profiler' && type.delta
        );
      }
      if (profileType == null) {
        profileType = profileTypesData.types.find(
          type => type.name === 'otel_profiling_agent_on_cpu' && type.delta
        );
      }
      if (profileType == null) {
        profileType = profileTypesData.types.find(
          type => type.name === 'parca_agent_cpu' && type.delta
        );
      }
      if (profileType == null) {
        profileType = profileTypesData.types.find(
          type => type.name === 'process_cpu' && type.delta
        );
      }
      if (profileType == null) {
        profileType = profileTypesData.types[0];
      }
      setProfileName(constructProfileName(profileType));
    })();
  }, [
    profileTypesData,
    selectedProfileName,
    autoQuery,
    dispatch,
    setQueryExpression,
    setProfileName,
    defaultProfileType,
  ]);

  useEffect(() => {
    void (async () => {
      if (
        autoQuery !== 'true' ||
        profileTypesData?.types == null ||
        profileTypesData.types.length < 1 ||
        selectedProfileName.length === 0 ||
        loading
      ) {
        return;
      }
      setQueryExpression();
      dispatch(setAutoQuery('false'));
    })();
  }, [
    profileTypesData,
    setQueryExpression,
    autoQuery,
    setProfileName,
    dispatch,
    selectedProfileName,
    loading,
  ]);
};
