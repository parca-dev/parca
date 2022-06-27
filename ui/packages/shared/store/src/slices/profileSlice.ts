import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import type {RootState} from '../store';

// Define a type for the slice state
interface ProfileState {
  compare: boolean;
  searchNodeString: string | undefined;
}

// Define the initial state using that type
const initialState: ProfileState = {
  compare: false,
  searchNodeString: undefined,
};

export const profileSlice = createSlice({
  name: 'profile',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    setCompare: (state, action: PayloadAction<boolean>) => {
      state.compare = action.payload;
    },
    setSearchNodeString: (state, action: PayloadAction<string | undefined>) => {
      state.searchNodeString = action.payload;
    },
  },
});

export const {setCompare, setSearchNodeString} = profileSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectCompareMode = (state: RootState) => state.profile.compare;
export const selectSearchNodeString = (state: RootState) => state.profile.searchNodeString;

export default profileSlice.reducer;
