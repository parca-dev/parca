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

import {ProfileTypesResponse} from '@parca/client';
import {selectAutoQuery, setAutoQuery, useAppDispatch, useAppSelector} from '@parca/store';
import {type NavigateFunction} from '@parca/utilities';

import {ProfileSelectionFromParams, SuffixParams} from '..';
import {QuerySelection} from '../ProfileSelector';
import {constructProfileName} from '../ProfileTypeSelector';

interface Props {
  selectedProfileName: string;
  profileTypesData: ProfileTypesResponse | undefined;
  setProfileName: (name: string) => void;
  setQueryExpression: () => void;
  querySelection: QuerySelection;
  navigateTo: NavigateFunction;
  loading: boolean;
}

export const useAutoQuerySelector = ({
  selectedProfileName,
  profileTypesData,
  setProfileName,
  setQueryExpression,
  querySelection,
  navigateTo,
  loading,
}: Props): void => {
  const autoQuery = useAppSelector(selectAutoQuery);
  const dispatch = useAppDispatch();
  const queryParams = new URLSearchParams(location.search);
  const compareA = queryParams.get('compare_a');
  const compareB = queryParams.get('compare_b');
  const comparing = compareA === 'true' || compareB === 'true';
  const expressionA = queryParams.get('expression_a');
  const expressionB = queryParams.get('expression_b');

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
    if (comparing && expressionA !== null && expressionA !== undefined && expressionB === null && !hasSetupCompareMode.current) {
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

      const sumBy = queryA.sumBy?.join(',');

      let compareQuery: Record<string, string> = {
        compare_a: 'true',
        expression_a: queryA.expression,
        from_a: queryA.from.toString(),
        to_a: queryA.to.toString(),
        time_selection_a: queryA.timeSelection,

        compare_b: 'true',
        expression_b: queryA.expression,
        from_b: queryA.from.toString(),
        to_b: queryA.to.toString(),
        time_selection_b: queryA.timeSelection,
      };

      if (sumBy != null) {
        compareQuery.sum_by_a = sumBy;
        compareQuery.sum_by_b = sumBy;
      }

      if (profileA != null) {
        compareQuery = {
          ...SuffixParams(profileA.HistoryParams(), '_a'),
          ...compareQuery,
        };
      }

      hasSetupCompareMode.current = true;
      void navigateTo('/', {
        ...compareQuery,
        search_string: '',
        dashboard_items: ['flamegraph'],
      });
    }
  }, [comparing, querySelection, navigateTo, expressionA, expressionB, dispatch, loading]);

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
      let profileType = profileTypesData.types.find(
        type => type.name === 'parca_agent' && type.delta
      );
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
