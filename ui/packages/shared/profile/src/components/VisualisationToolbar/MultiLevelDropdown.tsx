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

import React from 'react';

import {Menu} from '@headlessui/react';
import {Icon} from '@iconify/react';

type MenuItemType = {
  label: string;
  items?: MenuItemType[];
};

type MenuItemProps = MenuItemType & {
  onSelect: (path: string[]) => void;
  path?: string[];
};

const MenuItem: React.FC<MenuItemProps> = ({label, items, onSelect, path = []}) => {
  const handleSelect = () => {
    if (!items) {
      onSelect([...path, label]);
    }
  };

  return (
    <div className="relative">
      <Menu>
        {({close}) => (
          <>
            <Menu.Button
              className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex justify-between items-center"
              onClick={handleSelect}
            >
              {label}
              {items && <Icon icon="flowbite:caret-right-solid" className="h-[14px] w-[14px]" />}
            </Menu.Button>
            {items && (
              <Menu.Items className="absolute left-full top-0 w-56 mt-0 origin-top-right bg-white border border-gray-200 divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
                {items.map((item, index) => (
                  <MenuItem
                    key={index}
                    {...item}
                    onSelect={selectedPath => {
                      onSelect([...path, ...selectedPath]);
                      close();
                    }}
                    path={[...path, label]}
                  />
                ))}
              </Menu.Items>
            )}
          </>
        )}
      </Menu>
    </div>
  );
};

type MultiLevelDropdownProps = {
  onSelect: (path: string[]) => void;
};

const MultiLevelDropdown: React.FC<MultiLevelDropdownProps> = ({onSelect}) => {
  const menuItems: MenuItemType[] = [
    {
      label: 'Sort by',
      items: [{label: 'Function'}, {label: 'Cumulative'}, {label: 'Diff'}],
    },
    {
      label: 'Color by',
      items: [{label: 'Function'}, {label: 'Module'}, {label: 'Filename'}],
    },
    {label: 'Show legend'},
    {label: 'Invert call stack'},
  ];

  return (
    <div className="relative inline-block text-left">
      <Menu>
        {({open}) => (
          <>
            <Menu.Button className="inline-flex justify-center w-full px-4 py-2 text-sm font-medium text-white bg-white rounded-md focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-opacity-75 border border-gray-200">
              <Icon icon="pepicons-pencil:dots-x" className="h-5 w-5 text-gray-800" />
            </Menu.Button>
            {open && (
              <>
                <span className="text-xs text-gray-400 capitalize px-2">actions</span>
                <Menu.Items className="absolute z-10 right-0 w-56 mt-2 origin-top-right bg-white border border-gray-200 divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
                  {menuItems.map((item, index) => (
                    <MenuItem key={index} {...item} onSelect={onSelect} />
                  ))}
                </Menu.Items>
              </>
            )}
          </>
        )}
      </Menu>
    </div>
  );
};

export default MultiLevelDropdown;
