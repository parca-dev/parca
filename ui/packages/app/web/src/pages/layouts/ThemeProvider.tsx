import {useAppSelector, selectDarkMode} from '@parca/store';
import React, {useEffect} from 'react';

const ThemeProvider = ({children}: {children: React.ReactNode}) => {
  const darkMode = useAppSelector(selectDarkMode);

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [darkMode]);

  return <div style={{minHeight: '100vh'}}>{children}</div>;
};

export default ThemeProvider;
