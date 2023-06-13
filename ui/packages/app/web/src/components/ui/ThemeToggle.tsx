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

import {useCallback, useEffect, useState} from 'react';

import {Icon} from '@iconify/react';

import {Dropdown, IconButton} from '@parca/components';
import {
  selectDarkMode,
  selectParcaThemeSystemSettings,
  setDarkMode,
  setParcaThemeSystemSettings,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';

const ThemeToggle = () => {
  const dispatch = useAppDispatch();
  const isDarkMode = useAppSelector(selectDarkMode);
  const isSystemSettingsTheme = useAppSelector(selectParcaThemeSystemSettings);
  const [systemSettingsDarkMode, setSystemSettingsDarkMode] = useState(false);

  const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

  useEffect(() => {
    // Listen for changes to the prefers-color-scheme media query and then update the theme accordingly.
    mediaQuery.onchange = media => {
      if (mediaQuery.matches === true) {
        setSystemSettingsDarkMode(true);
      } else {
        setSystemSettingsDarkMode(false);
      }
    };
  }, [mediaQuery]);

  const conditionalDarkMode = useCallback(() => {
    if (isSystemSettingsTheme && systemSettingsDarkMode) {
      return true;
    } else {
      return isDarkMode;
    }
  }, [isSystemSettingsTheme, systemSettingsDarkMode, isDarkMode]);

  const updateWithSystemSettings = () => {
    dispatch(setDarkMode(systemSettingsDarkMode));
    dispatch(setParcaThemeSystemSettings(true));
  };

  const updateModesOnly = (value: boolean) => {
    dispatch(setDarkMode(value));
    dispatch(setParcaThemeSystemSettings(false));
  };

  const modes = [
    {
      key: 'light',
      label: 'Light',
      icon: 'heroicons:sun-20-solid',
      onSelect: () => updateModesOnly(false),
      selected: !isSystemSettingsTheme && !isDarkMode,
    },

    {
      key: 'dark',
      label: 'Dark',
      icon: 'heroicons:moon-20-solid',
      onSelect: () => updateModesOnly(true),
      selected: !isSystemSettingsTheme && isDarkMode,
    },

    {
      key: 'system',
      label: 'System',
      icon: 'heroicons:computer-desktop-solid',
      onSelect: updateWithSystemSettings,
      selected: isSystemSettingsTheme,
    },
  ];

  return (
    <div>
      <Dropdown
        element={
          <IconButton
            icon={
              <Icon
                className="h-5 w-5"
                aria-hidden="true"
                icon={conditionalDarkMode() ? 'heroicons:moon-20-solid' : 'heroicons:sun-20-solid'}
              />
            }
          />
        }
      >
        {modes.map(item => (
          <Dropdown.Item key={item.key} onSelect={item.onSelect} selected={item.selected}>
            <div className="flex items-center">
              <span className="mr-2">
                <Icon icon={item.icon} />
              </span>
              {item.label}
            </div>
          </Dropdown.Item>
        ))}
      </Dropdown>
    </div>
  );
};

export default ThemeToggle;
