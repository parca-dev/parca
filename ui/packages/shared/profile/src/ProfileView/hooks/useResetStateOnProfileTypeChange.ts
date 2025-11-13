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

import {useURLState, useURLStateBatch} from '@parca/components';

import {useProfileFilters} from '../components/ProfileFilters/useProfileFilters';

export const useResetStateOnProfileTypeChange = (): (() => void) => {
  const [groupBy, setGroupBy] = useURLState('group_by');
  const [curPath, setCurPath] = useURLState('cur_path');
  const {resetFilters} = useProfileFilters();
  const [sandwichFunctionName, setSandwichFunctionName] = useURLState('sandwich_function_name');
  const batchUpdates = useURLStateBatch();

  return () => {
    // Batch all URL state resets into a single navigation
    batchUpdates(() => {
      if (groupBy !== undefined) {
        setGroupBy(undefined);
      }
      if (curPath !== undefined) {
        setCurPath(undefined);
      }
      if (sandwichFunctionName !== undefined) {
        setSandwichFunctionName(undefined);
      }

      resetFilters();
    });
  };
};
