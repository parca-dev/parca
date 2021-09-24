import produce from 'immer'

export type UiState = ReturnType<typeof createSlice>

export default function createSlice (set, _get): {
  ui: {
    darkMode: boolean
  }
  setDarkMode: (mode: boolean) => void
} {
  return {
    // state
    ui: {
      darkMode: false
    },
    // actions
    setDarkMode: (mode: boolean) => {
      set(
        produce<UiState>(state => {
          state.ui.darkMode = mode
        })
      )
    }
  }
}

export const selectUi = (state: UiState) => state.ui
