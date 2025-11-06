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

import {useCallback} from 'react';

import {useURLState, useURLStateBatch} from '@parca/components';

/**
 * Hook to manage compare mode state and operations
 * Returns compare mode flags and a function to close compare mode
 */
export const useCompareModeMeta = (): {
  isCompareMode: boolean;
  isCompareAbsolute: boolean;
  closeCompareMode: (card: 'A' | 'B') => void;
} => {
  const batchUpdates = useURLStateBatch();

  // Side A URL state (only setters needed)
  const [, setExpressionA] = useURLState<string>('expression_a');
  const [, setFromA] = useURLState<string>('from_a');
  const [, setToA] = useURLState<string>('to_a');
  const [, setTimeSelectionA] = useURLState<string>('time_selection_a');
  const [, setSumByA] = useURLState<string>('sum_by_a');
  const [, setMergeFromA] = useURLState<string>('merge_from_a');
  const [, setMergeToA] = useURLState<string>('merge_to_a');
  const [, setSelectionA] = useURLState<string>('selection_a');

  // Side B URL state
  const [expressionB, setExpressionB] = useURLState<string>('expression_b');
  const [fromB, setFromB] = useURLState<string>('from_b');
  const [toB, setToB] = useURLState<string>('to_b');
  const [timeSelectionB, setTimeSelectionB] = useURLState<string>('time_selection_b');
  const [sumByB, setSumByB] = useURLState<string>('sum_by_b');
  const [mergeFromB, setMergeFromB] = useURLState<string>('merge_from_b');
  const [mergeToB, setMergeToB] = useURLState<string>('merge_to_b');
  const [selectionB, setSelectionB] = useURLState<string>('selection_b');

  // Compare mode flags (expose values for routing decisions)
  const [compareA, setCompareA] = useURLState<string>('compare_a');
  const [compareB, setCompareB] = useURLState<string>('compare_b');
  const [compareAbsolute, setCompareAbsolute] = useURLState<string>('compare_absolute');

  const closeCompareMode = useCallback(
    (side: 'A' | 'B') => {
      batchUpdates(() => {
        // If closing side A, swap A and B params first (keep B's data as the single view)
        if (side === 'A') {
          // Copy B to A
          setExpressionA(expressionB);
          setFromA(fromB);
          setToA(toB);
          setTimeSelectionA(timeSelectionB);
          setSumByA(sumByB);
          setMergeFromA(mergeFromB);
          setMergeToA(mergeToB);
          setSelectionA(selectionB);
        }

        // Clear all B params
        setExpressionB(undefined);
        setFromB(undefined);
        setToB(undefined);
        setTimeSelectionB(undefined);
        setSumByB(undefined);
        setMergeFromB(undefined);
        setMergeToB(undefined);
        setSelectionB(undefined);

        // Clear compare mode flags
        setCompareA(undefined);
        setCompareB(undefined);
        setCompareAbsolute(undefined);
      });
    },
    [
      batchUpdates,
      // Side A setters
      setExpressionA,
      setFromA,
      setToA,
      setTimeSelectionA,
      setSumByA,
      setMergeFromA,
      setMergeToA,
      setSelectionA,
      // Side B values (for swapping)
      expressionB,
      fromB,
      toB,
      timeSelectionB,
      sumByB,
      mergeFromB,
      mergeToB,
      selectionB,
      // Side B setters
      setExpressionB,
      setFromB,
      setToB,
      setTimeSelectionB,
      setSumByB,
      setMergeFromB,
      setMergeToB,
      setSelectionB,
      // Compare flags
      setCompareA,
      setCompareB,
      setCompareAbsolute,
    ]
  );

  // Derive isCompareMode from flags
  const isCompareMode = compareA === 'true' || compareB === 'true';
  const isCompareAbsolute = compareAbsolute === 'true';

  return {
    isCompareMode,
    isCompareAbsolute,
    closeCompareMode,
  };
};
