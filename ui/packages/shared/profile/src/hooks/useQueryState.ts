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
import {useResetFlameGraphState} from '../ProfileView/hooks/useResetFlameGraphState';
import {useResetStateOnProfileTypeChange} from '../ProfileView/hooks/useResetStateOnProfileTypeChange';
import {DEFAULT_EMPTY_SUM_BY, sumByToParam, useSumBy, useSumByFromParams} from '../useSumBy';

interface ViewDefaults {
  expression?: string;
  sumBy?: string[];
  groupBy?: string[];
}

interface UseQueryStateOptions {
  suffix?: '_a' | '_b'; // For comparison mode
  defaultExpression?: string;
  defaultTimeSelection?: string; // Should be in format like 'relative:hour|1' or 'absolute:...'
  defaultFrom?: number;
  defaultTo?: number;
  comparing?: boolean; // If true, don't auto-select for delta profiles
  viewDefaults?: ViewDefaults; // View-specific defaults that don't overwrite URL params
  sharedDefaults?: ViewDefaults; // Shared defaults across both comparison sides
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

  // Parsed expression components
  hasProfileType: boolean;
  profileTypeString: string;
  matchersOnly: string;
  fullExpression: string;

  // Group-by state (only for _a hook, undefined for _b)
  groupBy?: string[];
  setGroupBy?: (groupBy: string[] | undefined) => void;
  isGroupByLoading?: boolean;

  // Methods
  applyViewDefaults: () => void;
  resetQuery: () => void;
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
    viewDefaults,
    sharedDefaults,
  } = options;

  const batchUpdates = useURLStateBatch();
  const resetFlameGraphState = useResetFlameGraphState();
  const resetStateOnProfileTypeChange = useResetStateOnProfileTypeChange();

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

  // Group-by state - only enabled for _a hook (or when no suffix)
  // This ensures only one hook manages the shared group_by param in comparison mode
  const isGroupByEnabled = suffix === '' || suffix === '_a';
  const [groupByParam, setGroupByParam] = useURLState<string>('group_by', {
    enabled: isGroupByEnabled,
  });

  // Separate setters for applying view defaults with preserve-existing strategy
  const [, setExpressionWithPreserve] = useURLState<string>(`expression${suffix}`, {
    mergeStrategy: 'preserve-existing',
  });
  const [, setSumByWithPreserve] = useURLState<string>(`sum_by${suffix}`, {
    mergeStrategy: 'preserve-existing',
  });
  const [, setGroupByWithPreserve] = useURLState<string>('group_by', {
    enabled: isGroupByEnabled,
    mergeStrategy: 'preserve-existing',
  });

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
  // Parse the draft query to extract profile information
  const draftQuery = useMemo(() => {
    try {
      return Query.parse(draftExpression ?? '');
    } catch {
      return Query.parse('');
    }
  }, [draftExpression]);

  const query = useMemo(() => {
    try {
      return Query.parse(expression ?? '');
    } catch {
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
    if (sumByParam === undefined && computedSumByFromURL !== undefined && !sumBySelectionLoading) {
      setSumByParam(sumByToParam(computedSumByFromURL));
    }
  }, [sumByParam, computedSumByFromURL, sumBySelectionLoading, setSumByParam]);

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

        // Auto-calculate merge parameters for delta profiles
        // Parse the final expression to check if it's a delta profile
        const finalQuery = Query.parse(finalExpression);
        const isDelta = finalQuery.profileType().delta;
        if (isDelta) {
          setSumByParam(sumByToParam(draftSumBy));
        } else {
          setSumByParam(DEFAULT_EMPTY_SUM_BY);
        }

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
        if (
          draftProfileType.toString() !==
          Query.parse(querySelection.expression).profileType().toString()
        ) {
          resetStateOnProfileTypeChange();
        }
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
      resetStateOnProfileTypeChange,
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
      batchUpdates(() => {
        setSelectionParam(query.toString());
        setMergeFromState(mergeFrom.toString());
        setMergeToState(mergeTo.toString());
      });
    },
    [batchUpdates, setSelectionParam, setMergeFromState, setMergeToState]
  );

  // Apply view defaults to URL params (only if URL params are empty)
  const applyViewDefaults = useCallback(() => {
    batchUpdates(() => {
      const defaults = suffix === '' || suffix === '_a' ? viewDefaults : sharedDefaults;
      if (defaults === undefined) return;

      // Apply expression default using preserve-existing strategy
      if (defaults.expression !== undefined) {
        setExpressionWithPreserve(defaults.expression);
      }

      // Apply sum_by default using preserve-existing strategy
      if (defaults.sumBy !== undefined) {
        setSumByWithPreserve(sumByToParam(defaults.sumBy));
      }

      // Apply group_by default only for _a hook using preserve-existing strategy
      if (isGroupByEnabled && defaults.groupBy !== undefined) {
        setGroupByWithPreserve(defaults.groupBy.join(','));
      }
    });
  }, [
    batchUpdates,
    suffix,
    viewDefaults,
    sharedDefaults,
    setExpressionWithPreserve,
    setSumByWithPreserve,
    isGroupByEnabled,
    setGroupByWithPreserve,
  ]);

  // Reset query to default state
  const resetQuery = useCallback(() => {
    batchUpdates(() => {
      setExpressionState(defaultExpression);
      setSumByParam(undefined);
      if (isGroupByEnabled) {
        setGroupByParam(undefined);
      }
    });
  }, [
    batchUpdates,
    setExpressionState,
    defaultExpression,
    setSumByParam,
    isGroupByEnabled,
    setGroupByParam,
  ]);

  const draftParsedQuery = useMemo(() => {
    try {
      return Query.parse(draftSelection.expression ?? '');
    } catch {
      return Query.parse('');
    }
  }, [draftSelection.expression]);

  const parsedQuery = useMemo(() => {
    try {
      return Query.parse(querySelection.expression ?? '');
    } catch {
      return Query.parse('');
    }
  }, [querySelection.expression]);

  // Parse expression components using existing Query parser
  const {hasProfileType, profileTypeString, matchersOnly, fullExpression} = useMemo(() => {
    const expr = expression ?? defaultExpression;
    const parsed = Query.parse(expr);

    const profileType = parsed.profileType();
    const profileTypeStr = profileType.toString();
    const hasProfile = profileTypeStr !== '';
    const matchers = `{${parsed.matchersString()}}`;

    return {
      hasProfileType: hasProfile,
      profileTypeString: profileTypeStr,
      matchersOnly: matchers,
      fullExpression: parsed.toString(),
    };
  }, [expression, defaultExpression]);

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

    // Parsed expression components
    hasProfileType,
    profileTypeString,
    matchersOnly,
    fullExpression,

    // Group-by state (only for _a hook)
    ...(isGroupByEnabled
      ? {
          groupBy: groupByParam?.split(',').filter(Boolean),
          setGroupBy: (groupBy: string[] | undefined) => {
            setGroupByParam(groupBy?.join(','));
          },
          isGroupByLoading: false,
        }
      : {}),

    // Methods
    applyViewDefaults,
    resetQuery,
  };
};
