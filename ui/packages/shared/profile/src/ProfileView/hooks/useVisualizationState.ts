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

import {useCallback, useState} from 'react';

import {useURLState} from '@parca/components';

import {FIELD_FUNCTION_NAME, FIELD_LABELS} from '../../ProfileIcicleGraph/IcicleGraphArrow';

export const useVisualizationState = (): {
  curPath: string[];
  setCurPath: (path: string[]) => void;
  currentSearchString: string | undefined;
  setSearchString: (searchString: string | undefined) => void;
  colorStackLegend: string | undefined;
  colorBy: string;
  groupBy: string[];
  setGroupBy: (keys: string[]) => void;
  toggleGroupBy: (key: string) => void;
  clearSelection: () => void;
  setGroupByLabels: (labels: string[]) => void;
} => {
  const [curPath, setCurPath] = useState<string[]>([]);
  const [currentSearchString, setSearchString] = useURLState<string | undefined>('search_string');
  const [colorStackLegend] = useURLState<string | undefined>('color_stack_legend');
  const [colorBy] = useURLState('color_by');
  const [groupBy, setStoreGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const toggleGroupBy = useCallback(
    (key: string): void => {
      groupBy.includes(key)
        ? setGroupBy(groupBy.filter(v => v !== key)) // remove
        : setGroupBy([...groupBy, key]); // add
    },
    [groupBy, setGroupBy]
  );

  const setGroupByLabels = useCallback(
    (labels: string[]): void => {
      setGroupBy(groupBy.filter(l => !l.startsWith(`${FIELD_LABELS}.`)).concat(labels));
    },
    [groupBy, setGroupBy]
  );

  const clearSelection = useCallback((): void => {
    setSearchString?.('');
  }, [setSearchString]);

  return {
    curPath,
    setCurPath,
    currentSearchString,
    setSearchString,
    colorStackLegend,
    colorBy: (colorBy as string) ?? '',
    groupBy,
    setGroupBy,
    toggleGroupBy,
    setGroupByLabels,
    clearSelection,
  };
};
