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

import {Fragment} from 'react';

import {Listbox, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';

import {useParcaContext} from '../ParcaContext';

export interface SelectElement {
  active: JSX.Element;
  expanded: JSX.Element;
}

export interface SelectItem {
  key: string;
  disabled?: boolean;
  element: SelectElement;
}

export function contructItemsFromArray(items: any[]): SelectItem[] {
  return items.map(item => ({
    key: item.key,
    element: {active: <>{item.label}</>, expanded: <>{item.label}</>},
  }));
}

const Select = ({
  items,
  selectedKey,
  onSelection,
  placeholder,
  width,
  className = '',
  loading,
  primary = false,
  disabled = false,
}: {
  items: SelectItem[];
  selectedKey: string | undefined;
  onSelection: (value: string) => void;
  placeholder?: string;
  width?: number;
  className?: string;
  loading?: boolean;
  primary?: boolean;
  disabled?: boolean;
}): JSX.Element => {
  const selection = items.find(v => v.key === selectedKey) ?? {
    key: selectedKey,
    element: {active: <>{selectedKey}</>, expanded: <>{selectedKey}</>},
  };
  const {loader} = useParcaContext();
  const defaultStyles =
    'relative bg-gray-50 dark:bg-gray-900 border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-sm';
  const primaryStyles =
    'relative text-gray-100 dark-gray-900 justify-center bg-indigo-600 border border-indigo-500 font-medium py-2 px-4 rounded-md shadow-sm pl-3 pr-10 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm text-sm';

  return (
    <Listbox value={selectedKey} onChange={onSelection}>
      {({open}) => (
        <div className="min-w-fit">
          <Listbox.Button
            className={cx(
              width !== undefined ? `w-${width}` : '',
              disabled ? 'opacity-50 cursor-not-allowed' : '',
              primary ? primaryStyles : defaultStyles,
              {[className]: className.length > 0}
            )}
          >
            <span className="flex items-center">
              <span className="ml-3 block overflow-x-hidden text-ellipsis">
                {selection?.key !== '' ? selection.element.active : placeholder}
              </span>
            </span>
            <span className="ml-3 absolute inset-y-0 right-0 flex items-center pr-2 pointer-events-none text-gray-400">
              <Icon icon="heroicons:chevron-up-down-20-solid" aria-hidden="true" />
            </span>
          </Listbox.Button>
          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options>
              {loading === true ? (
                <div className="absolute z-10 mt-1 bg-gray-50 dark:bg-gray-900 dark:border-gray-600 shadow-lg rounded-md py-1 text-base ring-1 ring-black dark:ring-white ring-opacity-5 dark:ring-opacity-20 overflow-auto focus:outline-none sm:text-sm">
                  <div className="w-[270px]">{loader}</div>
                </div>
              ) : (
                <>
                  {items.length > 0 && (
                    <div
                      className={cx(
                        width !== undefined ? `w-${width}` : '',
                        'absolute z-10 mt-1 bg-gray-50 dark:bg-gray-900 dark:border-gray-600 shadow-lg rounded-md py-1 text-base ring-1 ring-black dark:ring-white ring-opacity-5 dark:ring-opacity-20 overflow-auto focus:outline-none sm:text-sm'
                      )}
                    >
                      {items.map(option => (
                        <Listbox.Option
                          key={option.key}
                          disabled={option.disabled ?? false}
                          className={({active, disabled}) =>
                            cx(
                              active && 'text-white bg-indigo-600',
                              'cursor-default select-none relative py-2 pl-3 pr-9',
                              disabled && 'opacity-50'
                            )
                          }
                          value={option.key}
                        >
                          {({selected, active}) => (
                            <>
                              <div className="flex items-center">
                                <span
                                  className={cx(selected ? 'font-semibold' : 'font-normal', 'ml-3')}
                                >
                                  {option.element.expanded}
                                </span>
                              </div>
                              {selected ? (
                                <span
                                  className={cx(
                                    active ? 'text-white' : 'text-indigo-600',
                                    'absolute inset-y-0 right-0 flex items-center pr-4'
                                  )}
                                >
                                  <Icon icon="heroicons:check-20-solid" aria-hidden="true" />
                                </span>
                              ) : null}
                            </>
                          )}
                        </Listbox.Option>
                      ))}
                    </div>
                  )}
                </>
              )}
            </Listbox.Options>
          </Transition>
        </div>
      )}
    </Listbox>
  );
};

export default Select;
