import {BrowserRouter, Route, Routes} from 'react-router-dom';
import {StoreProvider, useCreateStore} from './store';

import 'tailwindcss/tailwind.css';
import './style/file-input.css';
import './style/metrics.css';
import './style/profile.css';
import './style/sidenav.css';
import Header from './pages/layouts/Header';
import ThemeProvider from './pages/layouts/ThemeProvider';
import HomePage from './pages/index';

const App = () => {
  const createStore = useCreateStore();

  return (
    <StoreProvider createStore={createStore}>
      <BrowserRouter>
        <ThemeProvider>
          <Header />
          <div className="px-3">
            <Routes>
              <Route path="/" element={<HomePage />}></Route>
            </Routes>
          </div>
        </ThemeProvider>
      </BrowserRouter>
    </StoreProvider>
  );
};

export default App;
