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

import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import type {RootState} from '../store';

// Define a type for the slice state

export type DashboardItem = 'callgraph' | 'icicle' | 'table';
export interface ProfileState {
  compare: boolean;
  dashboardItems: DashboardItem[];
  searchNodeString: string | undefined;
  filterByFunction: string | undefined;
}

// Define the initial state using that type
const initialState: ProfileState = {
  compare: false,
  dashboardItems: ['icicle'],
  searchNodeString: undefined,
  filterByFunction: undefined,
};

export const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    setCompare: (state, action: PayloadAction<boolean>) => {
      state.compare = action.payload;
    },
    setDashboardItems: (state, action: PayloadAction<DashboardItem[] | undefined>) => {
      state.dashboardItems = action.payload;
    },
    setSearchNodeString: (state, action: PayloadAction<string | undefined>) => {
      state.searchNodeString = action.payload;
    },
    setFilterByFunction: (state, action: PayloadAction<string | undefined>) => {
      state.filterByFunction = action.payload;
    },
  },
});

export const {setCompare, setSearchNodeString, setFilterByFunction, setDashboardItems} =
  profileSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectCompareMode = (state: RootState): boolean => state.profile.compare;
export const selectDashboardItems = (state: RootState): DashboardItem[] =>
  state.profile.dashboardItems;
export const selectSearchNodeString = (state: RootState): string | undefined =>
  state.profile.searchNodeString;
export const selectFilterByFunction = (state: RootState): string | undefined =>
  state.profile.filterByFunction;

export default profileSlice.reducer;
