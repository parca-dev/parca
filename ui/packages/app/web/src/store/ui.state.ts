/* eslint-disable @typescript-eslint/explicit-module-boundary-types */
import produce from 'immer';
import {GetState} from 'zustand';

export type UiState = ReturnType<typeof createSlice>;

export default function createSlice(
  set: any,
  _get: GetState<{[x: string]: {}}>
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
        produce<UiState>(state => {
          state.ui.darkMode = mode;
        })
      );
    },
  };
}

export const selectUi = (state: UiState) => state.ui;
