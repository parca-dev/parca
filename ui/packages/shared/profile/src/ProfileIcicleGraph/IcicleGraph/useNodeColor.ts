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

import {EVERYTHING_ELSE, selectDarkMode, selectStackColors, useAppSelector} from '@parca/store';
import {COLOR_PROFILES, diffColor} from '@parca/utilities';

import type {ColoredFlamegraphNode} from './useColoredGraph';

interface Props {
  data: ColoredFlamegraphNode;
  compareMode: boolean;
}

const useNodeColor = ({data, compareMode}: Props): string => {
  const colors = useAppSelector(selectStackColors);
  const isDarkMode = useAppSelector(selectDarkMode);

  const color: string = useMemo(() => {
    if (compareMode) {
      const diff = data.diff;
      const cumulative = data.cumulative;
      return diffColor(diff, cumulative, isDarkMode);
    }

    const color =
      colors[data.feature ?? EVERYTHING_ELSE] ??
      (!isDarkMode ? COLOR_PROFILES.default.colors[0][0] : COLOR_PROFILES.default.colors[0][1]);

    return color;
  }, [data, colors, isDarkMode, compareMode]);

  return color;
};

export default useNodeColor;
