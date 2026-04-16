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

import {useQueryStates} from 'nuqs';

import {boolParam, stringParam} from './urlParsers';

/**
 * Hook to manage compare mode state and operations
 * Returns compare mode flags and a function to close compare mode
 */
export const useCompareModeMeta = (): {
  isCompareMode: boolean;
  isCompareAbsolute: boolean;
  closeCompareMode: (card: 'A' | 'B') => void;
} => {
  const [state, setState] = useQueryStates(
    {
      // Side A
      expression_a: stringParam,
      from_a: stringParam,
      to_a: stringParam,
      time_selection_a: stringParam,
      sum_by_a: stringParam,
      merge_from_a: stringParam,
      merge_to_a: stringParam,
      selection_a: stringParam,
      // Side B
      expression_b: stringParam,
      from_b: stringParam,
      to_b: stringParam,
      time_selection_b: stringParam,
      sum_by_b: stringParam,
      merge_from_b: stringParam,
      merge_to_b: stringParam,
      selection_b: stringParam,
      // Compare flags
      compare_a: boolParam,
      compare_b: boolParam,
      compare_absolute: boolParam,
    },
    {history: 'replace'}
  );

  const closeCompareMode = useCallback(
    (side: 'A' | 'B') => {
      // If closing side A, swap B → A first (keep B's data as the single view)
      const swapAFromB =
        side === 'A'
          ? {
              expression_a: state.expression_b,
              from_a: state.from_b,
              to_a: state.to_b,
              time_selection_a: state.time_selection_b,
              sum_by_a: state.sum_by_b,
              merge_from_a: state.merge_from_b,
              merge_to_a: state.merge_to_b,
              selection_a: state.selection_b,
            }
          : {};

      // Atomic update: swap A (if needed), clear all B params and compare flags
      void setState({
        ...swapAFromB,
        expression_b: null,
        from_b: null,
        to_b: null,
        time_selection_b: null,
        sum_by_b: null,
        merge_from_b: null,
        merge_to_b: null,
        selection_b: null,
        compare_a: null,
        compare_b: null,
        compare_absolute: null,
      });
    },
    [state, setState]
  );

  const isCompareMode = state.compare_a === true || state.compare_b === true;
  const isCompareAbsolute = state.compare_absolute === true;

  return {
    isCompareMode,
    isCompareAbsolute,
    closeCompareMode,
  };
};
