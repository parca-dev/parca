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

import {scaleLinear} from 'd3';

export const getColorRange = (isDarkMode: boolean): [string, string] => {
  const maxColor = getNewSpanColor(isDarkMode);
  const minColor = scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  return [minColor, maxColor];
};

export const getNewSpanColor = (isDarkMode: boolean): string => {
  return isDarkMode ? '#4F46E5' : '#4338CA'; // Example colors - adjust as needed
};
