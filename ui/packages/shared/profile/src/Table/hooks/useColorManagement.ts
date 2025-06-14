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

import {type ColorConfig} from '@parca/utilities';

import {getFilenameColors, getMappingColors} from '../../ProfileIcicleGraph/IcicleGraphArrow';

interface UseColorManagementProps {
  isDarkMode: boolean;
  currentColorProfile: ColorConfig;
  mappingsList: string[];
  filenamesList: string[];
  colorBy: string;
}

export function useColorManagement({
  isDarkMode,
  currentColorProfile,
  mappingsList,
  filenamesList,
  colorBy,
}: UseColorManagementProps): {
  colorByColors: Record<string, string>;
  colorByValue: string;
} {
  const filenameColors = useMemo(() => {
    return getFilenameColors(filenamesList, isDarkMode, currentColorProfile);
  }, [isDarkMode, filenamesList, currentColorProfile]);

  const mappingColors = useMemo(() => {
    return getMappingColors(mappingsList, isDarkMode, currentColorProfile);
  }, [isDarkMode, mappingsList, currentColorProfile]);

  const colorByList = {
    filename: filenameColors,
    binary: mappingColors,
  };

  const colorByValue = colorBy === undefined || colorBy === '' ? 'binary' : colorBy;
  const colorByColors = colorByList[colorByValue as keyof typeof colorByList];

  return {
    colorByColors,
    colorByValue,
  };
}
