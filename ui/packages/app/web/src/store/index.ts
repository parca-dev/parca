import { useLayoutEffect } from 'react'
import create from 'zustand'
import createContext from 'zustand/context'
import { persist } from 'zustand/middleware'
import CookieStorage from './cookie-storage'
import NoopStorage from './noop-storage'
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

export const initializeStore = (initialState = {}) => {
  return create(
    persist(
      (set, get) => ({
        ...stateSlices(set, get),
        ...initialState
      }),
      {
        name: 'parca',
        whitelist: whitelistPersist,
        getStorage: () => (typeof document !== 'undefined' ? CookieStorage : NoopStorage)
      }
    )
  )
}

export function useCreateStore(initialState) {
  // For SSR & SSG, always use a new store.
  if (typeof window === 'undefined') {
    return () => initializeStore(initialState)
  }

  // For CSR, always re-use same store.
  store = store ?? initializeStore(initialState)
  // And if initialState changes, then merge states in the next render cycle.
  // @todo does initialState ever change?
  useLayoutEffect(() => {
    if (initialState && store) {
      store.setState({
        ...store.getState(),
        ...initialState
      })
    }
  }, [initialState])

  return () => store
}
