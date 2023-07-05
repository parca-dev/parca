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

import type {RootState} from '../store';

// Define a type for the slice state
export interface UiState {
  darkMode: boolean;
  parcaThemeSystemSettings: boolean;
}

// Define the initial state using that type
const initialState: UiState = {
  darkMode: false,
  parcaThemeSystemSettings: true,
};

export const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    setDarkMode: (state, action: PayloadAction<boolean>) => {
      state.darkMode = action.payload;
    },
    setParcaThemeSystemSettings: (state, action: PayloadAction<boolean>) => {
      state.parcaThemeSystemSettings = action.payload;
    },
  },
});

export const {setDarkMode, setParcaThemeSystemSettings} = uiSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectDarkMode = (state: RootState): boolean => state.ui.darkMode;

export const selectParcaThemeSystemSettings = (state: RootState): boolean =>
  state.ui.parcaThemeSystemSettings;

export default uiSlice.reducer;
