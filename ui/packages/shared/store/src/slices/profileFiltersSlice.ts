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

export interface ProfileFilter {
  id: string;
  type?: 'stack' | 'frame' | string; // string allows preset keys
  field?: 'function_name' | 'binary' | 'system_name' | 'filename' | 'address' | 'line_number';
  matchType?: 'equal' | 'not_equal' | 'contains' | 'not_contains';
  value: string;
}

// Define a type for the slice state
export interface ProfileFiltersState {
  localFilters: ProfileFilter[];
}

// Define the initial state using that type
const initialState: ProfileFiltersState = {
  localFilters: [],
};

export const profileFiltersSlice = createSlice({
  name: 'profileFilters',
  initialState,
  reducers: {
    setLocalFilters: (state, action: PayloadAction<ProfileFilter[]>) => {
      state.localFilters = action.payload;
    },
  },
});

export const {setLocalFilters} = profileFiltersSlice.actions;

export const selectLocalFilters = (state: RootState): ProfileFilter[] =>
  state.profileFilters.localFilters;

export default profileFiltersSlice.reducer;
