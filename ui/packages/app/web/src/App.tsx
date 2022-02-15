import {BrowserRouter, Link, Navigate, Route, Routes} from 'react-router-dom';
import {StoreProvider, useCreateStore} from './store';

import 'tailwindcss/tailwind.css';
import './style/file-input.css';
import './style/metrics.css';
import './style/profile.css';
import './style/sidenav.css';
import Header from './pages/layouts/Header';
import ThemeProvider from './pages/layouts/ThemeProvider';
import HomePage from './pages/index';
import Component404 from './pages/layouts/Component404';
import {isDevMode} from '@parca/functions';

declare global {
  interface Window {
    PATH_PREFIX: string;
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

const App = () => {
  const createStore = useCreateStore();

  return (
    <StoreProvider createStore={createStore}>
      <BrowserRouter basename={getBasename()}>
        <ThemeProvider>
          <Header />
          <div className="px-3">
            <Routes>
              <Route path="/" element={<HomePage />}></Route>
              {isDevMode() && (
                <Route path="/PATH_PREFIX_VAR" element={<Navigate to="/" replace />} />
              )}
              <Route path="*" element={<Component404 />} />
            </Routes>
          </div>
        </ThemeProvider>
      </BrowserRouter>
    </StoreProvider>
  );
};

export default App;
