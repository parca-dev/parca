import {BrowserRouter, Navigate, Route, Routes} from 'react-router-dom';
import {PersistGate} from 'redux-persist/integration/react';
import store from './store';
import 'tailwindcss/tailwind.css';
import './style/file-input.css';
import './style/metrics.css';
import './style/profile.css';
import './style/sidenav.css';
import Header from './pages/layouts/Header';
import ThemeProvider from './pages/layouts/ThemeProvider';
import HomePage from './pages/index';
import TargetsPage from './pages/targets';
import Component404 from './pages/layouts/Component404';
import {isDevMode} from '@parca/functions';
import {Provider} from 'react-redux';

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

const {store: reduxStore, persistor} = store();

const App = () => {
  return (
    <Provider store={reduxStore}>
      <PersistGate loading={null} persistor={persistor}>
        <BrowserRouter basename={getBasename()}>
          <ThemeProvider>
            <Header />
            <div className="px-3">
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/targets" element={<TargetsPage />} />
                {isDevMode() && (
                  <Route path="/PATH_PREFIX_VAR" element={<Navigate to="/" replace />} />
                )}
                <Route path="*" element={<Component404 />} />
              </Routes>
            </div>
          </ThemeProvider>
        </BrowserRouter>
      </PersistGate>
    </Provider>
  );
};

export default App;
