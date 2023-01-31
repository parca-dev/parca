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

import {diffColor} from '@parca/functions';
import {EVERYTHING_ELSE, selectDarkMode, selectStackColors, useAppSelector} from '@parca/store';
import {useMemo} from 'react';
import type {ColoredFlamegraphNode} from './useColoredGraph';

interface Props {
  data: ColoredFlamegraphNode;
}

const useNodeColor = ({data}: Props): string => {
  const colors = useAppSelector(selectStackColors);
  const isDarkMode = useAppSelector(selectDarkMode);

  const color: string = useMemo(() => {
    const diff = parseFloat(data.diff);
    // eslint-disable-next-line no-constant-condition
    if (Math.abs(diff) > 0) {
      const cumulative = parseFloat(data.cumulative);
      return diffColor(diff, cumulative, isDarkMode);
    }

    const color = colors[data.feature ?? EVERYTHING_ELSE];
    return color;
  }, [data, colors, isDarkMode]);

  return color;
};

export default useNodeColor;
