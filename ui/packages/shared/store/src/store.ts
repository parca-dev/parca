// Copyright 2022 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import {combineReducers, configureStore} from '@reduxjs/toolkit';
import type {Store} from 'redux';
import {
  FLUSH,
  PAUSE,
  PERSIST,
  PURGE,
  REGISTER,
  REHYDRATE,
  persistReducer,
  persistStore,
  type Persistor,
} from 'redux-persist';
import storage from 'redux-persist/lib/storage';

import {type ColorConfig} from '@parca/utilities';

import colorsReducer, {initialColorState} from './slices/colorsSlice';
import profileReducer from './slices/profileSlice';
import uiReducer from './slices/uiSlice';

const rootReducer = combineReducers({
  ui: uiReducer,
  profile: profileReducer,
  colors: colorsReducer,
});

const slicesToPersist = ['ui'];

// Infer the `RootState` and `AppDispatch` types from the store itself
export type RootState = ReturnType<typeof rootReducer>;

const persistConfig = {
  key: 'root',
  version: 1,
  storage,
  whitelist: slicesToPersist,
};

const persistedReducer = persistReducer(persistConfig, rootReducer);

export const createStore = (
  additionalColorProfiles: Record<string, ColorConfig> = {}
): {store: Store; persistor: Persistor} => {
  const store = configureStore({
    reducer: persistedReducer,
    devTools: import.meta.env?.DEV ?? false,
    middleware: getDefaultMiddleware =>
      getDefaultMiddleware({
        serializableCheck: {
          ignoredActions: [
            FLUSH,
            REHYDRATE,
            PAUSE,
            PERSIST,
            PURGE,
            REGISTER,
            'colors/setHoveringNode',
          ],
          ignoredPaths: ['colors.hoveringNode'],
        },
      }),
    preloadedState: {
      colors: {
        ...initialColorState,
        colorProfiles: {...initialColorState.colorProfiles, ...additionalColorProfiles},
      },
    },
  });

  const persistor = persistStore(store);
  return {store, persistor};
};

type StoreAndPersistor = ReturnType<typeof createStore>;
type AppStore = StoreAndPersistor['store'];
export type AppDispatch = AppStore['dispatch'];

export * from './slices/uiSlice';
export * from './slices/profileSlice';
export * from './slices/colorsSlice';
export * from './hooks';
