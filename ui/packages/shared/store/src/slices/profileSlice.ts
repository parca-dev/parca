import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import type {RootState} from '../index';

// Define a type for the slice state
interface ProfileState {
  compare: boolean;
}

// Define the initial state using that type
const initialState: ProfileState = {
  compare: false,
};

export const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    setCompare: (state, action: PayloadAction<boolean>) => {
      state.compare = action.payload;
    },
  },
});

export const {setCompare} = profileSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectCompareMode = (state: RootState) => state.profile.compare;

export default profileSlice.reducer;
