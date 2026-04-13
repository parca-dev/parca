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

import {useCallback, useMemo} from 'react';

import {
  JSONParser,
  JSONSerializer,
  useURLState,
  useURLStateBatch,
  useURLStateCustom,
} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../ProfileFlameGraph/FlameGraphArrow';
import {CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';
import {useResetFlameGraphState} from './useResetFlameGraphState';

export const useVisualizationState = (): {
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  colorStackLegend: string | undefined;
  colorBy: string;
  setColorBy: (colorBy: string) => void;
  groupBy: string[];
  setGroupBy: (keys: string[]) => void;
  toggleGroupBy: (key: string) => void;
  setGroupByLabels: (labels: string[]) => void;
  flamechartDimension: string[];
  setFlamechartDimension: (labels: string[]) => void;
  sandwichFunctionName: string | undefined;
  setSandwichFunctionName: (sandwichFunctionName: string | undefined) => void;
  resetSandwichFunctionName: () => void;
  alignFunctionName: string;
  setAlignFunctionName: (align: string) => void;
} => {
  const [colorByPreference, setColorByPreference] = useUserPreference<string>(
    USER_PREFERENCES.COLOR_BY.key
  );
  const [alignFunctionNamePreference, setAlignFunctionNamePreference] = useUserPreference<string>(
    USER_PREFERENCES.ALIGN_FUNCTION_NAME.key
  );

  const [curPathArrow, setCurPathArrow] = useURLStateCustom<CurrentPathFrame[]>('cur_path', {
    parse: JSONParser<CurrentPathFrame[]>,
    stringify: JSONSerializer,
    defaultValue: '[]',
  });
  const [colorStackLegend] = useURLState<string | undefined>('color_stack_legend');
  const [colorBy, setStoreColorBy] = useURLState('color_by', {
    defaultValue: colorByPreference,
  });
  const [alignFunctionName, setStoreAlignFunctionName] = useURLState('align_function_name', {
    defaultValue: alignFunctionNamePreference,
  });
  const [groupBy, setStoreGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });
  const [sandwichFunctionName, setSandwichFunctionName] = useURLState<string | undefined>(
    'sandwich_function_name'
  );
  const [flamechartDimension, setStoreFlamechartDimension] = useURLState<string[]>(
    'flamechart_dimension',
    {
      alwaysReturnArray: true,
    }
  );
  const resetFlameGraphState = useResetFlameGraphState();
  const batchUpdates = useURLStateBatch();

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
      // Batch updates to combine setGroupBy + resetFlameGraphState into single URL navigation
      batchUpdates(() => {
        if (groupBy.includes(key)) {
          setGroupBy(groupBy.filter(v => v !== key)); // remove
        } else {
          const filteredGroupBy = groupBy.filter(item => !levelsOfProfiling.includes(item));
          setGroupBy([...filteredGroupBy, key]); // add
        }

        resetFlameGraphState();
      });
    },
    [groupBy, setGroupBy, levelsOfProfiling, resetFlameGraphState, batchUpdates]
  );

  const setGroupByLabels = useCallback(
    (labels: string[]): void => {
      // Batch updates to combine setGroupBy + resetFlameGraphState into single URL navigation
      batchUpdates(() => {
        setGroupBy(groupBy.filter(l => !l.startsWith(`${FIELD_LABELS}.`)).concat(labels));

        resetFlameGraphState();
      });
    },
    [groupBy, setGroupBy, resetFlameGraphState, batchUpdates]
  );

  const setFlamechartDimension = useCallback(
    (labels: string[]): void => {
      setStoreFlamechartDimension(labels.filter(l => l.startsWith(`${FIELD_LABELS}.`)));
    },
    [setStoreFlamechartDimension]
  );

  const resetSandwichFunctionName = useCallback((): void => {
    setSandwichFunctionName(undefined);
  }, [setSandwichFunctionName]);

  const setColorBy = useCallback(
    (value: string): void => {
      setStoreColorBy(value);
      setColorByPreference(value);
    },
    [setStoreColorBy, setColorByPreference]
  );

  const setAlignFunctionName = useCallback(
    (value: string): void => {
      setStoreAlignFunctionName(value);
      setAlignFunctionNamePreference(value);
    },
    [setStoreAlignFunctionName, setAlignFunctionNamePreference]
  );

  return {
    curPathArrow,
    setCurPathArrow,
    colorStackLegend,
    colorBy: (colorBy as string) ?? '',
    setColorBy,
    groupBy,
    setGroupBy,
    toggleGroupBy,
    setGroupByLabels,
    flamechartDimension,
    setFlamechartDimension,
    sandwichFunctionName,
    setSandwichFunctionName,
    resetSandwichFunctionName,
    alignFunctionName: (alignFunctionName as string) ?? 'left',
    setAlignFunctionName,
  };
};
