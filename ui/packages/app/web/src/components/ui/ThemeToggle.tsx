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

import {Icon} from '@iconify/react';

import {Dropdown, IconButton} from '@parca/components';
import {selectDarkMode, setDarkMode, useAppDispatch, useAppSelector} from '@parca/store';

const modes = [
  {
    key: 'light',
    label: 'Light',
    icon: 'heroicons:sun-20-solid',
  },

  {
    key: 'dark',
    label: 'Dark',
    icon: 'heroicons:moon-20-solid',
  },

  {
    key: 'system',
    label: 'System',
    icon: 'heroicons:computer-desktop-solid',
  },
];

const ThemeToggle = () => {
  const dispatch = useAppDispatch();
  const isDarkMode = useAppSelector(selectDarkMode);

  return (
    <div>
      <Dropdown
        text="text here"
        element={
          <IconButton
            icon={
              <Icon
                className="h-5 w-5"
                aria-hidden="true"
                icon={isDarkMode ? 'heroicons:moon-20-solid' : 'heroicons:sun-20-solid'}
              />
            }
          />
        }
      >
        {modes.map(item => (
          <Dropdown.Item key={item.key} onSelect={() => dispatch(setDarkMode(!isDarkMode))}>
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
