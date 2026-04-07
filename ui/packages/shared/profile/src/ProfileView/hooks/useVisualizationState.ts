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

import {useQueryState} from 'nuqs';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../ProfileFlameGraph/FlameGraphArrow';
import {CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';
import {
  colorByParser,
  flamechartDimensionParser,
  groupByParser,
  jsonParser,
  stringParam,
} from '../../hooks/urlParsers';
import {useResetFlameGraphState} from './useResetFlameGraphState';

export const useVisualizationState = (): {
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  colorStackLegend: string | null;
  colorBy: string;
  setColorBy: (colorBy: string) => void;
  groupBy: string[];
  setGroupBy: (keys: string[]) => void;
  toggleGroupBy: (key: string) => void;
  setGroupByLabels: (labels: string[]) => void;
  flamechartDimension: string[];
  setFlamechartDimension: (labels: string[]) => void;
  sandwichFunctionName: string | null;
  setSandwichFunctionName: (sandwichFunctionName: string | null) => void;
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

  const [curPathArrow, setRawCurPathArrow] = useQueryState(
    'cur_path',
    jsonParser<CurrentPathFrame[]>().withDefault([])
  );
  const setCurPathArrow = useCallback(
    (path: CurrentPathFrame[]) => {
      void setRawCurPathArrow(path);
    },
    [setRawCurPathArrow]
  );
  const [colorStackLegend] = useQueryState('color_stack_legend', stringParam);
  const [colorBy, setStoreColorBy] = useQueryState('color_by', colorByParser);
  const [alignFunctionNameRaw, setStoreAlignFunctionName] = useQueryState(
    'align_function_name',
    stringParam
  );
  const alignFunctionName = alignFunctionNameRaw ?? alignFunctionNamePreference ?? 'left';
  const [groupBy, setStoreGroupBy] = useQueryState(
    'group_by',
    groupByParser.withDefault([FIELD_FUNCTION_NAME])
  );
  const [sandwichFunctionName, setRawSandwichFunctionName] = useQueryState(
    'sandwich_function_name',
    stringParam
  );
  const setSandwichFunctionName = useCallback(
    (name: string | null) => {
      void setRawSandwichFunctionName(name);
    },
    [setRawSandwichFunctionName]
  );
  const [flamechartDimension, setStoreFlamechartDimension] = useQueryState(
    'flamechart_dimension',
    flamechartDimensionParser.withDefault([])
  );
  const resetFlameGraphState = useResetFlameGraphState();

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
      void setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const toggleGroupBy = useCallback(
    (key: string): void => {
      if (groupBy.includes(key)) {
        setGroupBy(groupBy.filter(v => v !== key));
      } else {
        const filteredGroupBy = groupBy.filter(item => !levelsOfProfiling.includes(item));
        setGroupBy([...filteredGroupBy, key]);
      }
      resetFlameGraphState();
    },
    [groupBy, setGroupBy, levelsOfProfiling, resetFlameGraphState]
  );

  const setGroupByLabels = useCallback(
    (labels: string[]): void => {
      setGroupBy(groupBy.filter(l => !l.startsWith(`${FIELD_LABELS}.`)).concat(labels));
      resetFlameGraphState();
    },
    [groupBy, setGroupBy, resetFlameGraphState]
  );

  const setFlamechartDimension = useCallback(
    (labels: string[]): void => {
      void setStoreFlamechartDimension(labels.filter(l => l.startsWith(`${FIELD_LABELS}.`)));
    },
    [setStoreFlamechartDimension]
  );

  const resetSandwichFunctionName = useCallback((): void => {
    setSandwichFunctionName(null);
  }, [setSandwichFunctionName]);

  const setColorBy = useCallback(
    (value: string): void => {
      void setStoreColorBy(value);
      setColorByPreference(value);
    },
    [setStoreColorBy, setColorByPreference]
  );

  const setAlignFunctionName = useCallback(
    (value: string): void => {
      void setStoreAlignFunctionName(value);
      setAlignFunctionNamePreference(value);
    },
    [setStoreAlignFunctionName, setAlignFunctionNamePreference]
  );

  return {
    curPathArrow,
    setCurPathArrow,
    colorStackLegend,
    colorBy,
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
    alignFunctionName,
    setAlignFunctionName,
  };
};
