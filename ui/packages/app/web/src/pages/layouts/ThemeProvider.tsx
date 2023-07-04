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

import React, {useEffect} from 'react';

import {
  selectDarkMode,
  selectParcaThemeSystemSettings,
  setDarkMode,
  setParcaThemeSystemSettings,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';

const ThemeProvider = ({children}: {children: React.ReactNode}) => {
  const dispatch = useAppDispatch();
  const darkMode = useAppSelector(selectDarkMode);
  const isSystemSettingsTheme = useAppSelector(selectParcaThemeSystemSettings);

  //On the first load, if the system settings are set to dark mode, then set the dark mode to true.
  useEffect(() => {
    if (window.matchMedia('(prefers-color-scheme: dark)').matches && isSystemSettingsTheme) {
      document.documentElement.classList.add('dark');
      dispatch(setParcaThemeSystemSettings(true));
      dispatch(setDarkMode(true));
    }
  }, [dispatch, isSystemSettingsTheme]);

  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

  useEffect(() => {
    // Listen for changes to the prefers-color-scheme media query and then update the theme accordingly.
    mediaQuery.onchange = media => {
      // @ts-expect-error
      if (media.currentTarget.matches === true && isSystemSettingsTheme) {
        document.documentElement.classList.add('dark');
      } else if (
        // @ts-expect-error
        media.currentTarget.matches === false &&
        isSystemSettingsTheme
      ) {
        document.documentElement.classList.remove('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    };

    if (isSystemSettingsTheme) dispatch(setDarkMode(mediaQuery.matches));
  }, [mediaQuery, dispatch, isSystemSettingsTheme]);

  // This useffect is responsible for updating the theme when the user changes the theme from the dropdown in the navbar.
  useEffect(() => {
    if (isSystemSettingsTheme && window.matchMedia('(prefers-color-scheme: dark)').matches) {
      document.documentElement.classList.add('dark');
    } else if (darkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [darkMode, isSystemSettingsTheme]);

  return <div style={{minHeight: '100vh'}}>{children}</div>;
};

export default ThemeProvider;
