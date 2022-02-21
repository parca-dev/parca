/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import produce from 'immer';

export type UiState = ReturnType<typeof createSlice>;

export default function createSlice(
  set: any,
  _get: any
): {
  ui: {
    darkMode: boolean;
  };
  setDarkMode: (mode: boolean) => void;
} {
  return {
    // state
    ui: {
      darkMode: false,
    },
    // actions
    setDarkMode: (mode: boolean) => {
      set(
        produce((state: any) => {
          state.ui.darkMode = mode;
        })
      );
    },
  };
}

export const selectUi = (state: UiState) => state.ui;
