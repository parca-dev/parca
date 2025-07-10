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

import {useCallback, useMemo, useState} from 'react';

import {JSONParser, JSONSerializer, useURLState, useURLStateCustom} from '@parca/components';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../ProfileFlameGraph/FlameGraphArrow';
import {CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';

export const useVisualizationState = (): {
  curPath: string[];
  setCurPath: (path: string[]) => void;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  currentSearchString: string | undefined;
  setSearchString: (searchString: string | undefined) => void;
  colorStackLegend: string | undefined;
  colorBy: string;
  setColorBy: (colorBy: string) => void;
  groupBy: string[];
  setGroupBy: (keys: string[]) => void;
  toggleGroupBy: (key: string) => void;
  clearSelection: () => void;
  setGroupByLabels: (labels: string[]) => void;
  sandwichFunctionName: string | undefined;
  setSandwichFunctionName: (sandwichFunctionName: string | undefined) => void;
  resetSandwichFunctionName: () => void;
} => {
  const [curPath, setCurPath] = useState<string[]>([]);
  const [curPathArrow, setCurPathArrow] = useURLStateCustom<CurrentPathFrame[]>('cur_path', {
    parse: JSONParser<CurrentPathFrame[]>,
    stringify: JSONSerializer,
    defaultValue: '[]',
  });
  const [currentSearchString, setSearchString] = useURLState<string | undefined>('search_string');
  const [colorStackLegend] = useURLState<string | undefined>('color_stack_legend');
  const [colorBy, setColorBy] = useURLState('color_by');
  const [groupBy, setStoreGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });
  const [sandwichFunctionName, setSandwichFunctionName] = useURLState<string | undefined>(
    'sandwich_function_name'
  );

  const levelsOfProfiling = useMemo(
    () => [
      FIELD_FUNCTION_NAME,
      FIELD_FUNCTION_FILE_NAME,
      FIELD_LOCATION_ADDRESS,
      FIELD_MAPPING_FILE,
    ],
    []
  );

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const toggleGroupBy = useCallback(
    (key: string): void => {
      if (groupBy.includes(key)) {
        setGroupBy(groupBy.filter(v => v !== key)); // remove
      } else {
        const filteredGroupBy = groupBy.filter(item => !levelsOfProfiling.includes(item));
        setGroupBy([...filteredGroupBy, key]); // add
      }
    },
    [groupBy, setGroupBy, levelsOfProfiling]
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

  const resetSandwichFunctionName = useCallback((): void => {
    setSandwichFunctionName(undefined);
  }, [setSandwichFunctionName]);

  return {
    curPath,
    setCurPath,
    curPathArrow,
    setCurPathArrow,
    currentSearchString,
    setSearchString,
    colorStackLegend,
    colorBy: (colorBy as string) ?? '',
    setColorBy,
    groupBy,
    setGroupBy,
    toggleGroupBy,
    setGroupByLabels,
    clearSelection,
    sandwichFunctionName,
    setSandwichFunctionName,
    resetSandwichFunctionName,
  };
};
