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

import React, {Fragment} from 'react';

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';

import {Button} from '../Button';

const Dropdown = ({
  text,
  element,
  variant = 'neutral',
  children,
}: {
  text?: string;
  element?: React.ReactNode;
  variant?: 'neutral' | 'link' | 'primary' | undefined;
  children?: React.ReactNode;
}): JSX.Element => {
  return (
    <>
      <Menu as="div" className="relative z-[999999999] inline-block text-left">
        <div>
          <Menu.Button as="div">
            {element === undefined ? (
              <Button variant={variant}>
                {text}{' '}
                <Icon
                  icon="heroicons:chevron-down-20-solid"
                  aria-hidden="true"
                  className="text-violet-200 hover:text-violet-100"
                />
              </Button>
            ) : (
              <div className="cursor-pointer">{element}</div>
            )}
          </Menu.Button>
        </div>
        <Transition
          as={Fragment}
          enter="transition ease-out duration-100"
          enterFrom="transform opacity-0 scale-95"
          enterTo="transform opacity-100 scale-100"
          leave="transition ease-in duration-75"
          leaveFrom="transform opacity-100 scale-100"
          leaveTo="transform opacity-0 scale-95"
        >
          <Menu.Items className="absolute right-0 mt-2 w-36 origin-top-right divide-y divide-gray-100 rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-gray-900">
            <div className="px-1 py-1 ">{children}</div>
          </Menu.Items>
        </Transition>
      </Menu>
    </>
  );
};

const Item = ({
  children,
  onSelect,
  selected,
}: {
  children: React.ReactNode;
  onSelect: () => void;
  selected?: boolean;
}): JSX.Element => {
  return (
    <Menu.Item>
      {({active}) => (
        <button
          className={cx(
            active ? 'bg-indigo-500 text-white' : 'text-gray-900 dark:text-white',
            'group mb-px flex w-full items-center rounded-md px-2 py-2 text-sm',
            selected != null && selected ? 'bg-indigo-500 font-bold !text-white' : ''
          )}
          onClick={onSelect}
        >
          {children}
        </button>
      )}
    </Menu.Item>
  );
};

export default Object.assign(Dropdown, {
  Item,
});
