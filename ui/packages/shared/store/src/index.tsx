import store from './store';
import {useAppDispatch, useAppSelector} from './hooks';
import {selectDarkMode, setDarkMode} from './slices/uiSlice';
import {
  selectCompareMode,
  setCompare,
  selectSearchNodeString,
  setSearchNodeString,
} from './slices/profileSlice';

export {
  store,
  useAppDispatch,
  useAppSelector,
  selectCompareMode,
  setCompare,
  selectDarkMode,
  setDarkMode,
  selectSearchNodeString,
  setSearchNodeString,
};
