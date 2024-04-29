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

import {EVERYTHING_ELSE} from '@parca/store';
import {diffColor, diffColorPerSecond, getLastItem} from '@parca/utilities';

interface mappingColors {
  [key: string]: string;
}

interface Props {
  isDarkMode: boolean;
  compareMode: boolean;
  cumulative: bigint;
  cumulativePerSecond: number | null;
  diff: bigint | null;
  diffPerSecond: number | null;
  mappingColors: mappingColors;
  functionName: string | null;
  mappingFile: string | null;
}

const useNodeColor = ({
  isDarkMode,
  compareMode,
  cumulative,
  cumulativePerSecond,
  diff,
  diffPerSecond,
  mappingColors,
  functionName,
  mappingFile,
}: Props): string => {
  if (compareMode) {
    if (cumulativePerSecond !== null && diffPerSecond !== null) {
      return diffColorPerSecond(diffPerSecond, cumulativePerSecond, isDarkMode);
    }

    return diffColor(diff ?? 0n, cumulative, isDarkMode);
  }

  // To get the color we first check if the function name starts with 'runtime'.
  // If it does, we color it as runtime. Otherwise, we check the mapping file.
  // If there is no mapping file, we color it as 'everything else'.
  return functionName?.startsWith('runtime') === true
    ? mappingColors.runtime
    : mappingColors[getLastItem(mappingFile ?? '') ?? EVERYTHING_ELSE];
};

export default useNodeColor;
