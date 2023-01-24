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

import {COLOR_PROFILES} from '@parca/functions';
import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import type {RootState} from '../store';

// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
export type StackColorMap = {[key: string]: string};

// Define a type for the slice state
export interface ColorsState {
  colors: StackColorMap;
}

// Define the initial state using that type
const initialState: ColorsState = {
  colors: {},
};

export interface StackColor {
  color: string;
  name: string;
}

const findAColor = (colorIndex: number, colors: string[][]): string[] => {
  return colors[colorIndex];
  // TODO: add some logic to find unallocated colors if this index is already allocated to another feature for better color distribution.
};

const getColorForFeature = (feature: string, isDarkMode: boolean, colorProfileName): string => {
  const colors = COLOR_PROFILES[colorProfileName].colors;

  // Add charaters in the feature name to the color map
  const colorIndex =
    feature === 'Everything else'
      ? colors.length - 1
      : feature
          .toLowerCase()
          .split('')
          .reduce((acc, char) => {
            acc += char.charCodeAt(0);
            return acc;
          }, 0) % (colors.length > 1 ? colors.length - 1 : 1);

  const color = findAColor(colorIndex, colors);
  return !isDarkMode ? color[0] : color[1];
};

export interface GenerateColorRequest {
  feature: string;
  colorProfileName: string;
}

export const colorsSlice = createSlice({
  name: 'colors',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    addColor: (state, action: PayloadAction<StackColor>) => {
      state.colors = {
        ...state.colors,
        [action.payload.name]: action.payload.color,
      };
    },
    generateColorForFeature: function generateColorForFeatureFunc(
      state,
      action: PayloadAction<GenerateColorRequest>
    ) {
      if (state.colors[action.payload.feature] != null) {
        return; // Don't generate a color if it already exists (e.g. when switching color profiles
      }
      console.log('Generating color for feature', action.payload.feature);
      state.colors = {
        ...state.colors,
        [action.payload.feature]: getColorForFeature(
          action.payload.feature,
          false,
          action.payload.colorProfileName
        ),
      };
    },
    resetColors: state => {
      state.colors = {};
    },
  },
});

export const {addColor, resetColors, generateColorForFeature} = colorsSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectStackColors = (state: RootState): StackColorMap => state.colors.colors;

export default colorsSlice.reducer;
