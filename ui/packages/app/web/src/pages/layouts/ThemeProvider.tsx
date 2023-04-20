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

import {selectDarkMode, setDarkMode, useAppDispatch, useAppSelector} from '@parca/store';

const ThemeProvider = ({children}: {children: React.ReactNode}) => {
  const dispatch = useAppDispatch();
  const darkMode = useAppSelector(selectDarkMode);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

    mediaQuery.onchange = media => {
      // @ts-expect-error
      if (localStorage['parcaDarkModeSystemSettings'] && media.currentTarget.matches) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    };

    if (localStorage['parcaDarkModeSystemSettings']) dispatch(setDarkMode(mediaQuery.matches));
  });

  useEffect(() => {
    if (
      localStorage['parcaDarkModeSystemSettings'] &&
      window.matchMedia('(prefers-color-scheme: dark)').matches
    ) {
      document.documentElement.classList.add('dark');
    } else if (darkMode) {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [darkMode]);

  return <div style={{minHeight: '100vh'}}>{children}</div>;
};

export default ThemeProvider;
