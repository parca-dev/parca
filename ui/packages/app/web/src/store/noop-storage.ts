import { StateStorage } from 'zustand/middleware'

const NoopStorage: StateStorage = {
  setItem: (_key, _value) => {},
  getItem: key => key
}

export default NoopStorage
