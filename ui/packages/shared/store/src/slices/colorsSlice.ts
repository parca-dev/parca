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

import {createSlice, type PayloadAction} from '@reduxjs/toolkit';

import {
  CallgraphNode,
  CallgraphNodeMeta,
  FlamegraphNode,
  FlamegraphNodeMeta,
  FlamegraphRootNode,
} from '@parca/client';
import {COLOR_PROFILES, type ColorProfileName, type ColorsDuo} from '@parca/utilities';

import type {RootState} from '../store';

// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
export type StackColorMap = {[key: string]: string};

interface ExtendedCallgraphNodeMeta extends CallgraphNodeMeta {
  lineIndex: number;
  locationIndex: number;
}

export interface HoveringNode extends FlamegraphRootNode, FlamegraphNode, CallgraphNode {
  diff: bigint;
  meta?: FlamegraphNodeMeta | ExtendedCallgraphNodeMeta;
  cumulative: bigint;
}

// Define a type for the slice state
export interface ColorsState {
  colors: StackColorMap;
  binaries: string[];
  hoveringNode: HoveringNode | undefined;
}

// Define the initial state using that type
const initialState: ColorsState = {
  colors: {},
  binaries: [],
  hoveringNode: undefined,
};

export interface StackColor {
  color: string;
  name: string;
}

export const FEATURE_TYPES = {
  Runtime: 'Runtime',
  Binary: 'Binary',
  Misc: 'Misc',
} as const;

export type FeatureType = (typeof FEATURE_TYPES)[keyof typeof FEATURE_TYPES];

export interface Feature {
  name: string;
  type: FeatureType;
}

export interface FeaturesMap {
  [key: string]: FeatureType;
}

export const EVERYTHING_ELSE = 'Everything else';

const findAColor = (colorIndex: number, colors: ColorsDuo[]): ColorsDuo => {
  return colors[colorIndex];
  // TODO: add some logic to find unallocated colors if this index is already allocated to another feature for better color distribution.
};

const getColorForFeature = (
  feature: string,
  isDarkMode: boolean,
  colorProfileName: ColorProfileName
): string => {
  const colors: ColorsDuo[] = COLOR_PROFILES[colorProfileName].colors;

  // Add charaters in the feature name to the color map
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
  features: FeaturesMap;
  colorProfileName: ColorProfileName;
  isDarkMode: boolean;
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
    setFeatures: (state, action: PayloadAction<SetFeaturesRequest>) => {
      state.binaries = Object.keys(action.payload.features).filter(name => {
        return action.payload.features[name] === FEATURE_TYPES.Binary;
      });
      state.colors = Object.keys(action.payload.features)
        .map(feature => {
          return [
            feature,
            getColorForFeature(feature, action.payload.isDarkMode, action.payload.colorProfileName),
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
              action.payload.colorProfileName
            ),
          }
        );
    },
    setHoveringNode: (state, action: PayloadAction<HoveringNode | undefined>) => {
      state.hoveringNode = action.payload;
    },
    resetColors: state => {
      state.colors = {};
    },
  },
});

export const {addColor, resetColors, setFeatures, setHoveringNode} = colorsSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectStackColors = (state: RootState): StackColorMap => state.colors.colors;

export const selectBinaries = (state: RootState): string[] => state.colors.binaries;

export const selectHoveringNode = (state: RootState): HoveringNode | undefined =>
  state.colors.hoveringNode;

export default colorsSlice.reducer;
