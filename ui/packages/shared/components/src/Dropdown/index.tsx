import {Menu, Transition} from '@headlessui/react';
import React, {Fragment} from 'react';
import {ChevronDownIcon} from '@heroicons/react/solid';
import Button from '../Button';
import cx from 'classnames';

const Dropdown = ({
  text,
  element,
  variant = 'neutral',
  children,
}: {
  text: string;
  element?: React.ReactNode;
  variant?: 'neutral' | 'link' | 'primary' | undefined;
  children?: React.ReactNode;
}) => {
  return (
    <>
      <Menu as="div" className="relative inline-block text-left">
        <div>
          <Menu.Button as="div">
            {element !== undefined ? (
              <Button variant={variant}>
                {text}{' '}
                <ChevronDownIcon
                  className="w-5 h-5 ml-2 -mr-1 text-violet-200 hover:text-violet-100"
                  aria-hidden="true"
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
          <Menu.Items className="absolute right-0 w-56 mt-2 origin-top-right bg-white divide-y divide-gray-100 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
            <div className="px-1 py-1 ">{children}</div>
          </Menu.Items>
        </Transition>
      </Menu>
    </>
  );
};

const Item = ({children, onSelect}: {children: React.ReactNode; onSelect: () => void}) => {
  return (
    <Menu.Item>
      {({active}) => (
        <button
          className={cx(
            active ? 'bg-indigo-500 text-white' : 'text-gray-900',
            'group flex rounded-md items-center w-full px-2 py-2 text-sm'
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
