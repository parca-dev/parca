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
import {diffColor, getLastItem} from '@parca/utilities';

interface mappingColors {
  [key: string]: string;
}

interface Props {
  isDarkMode: boolean;
  compareMode: boolean;
  cumulative: bigint;
  diff: bigint | null;
  mappingColors: mappingColors;
  mappingFile: string | null;
}

const useNodeColor = ({
  isDarkMode,
  compareMode,
  cumulative,
  diff,
  mappingColors,
  mappingFile,
}: Props): string => {
  if (compareMode) {
    return diffColor(diff ?? 0n, cumulative, isDarkMode);
  }

  return mappingColors[getLastItem(mappingFile ?? '') ?? EVERYTHING_ELSE];
};

export default useNodeColor;
