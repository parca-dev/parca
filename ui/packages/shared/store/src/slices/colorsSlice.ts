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

// This is needed to fix a type error referencing immer. See https://github.com/reduxjs/redux-toolkit/issues/1806#issuecomment-1709901225
import 'immer';

import {createSlice, type PayloadAction} from '@reduxjs/toolkit';

import {CallgraphNode, CallgraphNodeMeta, FlamegraphNode, FlamegraphNodeMeta} from '@parca/client';
import {
  COLOR_PROFILES,
  type ColorConfig,
  type ColorProfileName,
  type ColorsDuo,
} from '@parca/utilities';

import type {RootState} from '../store';

// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
export type StackColorMap = {[key: string]: string};

interface ExtendedCallgraphNodeMeta extends CallgraphNodeMeta {
  lineIndex: number;
  locationIndex: number;
}

export interface HoveringNode extends FlamegraphNode, CallgraphNode {
  diff: bigint;
  meta?: FlamegraphNodeMeta | ExtendedCallgraphNodeMeta;
  cumulative: bigint;
}

export interface HoveringRow {
  row: number;
}

// Define a type for the slice state
export interface ColorsState {
  colors: StackColorMap;
  binaries: string[];
  hoveringRow: HoveringRow | undefined;
  colorProfiles: Record<string, ColorConfig>;
}

// Define the initial state using that type
export const initialColorState: ColorsState = {
  colors: {},
  binaries: [],
  hoveringRow: undefined,
  colorProfiles: COLOR_PROFILES,
};

export interface StackColor {
  color: string;
  name: string;
}

export const FILENAMES_FEATURE_TYPES = {
  Filename: 'Filename',
  Misc: 'Misc',
} as const;

export const BINARY_FEATURE_TYPES = {
  Binary: 'Binary',
  Misc: 'Misc',
} as const;

export type BinaryFeatureType = (typeof BINARY_FEATURE_TYPES)[keyof typeof BINARY_FEATURE_TYPES];

export type FilenameFeatureType =
  (typeof FILENAMES_FEATURE_TYPES)[keyof typeof FILENAMES_FEATURE_TYPES];

export interface FilenameFeature {
  name: string;
  type: FilenameFeatureType;
}

export interface BinaryFeature {
  name: string;
  type: BinaryFeatureType;
}

export interface FilenameFeaturesMap {
  [key: string]: FilenameFeatureType;
}

export interface BinaryFeaturesMap {
  [key: string]: BinaryFeatureType;
}

export const EVERYTHING_ELSE = 'Everything else';

const findAColor = (colorIndex: number, colors: ColorsDuo[]): ColorsDuo => {
  return colors[colorIndex];
  // TODO: add some logic to find unallocated colors if this index is already allocated to another feature for better color distribution.
};

export const getColorForFeature = (
  feature: string,
  isDarkMode: boolean,
  colors: ColorsDuo[]
): string => {
  // Add characters in the feature name to the color map
  const colorIndex =
    feature === EVERYTHING_ELSE
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

export interface SetFeaturesRequest {
  features: BinaryFeaturesMap | FilenameFeaturesMap;
  colorProfileName: ColorProfileName;
  isDarkMode: boolean;
}

export const colorsSlice = createSlice({
  name: 'colors',
  initialState: initialColorState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    addColor: (state, action: PayloadAction<StackColor>) => {
      state.colors = {
        ...state.colors,
        [action.payload.name]: action.payload.color,
      };
    },
    setFeatures: (state, action: PayloadAction<SetFeaturesRequest>) => {
      const currentProfileColors = state.colorProfiles[action.payload.colorProfileName].colors;
      state.binaries = Object.keys(action.payload.features).filter(name => {
        return action.payload.features[name] === BINARY_FEATURE_TYPES.Binary;
      });
      state.colors = Object.keys(action.payload.features)
        .map(feature => {
          return [
            feature,
            getColorForFeature(feature, action.payload.isDarkMode, currentProfileColors),
          ];
        })
        .reduce(
          (acc: {[key: string]: string}, [feature, color]) => {
            acc[feature] = color;
            return acc;
          },
          {
            [EVERYTHING_ELSE]: getColorForFeature(
              EVERYTHING_ELSE,
              action.payload.isDarkMode,
              currentProfileColors
            ),
          }
        );
    },
    setHoveringRow: (state, action: PayloadAction<HoveringRow | undefined>) => {
      state.hoveringRow = action.payload;
    },
    resetColors: state => {
      state.colors = {};
    },
  },
});

export const {addColor, resetColors, setFeatures, setHoveringRow} = colorsSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectStackColors = (state: RootState): StackColorMap => state.colors.colors;

export const selectBinaries = (state: RootState): string[] => state.colors.binaries;

export const selectHoveringRow = (state: RootState): HoveringRow | undefined =>
  state.colors.hoveringRow;

export const selectColorProfiles = (state: RootState): Record<string, ColorConfig> =>
  state.colors.colorProfiles;

export default colorsSlice.reducer;
