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

import {useCallback, useEffect, useMemo, useState} from 'react';

import {useQueryState as useNuqsQueryState, useQueryStates} from 'nuqs';

import {DateTimeRange, useParcaContext} from '@parca/components';
import {Query} from '@parca/parser';

import {QuerySelection} from '../ProfileSelector';
import {ProfileSelection, ProfileSelectionFromParams, ProfileSource} from '../ProfileSource';
import {useResetFlameGraphState} from '../ProfileView/hooks/useResetFlameGraphState';
import {useResetStateOnProfileTypeChange} from '../ProfileView/hooks/useResetStateOnProfileTypeChange';
import {DEFAULT_EMPTY_SUM_BY, sumByToParam, useSumBy, useSumByFromParams} from '../useSumBy';
import {commaArrayParam, stringParam} from './urlParsers';

interface UseQueryStateOptions {
  suffix?: '_a' | '_b'; // For comparison mode
  defaultExpression?: string;
  defaultTimeSelection?: string; // Should be in format like 'relative:hour|1' or 'absolute:...'
  defaultFrom?: number;
  defaultTo?: number;
  comparing?: boolean; // If true, don't auto-select for delta profiles
  onProfileTypeChange?: () => void; // Called when profile type changes on commit, after reset
}

interface UseQueryStateReturn {
  // Current committed state (from URL)
  querySelection: QuerySelection;

  // Draft state (local changes not yet committed)
  draftSelection: QuerySelection;

  // Draft setters (update local state only)
  setDraftExpression: (expression: string) => void;
  setDraftTimeRange: (from: number, to: number, timeSelection: string) => void;
  setDraftSumBy: (sumBy: string[] | undefined) => void;
  setDraftProfileName: (profileName: string) => void;
  setDraftMatchers: (matchers: string) => void;

  // Commit function
  commitDraft: (refreshedTimeRange?: {from: number; to: number; timeSelection: string}) => void;

  // ProfileSelection state (separate from QuerySelection)
  profileSelection: ProfileSelection | null;

  // ProfileSource derived from ProfileSelection
  profileSource: ProfileSource | null;

  // ProfileSelection setter (auto-commits to URL)
  setProfileSelection: (mergeFrom: bigint, mergeTo: bigint, query: Query) => void;

  // Loading state for sumBy computation
  sumByLoading: boolean;

  // draft parsed query
  draftParsedQuery: Query | null;

  // parsed query
  parsedQuery: Query | null;

  setExpressionParam: (value: string | null) => void;
  setSumByParam: (value: string | null) => void;
  setGroupByParam: (value: string[] | null) => void;
}

export const useQueryState = (options: UseQueryStateOptions = {}): UseQueryStateReturn => {
  const {queryServiceClient: queryClient} = useParcaContext();
  const {
    suffix = '',
    defaultExpression = '',
    defaultTimeSelection = 'relative:minute|15', // Default to 15 minutes relative
    defaultFrom,
    defaultTo,
    comparing = false,
    onProfileTypeChange,
  } = options;

  const resetFlameGraphState = useResetFlameGraphState();
  const resetStateOnProfileTypeChange = useResetStateOnProfileTypeChange();

  // URL state hooks with appropriate suffixes via useQueryStates
  const [queryParams, setQueryParams] = useQueryStates(
    {
      expression: stringParam,
      from: stringParam,
      to: stringParam,
      time_selection: stringParam,
      sum_by: stringParam,
      merge_from: stringParam,
      merge_to: stringParam,
      selection: stringParam,
    },
    {
      history: 'replace',
      urlKeys: {
        expression: `expression${suffix}`,
        from: `from${suffix}`,
        to: `to${suffix}`,
        time_selection: `time_selection${suffix}`,
        sum_by: `sum_by${suffix}`,
        merge_from: `merge_from${suffix}`,
        merge_to: `merge_to${suffix}`,
        selection: `selection${suffix}`,
      },
    }
  );

  const expression = queryParams.expression ?? defaultExpression;
  const from = queryParams.from ?? defaultFrom?.toString();
  const to = queryParams.to ?? defaultTo?.toString();
  const timeSelection = queryParams.time_selection ?? defaultTimeSelection;
  const sumByParam = queryParams.sum_by;
  const mergeFrom = queryParams.merge_from;
  const mergeTo = queryParams.merge_to;
  const selectionParam = queryParams.selection;

  // Individual setters for direct access
  const setExpressionState = useCallback(
    (val: string | null) => void setQueryParams({expression: val}),
    [setQueryParams]
  );
  const setSumByParam = useCallback(
    (val: string | null) => void setQueryParams({sum_by: val}),
    [setQueryParams]
  );

  const [, setRawGroupByParam] = useNuqsQueryState('group_by', commaArrayParam);
  const setGroupByParam = useCallback(
    (val: string[] | null) => {
      void setRawGroupByParam(val);
    },
    [setRawGroupByParam]
  );

  // Parse sumBy from URL parameter format
  const sumBy = useSumByFromParams(sumByParam ?? undefined);

  // Draft state management
  const [draftExpression, setDraftExpression] = useState<string>(expression ?? defaultExpression);
  const [draftFrom, setDraftFrom] = useState<string>(from ?? defaultFrom?.toString() ?? '');
  const [draftTo, setDraftTo] = useState<string>(to ?? defaultTo?.toString() ?? '');
  const [draftTimeSelection, setDraftTimeSelection] = useState<string>(
    timeSelection ?? defaultTimeSelection
  );
  // Parse the draft query to extract profile information
  const draftQuery = useMemo(() => {
    try {
      return Query.parse(draftExpression ?? '');
    } catch (error) {
      console.warn('Failed to parse draft expression', {
        expression: draftExpression,
        error: error instanceof Error ? error.message : String(error),
      });
      return Query.parse('');
    }
  }, [draftExpression]);

  const query = useMemo(() => {
    try {
      return Query.parse(expression ?? '');
    } catch (error) {
      console.warn('Failed to parse expression', {
        expression,
        error: error instanceof Error ? error.message : String(error),
      });
      return Query.parse('');
    }
  }, [expression]);
  const draftProfileType = useMemo(() => draftQuery.profileType(), [draftQuery]);
  const draftProfileName = useMemo(() => draftQuery.profileName(), [draftQuery]);
  const profileType = useMemo(() => query.profileType(), [query]);

  // Compute draft time range for label fetching
  const draftTimeRange = useMemo(() => {
    return DateTimeRange.fromRangeKey(
      draftTimeSelection ?? defaultTimeSelection,
      draftFrom !== '' ? parseInt(draftFrom) : defaultFrom,
      draftTo !== '' ? parseInt(draftTo) : defaultTo
    );
  }, [draftTimeSelection, draftFrom, draftTo, defaultTimeSelection, defaultFrom, defaultTo]);
  // Use combined sumBy hook for fetching labels and computing defaults (based on committed state)
  const {
    sumBy: computedSumByFromURL,
    isLoading: sumBySelectionLoading,
    draftSumBy,
    setDraftSumBy,
    isDraftSumByLoading,
  } = useSumBy(
    queryClient,
    profileType?.profileName !== '' ? profileType : draftProfileType,
    draftTimeRange,
    draftProfileType,
    draftTimeRange,
    sumBy
  );

  // Sync draft state with URL state when URL changes externally
  useEffect(() => {
    setDraftExpression(expression ?? defaultExpression);
  }, [expression, defaultExpression]);

  useEffect(() => {
    setDraftFrom(from ?? defaultFrom?.toString() ?? '');
  }, [from, defaultFrom]);

  useEffect(() => {
    setDraftTo(to ?? defaultTo?.toString() ?? '');
  }, [to, defaultTo]);

  useEffect(() => {
    setDraftTimeSelection(timeSelection ?? defaultTimeSelection);
  }, [timeSelection, defaultTimeSelection]);

  useEffect(() => {
    setDraftSumBy(sumBy);
  }, [sumBy, setDraftSumBy]);

  // Sync computed sumBy to URL if URL doesn't already have a value
  // to ensure the shared URL can always pick it up.
  useEffect(() => {
    if (sumByParam === null && computedSumByFromURL !== undefined && !sumBySelectionLoading) {
      void setSumByParam(sumByToParam(computedSumByFromURL));
    }
  }, [sumByParam, computedSumByFromURL, sumBySelectionLoading, setSumByParam]);

  // Construct the QuerySelection object (committed state from URL)
  const querySelection: QuerySelection = useMemo(() => {
    const range = DateTimeRange.fromRangeKey(
      timeSelection ?? defaultTimeSelection,
      from != null && from !== '' ? parseInt(from) : defaultFrom,
      to != null && to !== '' ? parseInt(to) : defaultTo
    );

    return {
      expression: expression ?? defaultExpression,
      from: range.getFromMs(),
      to: range.getToMs(),
      timeSelection: range.getRangeKey(),
      sumBy: computedSumByFromURL,
      ...(mergeFrom != null && mergeFrom !== '' && mergeTo != null && mergeTo !== ''
        ? {mergeFrom, mergeTo}
        : {}),
    };
  }, [
    expression,
    from,
    to,
    timeSelection,
    computedSumByFromURL,
    mergeFrom,
    mergeTo,
    defaultExpression,
    defaultTimeSelection,
    defaultFrom,
    defaultTo,
  ]);

  // Construct the draft QuerySelection object (local draft state)
  const draftSelection: QuerySelection = useMemo(() => {
    const isDelta = draftProfileType.delta;
    const draftMergeFrom = isDelta
      ? (BigInt(draftTimeRange.getFromMs()) * 1_000_000n).toString()
      : undefined;
    const draftMergeTo = isDelta
      ? (BigInt(draftTimeRange.getToMs()) * 1_000_000n).toString()
      : undefined;

    const finalSumBy = draftSumBy ?? computedSumByFromURL;
    return {
      expression: draftExpression ?? defaultExpression,
      from: draftTimeRange.getFromMs(),
      to: draftTimeRange.getToMs(),
      timeSelection: draftTimeRange.getRangeKey(),
      sumBy: finalSumBy, // Use draft if set, otherwise fallback to computed
      ...(draftMergeFrom !== undefined &&
      draftMergeFrom !== '' &&
      draftMergeTo !== undefined &&
      draftMergeTo !== ''
        ? {mergeFrom: draftMergeFrom, mergeTo: draftMergeTo}
        : {}),
    };
  }, [
    draftExpression,
    draftTimeRange,
    draftSumBy,
    computedSumByFromURL,
    draftProfileType.delta,
    defaultExpression,
  ]);

  // Compute ProfileSelection from URL params
  const profileSelection = useMemo<ProfileSelection | null>(() => {
    return ProfileSelectionFromParams(
      mergeFrom ?? undefined,
      mergeTo ?? undefined,
      selectionParam ?? undefined
    );
  }, [mergeFrom, mergeTo, selectionParam]);

  // Compute ProfileSource from ProfileSelection
  const profileSource = useMemo<ProfileSource | null>(() => {
    if (profileSelection === null) return null;
    return profileSelection.ProfileSource();
  }, [profileSelection]);

  // Commit draft changes to URL
  // Optional refreshedTimeRange parameter allows re-evaluating relative time ranges (e.g., "last 15 minutes")
  // to the current moment when the Search button is clicked
  // Optional expression parameter allows updating the expression before committing
  const commitDraft = useCallback(
    (
      refreshedTimeRange?: {from: number; to: number; timeSelection: string},
      expression?: string
    ) => {
      // Use provided expression or current draft expression
      const finalExpression = expression ?? draftExpression;

      // Update draft state with new expression if provided
      if (expression !== undefined) {
        setDraftExpression(expression);
      }

      // Calculate the actual from/to values from draftSelection if not provided
      const calculatedFrom = draftSelection.from.toString();
      const calculatedTo = draftSelection.to.toString();

      const finalFrom =
        refreshedTimeRange?.from?.toString() ?? (draftFrom !== '' ? draftFrom : calculatedFrom);
      const finalTo =
        refreshedTimeRange?.to?.toString() ?? (draftTo !== '' ? draftTo : calculatedTo);
      const finalTimeSelection = refreshedTimeRange?.timeSelection ?? draftTimeSelection;

      // Update draft state with refreshed time range if provided
      if (refreshedTimeRange?.from !== undefined) {
        setDraftFrom(finalFrom);
      }
      if (refreshedTimeRange?.to !== undefined) {
        setDraftTo(finalTo);
      }
      if (refreshedTimeRange?.timeSelection !== undefined) {
        setDraftTimeSelection(finalTimeSelection);
      }

      // Auto-calculate merge parameters for delta profiles
      const finalQuery = Query.parse(finalExpression);
      const isDelta = finalQuery.profileType().delta;

      const sumByValue = isDelta ? sumByToParam(draftSumBy) : sumByToParam(DEFAULT_EMPTY_SUM_BY);
      let mergeFromValue: string | null = null;
      let mergeToValue: string | null = null;
      let selectionValue: string | null = null;

      if (isDelta && finalFrom !== '' && finalTo !== '') {
        const fromMs = parseInt(finalFrom);
        const toMs = parseInt(finalTo);
        mergeFromValue = (BigInt(fromMs) * 1_000_000n).toString();
        mergeToValue = (BigInt(toMs) * 1_000_000n).toString();

        if (!comparing) {
          selectionValue = finalExpression;
        }
      }

      // Atomic URL update with all params at once
      void setQueryParams({
        expression: finalExpression,
        from: finalFrom,
        to: finalTo,
        time_selection: finalTimeSelection,
        sum_by: sumByValue,
        merge_from: mergeFromValue,
        merge_to: mergeToValue,
        selection: selectionValue,
      });

      resetFlameGraphState();
      if (
        draftProfileType.toString() !==
        Query.parse(querySelection.expression).profileType().toString()
      ) {
        resetStateOnProfileTypeChange();
        onProfileTypeChange?.();
      }
    },
    [
      draftExpression,
      draftFrom,
      draftTo,
      draftTimeSelection,
      draftSumBy,
      draftSelection.from,
      draftSelection.to,
      comparing,
      setQueryParams,
      resetFlameGraphState,
      resetStateOnProfileTypeChange,
      onProfileTypeChange,
      draftProfileType,
      querySelection.expression,
    ]
  );

  const setDraftTimeRange = useCallback(
    (newFrom: number, newTo: number, newTimeSelection: string) => {
      setDraftFrom(newFrom.toString());
      setDraftTo(newTo.toString());
      setDraftTimeSelection(newTimeSelection);
    },
    []
  );

  const setDraftSumByCallback = useCallback(
    (newSumBy: string[] | undefined) => {
      setDraftSumBy(newSumBy);
    },
    [setDraftSumBy]
  );

  const setDraftProfileName = useCallback(
    (newProfileName: string) => {
      if (newProfileName === '') return;

      const [newQuery, changed] = draftQuery.setProfileName(newProfileName);
      if (changed) {
        setDraftExpression(newQuery.toString());
        setDraftSumBy(undefined);
      }
    },
    [draftQuery, setDraftSumBy]
  );

  const setDraftMatchers = useCallback(
    (matchers: string) => {
      const newExpression = `${draftProfileName}{${matchers}}`;
      setDraftExpression(newExpression);
    },
    [draftProfileName]
  );

  // Set ProfileSelection (auto-commits to URL immediately)
  const setProfileSelection = useCallback(
    (mergeFrom: bigint, mergeTo: bigint, query: Query) => {
      void setQueryParams({
        selection: query.toString(),
        merge_from: mergeFrom.toString(),
        merge_to: mergeTo.toString(),
      });
    },
    [setQueryParams]
  );

  const draftParsedQuery = useMemo(() => {
    try {
      return Query.parse(draftSelection.expression ?? '');
    } catch (error) {
      console.warn('Failed to parse draft selection expression', {
        expression: draftSelection.expression,
        error: error instanceof Error ? error.message : String(error),
      });
      return Query.parse('');
    }
  }, [draftSelection.expression]);

  const parsedQuery = useMemo(() => {
    try {
      return Query.parse(querySelection.expression ?? '');
    } catch (error) {
      console.warn('Failed to parse query selection expression', {
        expression: querySelection.expression,
        error: error instanceof Error ? error.message : String(error),
      });
      return Query.parse('');
    }
  }, [querySelection.expression]);

  return {
    // Current committed state
    querySelection,

    // Draft state
    draftSelection,

    // Draft setters
    setDraftExpression,
    setDraftTimeRange,
    setDraftSumBy: setDraftSumByCallback,
    setDraftProfileName,
    setDraftMatchers,

    // Commit function
    commitDraft,

    // ProfileSelection state
    profileSelection,
    profileSource,
    setProfileSelection,

    // Loading state
    sumByLoading: isDraftSumByLoading || sumBySelectionLoading,

    draftParsedQuery,
    parsedQuery,

    setExpressionParam: setExpressionState,
    setSumByParam,
    setGroupByParam,
  };
};
