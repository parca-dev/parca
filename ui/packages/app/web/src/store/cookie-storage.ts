import Cookies from 'universal-cookie'
import { StateStorage } from 'zustand/middleware'

const cookie = new Cookies()

const CookieStorage: StateStorage = {
  setItem: (key, value) => {
    cookie.set(key, value)
  },
  getItem: key => {
    return cookie.get(key)
  }
}

export default CookieStorage
