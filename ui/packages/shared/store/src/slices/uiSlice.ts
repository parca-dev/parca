import {createSlice, PayloadAction} from '@reduxjs/toolkit';
import type {RootState} from '../store';

// Define a type for the slice state
interface UiState {
  darkMode: boolean;
}

// Define the initial state using that type
const initialState: UiState = {
  darkMode: false,
};

export const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    // Use the PayloadAction type to declare the contents of `action.payload`
    setDarkMode: (state, action: PayloadAction<boolean>) => {
      state.darkMode = action.payload;
    },
  },
});

export const {setDarkMode} = uiSlice.actions;

// Other code such as selectors can use the imported `RootState` type
export const selectDarkMode = (state: RootState) => state.ui.darkMode;

export default uiSlice.reducer;
