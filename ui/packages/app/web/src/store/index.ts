import create from 'zustand'
import createContext from 'zustand/context'
import { persist } from 'zustand/middleware'
import createUiState from './ui.state'

let store

const whitelistPersist = ['ui']

const stateSlices = (set, get) => ({
  ...createUiState(set, get)
  // add more slices
})

type AppState = ReturnType<typeof stateSlices>

const zustandContext = createContext<AppState>()

export const StoreProvider = zustandContext.Provider

export const useStore = zustandContext.useStore

export const initializeStore = () => {
  return create(
    persist((set, get) => stateSlices(set, get), {
      name: 'parca',
      whitelist: whitelistPersist
    })
  )
}

export function useCreateStore() {
  // For SSR & SSG, always use a new store.
  if (typeof window === 'undefined') {
    return () => initializeStore()
  }

  // For CSR, always re-use same store.
  store = store ?? initializeStore()

  return () => store
}
