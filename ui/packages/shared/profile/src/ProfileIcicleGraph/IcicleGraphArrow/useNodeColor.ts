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

import {useMemo} from 'react';

import {Table} from 'apache-arrow';

import {EVERYTHING_ELSE, selectDarkMode, selectStackColors, useAppSelector} from '@parca/store';
import {diffColor} from '@parca/utilities';

import {FIELD_CUMULATIVE, FIELD_DIFF} from './index';

interface Props {
  table: Table<any>;
  row: number;
  compareMode: boolean;
}

const useNodeColor = ({table, row, compareMode}: Props): string => {
  const colors = useAppSelector(selectStackColors);
  const isDarkMode = useAppSelector(selectDarkMode);

  const cumulative = table.getChild(FIELD_CUMULATIVE)?.get(row);
  const diff = table.getChild(FIELD_DIFF)?.get(row);

  const color: string = useMemo(() => {
    if (compareMode) {
      return diffColor(diff, cumulative, isDarkMode);
    }

    console.log(colors);

    const color = colors[EVERYTHING_ELSE];
    // const color =
    //   colors[data.feature ?? EVERYTHING_ELSE] ??
    //   (!isDarkMode ? COLOR_PROFILES.default.colors[0][0] : COLOR_PROFILES.default.colors[0][1]);

    return color;
  }, [compareMode, colors, diff, cumulative, isDarkMode]);

  return color;
};

export default useNodeColor;
