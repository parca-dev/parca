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

const PREFERRED_DIMENSIONS = ['cpu', 'cpuid', 'thread', 'thread_id'];

/**
 * Auto-selects the best flamechart "group by" dimension on first load.
 * Priority: cpu > cpuid > thread > thread_id
 */
export const useAutoSelectDimension = (
  metadataLabels: string[] | undefined,
  flamechartDimension: string[] | undefined,
  setFlamechartDimension: (v: string[]) => void
): void => {
  const hasAutoSelected = useRef(false);

  useEffect(() => {
    if (hasAutoSelected.current) return;
    if (metadataLabels == null || metadataLabels.length === 0) return;
    if ((flamechartDimension ?? []).length > 0) {
      hasAutoSelected.current = true;
      return;
    }

    for (const name of PREFERRED_DIMENSIONS) {
      if (metadataLabels.includes(name)) {
        setFlamechartDimension([`labels.${name}`]);
        hasAutoSelected.current = true;
        return;
      }
    }
  }, [metadataLabels, flamechartDimension, setFlamechartDimension]);
};
