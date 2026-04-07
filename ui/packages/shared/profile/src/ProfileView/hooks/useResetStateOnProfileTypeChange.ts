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

import {useQueryStates} from 'nuqs';

import {stringParam} from '../../hooks/urlParsers';
import {useProfileFilters} from '../components/ProfileFilters/useProfileFilters';

export const useResetStateOnProfileTypeChange = (): (() => void) => {
  const [state, setState] = useQueryStates(
    {
      group_by: stringParam,
      cur_path: stringParam,
      sum_by_a: stringParam,
      sum_by_b: stringParam,
      sandwich_function_name: stringParam,
    },
    {history: 'replace'}
  );
  const {resetFilters} = useProfileFilters();

  return () => {
    // Atomic reset: clear all params in single URL update
    const updates: Record<string, null> = {};
    if (state.group_by !== null) updates.group_by = null;
    if (state.cur_path !== null) updates.cur_path = null;
    if (state.sandwich_function_name !== null) updates.sandwich_function_name = null;
    if (state.sum_by_a !== null) updates.sum_by_a = null;
    if (state.sum_by_b !== null) updates.sum_by_b = null;

    if (Object.keys(updates).length > 0) {
      void setState(updates);
    }

    resetFilters();
  };
};
