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

import {DateTimeRange, useParcaContext, useURLState, useURLStateBatch} from '@parca/components';
import {Query} from '@parca/parser';

import {QuerySelection} from '../ProfileSelector';
import {ProfileSelection, ProfileSelectionFromParams, ProfileSource} from '../ProfileSource';
import {sumByToParam, useSumBy, useSumByFromParams} from '../useSumBy';
import { useResetFlameGraphState } from '../ProfileView/hooks/useResetFlameGraphState';

interface UseQueryStateOptions {
  suffix?: '_a' | '_b'; // For comparison mode
  defaultExpression?: string;
  defaultTimeSelection?: string; // Should be in format like 'relative:hour|1' or 'absolute:...'
  defaultFrom?: number;
  defaultTo?: number;
  comparing?: boolean; // If true, don't auto-select for delta profiles
}

interface UseQueryStateReturn {
  // Current committed state (from URL)
  querySelection: QuerySelection;

  // Draft state (local changes not yet committed)
  draftSelection: QuerySelection;

  // Draft setters (update local state only)
  setDraftExpression: (expression: string, commit?: boolean) => void;
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
  } = options;

  const batchUpdates = useURLStateBatch();
  const resetFlameGraphState = useResetFlameGraphState();

  // URL state hooks with appropriate suffixes
  const [expression, setExpressionState] = useURLState<string>(`expression${suffix}`, {
    defaultValue: defaultExpression,
  });

  const [from, setFromState] = useURLState<string>(`from${suffix}`, {
    defaultValue: defaultFrom?.toString(),
  });

  const [to, setToState] = useURLState<string>(`to${suffix}`, {
    defaultValue: defaultTo?.toString(),
  });

  const [timeSelection, setTimeSelectionState] = useURLState<string>(`time_selection${suffix}`, {
    defaultValue: defaultTimeSelection,
  });

  const [sumByParam, setSumByParam] = useURLState<string>(`sum_by${suffix}`);

  const [mergeFrom, setMergeFromState] = useURLState<string>(`merge_from${suffix}`);
  const [mergeTo, setMergeToState] = useURLState<string>(`merge_to${suffix}`);

  // ProfileSelection URL state hooks - reuses merge_from/merge_to but adds selection
  const [selectionParam, setSelectionParam] = useURLState<string>(`selection${suffix}`);

  // Parse sumBy from URL parameter format
  const sumBy = useSumByFromParams(sumByParam);

  // Draft state management
  const [draftExpression, setDraftExpression] = useState<string>(expression ?? defaultExpression);
  const [draftFrom, setDraftFrom] = useState<string>(from ?? defaultFrom?.toString() ?? '');
  const [draftTo, setDraftTo] = useState<string>(to ?? defaultTo?.toString() ?? '');
  const [draftTimeSelection, setDraftTimeSelection] = useState<string>(
    timeSelection ?? defaultTimeSelection
  );
  const [draftSumBy, setDraftSumBy] = useState<string[] | undefined>(sumBy);

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
  }, [sumBy]);

  // Parse the draft query to extract profile information
  const draftQuery = useMemo(() => {
    try {
      return Query.parse(draftExpression ?? '');
    } catch {
      return Query.parse('');
    }
  }, [draftExpression]);

  const draftProfileType = useMemo(() => draftQuery.profileType(), [draftQuery]);
  const draftProfileName = useMemo(() => draftQuery.profileName(), [draftQuery]);

  // Compute draft time range for label fetching
  const draftTimeRange = useMemo(() => {
    return DateTimeRange.fromRangeKey(
      draftTimeSelection ?? defaultTimeSelection,
      draftFrom !== '' ? parseInt(draftFrom) : defaultFrom,
      draftTo !== '' ? parseInt(draftTo) : defaultTo
    );
  }, [draftTimeSelection, draftFrom, draftTo, defaultTimeSelection, defaultFrom, defaultTo]);
  // Use combined sumBy hook for fetching labels and computing defaults (based on committed state)
  const {sumBy: computedSumByFromURL, isLoading: sumBySelectionLoading} = useSumBy(
    queryClient,
    draftProfileType,
    draftTimeRange,
    sumBy
  );

  // Construct the QuerySelection object (committed state from URL)
  const querySelection: QuerySelection = useMemo(() => {
    const range = DateTimeRange.fromRangeKey(
      timeSelection ?? defaultTimeSelection,
      from !== undefined && from !== '' ? parseInt(from) : defaultFrom,
      to !== undefined && to !== '' ? parseInt(to) : defaultTo
    );

    return {
      expression: expression ?? defaultExpression,
      from: range.getFromMs(),
      to: range.getToMs(),
      timeSelection: range.getRangeKey(),
      sumBy: computedSumByFromURL,
      ...(mergeFrom !== undefined && mergeFrom !== '' && mergeTo !== undefined && mergeTo !== ''
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
    return ProfileSelectionFromParams(mergeFrom, mergeTo, selectionParam);
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
      batchUpdates(() => {
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

        setExpressionState(finalExpression);
        setFromState(finalFrom);
        setToState(finalTo);
        setTimeSelectionState(finalTimeSelection);
        setSumByParam(sumByToParam(draftSumBy));

        // Auto-calculate merge parameters for delta profiles
        // Parse the final expression to check if it's a delta profile
        const finalQuery = Query.parse(finalExpression);
        const isDelta = finalQuery.profileType().delta;

        if (isDelta && finalFrom !== '' && finalTo !== '') {
          const fromMs = parseInt(finalFrom);
          const toMs = parseInt(finalTo);
          setMergeFromState((BigInt(fromMs) * 1_000_000n).toString());
          setMergeToState((BigInt(toMs) * 1_000_000n).toString());

          // Auto-select the time range for delta profiles (but not in compare mode)
          // This applies both on initial load AND when Search is clicked
          // The selection will use the final expression and the updated time range
          if (!comparing) {
            setSelectionParam(finalExpression);
          } else {
            setSelectionParam(undefined);
          }
        } else {
          setMergeFromState(undefined);
          setMergeToState(undefined);
          // Clear ProfileSelection for non-delta profiles
          setSelectionParam(undefined);
        }
        resetFlameGraphState();
      });
    },
    [
      batchUpdates,
      draftExpression,
      draftFrom,
      draftTo,
      draftTimeSelection,
      draftSumBy,
      draftSelection.from,
      draftSelection.to,
      comparing,
      setExpressionState,
      setFromState,
      setToState,
      setTimeSelectionState,
      setSumByParam,
      setMergeFromState,
      setMergeToState,
      setSelectionParam,
      resetFlameGraphState,
    ]
  );

  // Draft setters (update local state only, or commit directly if specified)
  const setDraftExpressionCallback = useCallback(
    (newExpression: string, commit = false) => {
      if (commit) {
        // Commit with the new expression, which will also update merge params and selection
        commitDraft(undefined, newExpression);
      } else {
        // Only update draft state
        setDraftExpression(newExpression);
      }
    },
    [commitDraft]
  );

  const setDraftTimeRange = useCallback(
    (newFrom: number, newTo: number, newTimeSelection: string) => {
      setDraftFrom(newFrom.toString());
      setDraftTo(newTo.toString());
      setDraftTimeSelection(newTimeSelection);
    },
    []
  );

  const setDraftSumByCallback = useCallback((newSumBy: string[] | undefined) => {
    setDraftSumBy(newSumBy);
  }, []);

  const setDraftProfileName = useCallback(
    (newProfileName: string) => {
      if (newProfileName === '') return;

      const [newQuery, changed] = draftQuery.setProfileName(newProfileName);
      if (changed) {
        setDraftExpression(newQuery.toString());
      }
    },
    [draftQuery]
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
      batchUpdates(() => {
        setSelectionParam(query.toString());
        setMergeFromState(mergeFrom.toString());
        setMergeToState(mergeTo.toString());
      });
    },
    [batchUpdates, setSelectionParam, setMergeFromState, setMergeToState]
  );

  return {
    // Current committed state
    querySelection,

    // Draft state
    draftSelection,

    // Draft setters
    setDraftExpression: setDraftExpressionCallback,
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
    sumByLoading: sumBySelectionLoading,
  };
};
