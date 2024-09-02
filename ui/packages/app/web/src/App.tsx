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

import {BrowserRouter, Navigate, Route, Routes} from 'react-router-dom';
import {PersistGate} from 'redux-persist/integration/react';

import 'tailwindcss/tailwind.css';
import './style/file-input.css';
import './style/metrics.css';
import './style/profile.css';
import './style/sidenav.css';
import './style/source.css';
import './style/context-menu.css';
import './style/react-select.css';
import 'react-datepicker/dist/react-datepicker.css';

import {QueryClient, QueryClientProvider} from '@tanstack/react-query';
import {Provider} from 'react-redux';

import {createStore} from '@parca/store';

import HomePage from './pages/index';
import Component404 from './pages/layouts/Component404';
import Header from './pages/layouts/Header';
import ThemeProvider from './pages/layouts/ThemeProvider';
import SettingsPage from './pages/settings';
import TargetsPage from './pages/targets';

declare global {
  interface Window {
    PATH_PREFIX: string;
    APP_VERSION: string;
  }
}

function getBasename() {
  if (!window.PATH_PREFIX) {
    return '/';
  }
  if (window.PATH_PREFIX.startsWith('{{')) {
    return '/';
  }
  return window.PATH_PREFIX;
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
    },
  },
});

const {store: reduxStore, persistor} = createStore();

const App = () => {
  return (
    <Provider store={reduxStore}>
      <PersistGate loading={null} persistor={persistor}>
        <BrowserRouter basename={getBasename()}>
          <QueryClientProvider client={queryClient}>
            <ThemeProvider>
              <Header />
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/targets" element={<TargetsPage />} />
                <Route path="/settings" element={<SettingsPage />} />
                <Route path="/PATH_PREFIX_VAR" element={<Navigate to="/" replace />} />
                <Route path="*" element={<Component404 />} />
              </Routes>
            </ThemeProvider>
          </QueryClientProvider>
        </BrowserRouter>
      </PersistGate>
    </Provider>
  );
};

export default App;
