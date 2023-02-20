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
export interface ProfileState {
  filterByFunction: string | undefined;
  [key: string]: string | string[] | undefined;
}

// Define the initial state using that type
const initialState: ProfileState = {
  filterByFunction: undefined,
};

export const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    setProfileStateValue: (
      state,
      action: PayloadAction<{key: string; value: string | string[] | undefined}>
    ) => {
      state[action.payload.key] = action.payload.value;
    },
  },
});

export const {setProfileStateValue} = profileSlice.actions;

export const selectFilterByFunction = (state: RootState): string | undefined =>
  state.profile.filterByFunction;
export const selectProfileStateValue = (key: string) => (state: RootState) => {
  return state.profile[key];
};

export default profileSlice.reducer;
