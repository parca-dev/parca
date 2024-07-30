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

import {Fragment, useState} from 'react';

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
  icon,
  id,
  optionsClassname = '',
  searchable = false,
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
  icon?: JSX.Element;
  id?: string;
  optionsClassname?: string;
  searchable?: boolean;
}): JSX.Element => {
  const selection = items.find(v => v.key === selectedKey) ?? {
    key: selectedKey,
    element: {active: <>{selectedKey}</>, expanded: <>{selectedKey}</>},
  };
  const {loader} = useParcaContext();
  const [searchTerm, setSearchTerm] = useState('');

  const filteredItems = searchable
    ? items.filter(item =>
        item.element.active.props.children
          .toString()
          .toLowerCase()
          .includes(searchTerm.toLowerCase())
      )
    : items;

  const styles =
    'relative border rounded-md shadow-sm px-4 py-2 text-left cursor-default focus:outline-none focus:ring-1 items-center focus:ring-indigo-500 focus:border-indigo-500 text-sm flex gap-2 flex items-center justify-between';
  const defaultStyles = 'bg-white dark:bg-gray-900 dark:border-gray-600';
  const primaryStyles =
    'text-gray-100 dark:gray-900 bg-indigo-600 border-indigo-500 font-medium py-2 px-4';

  return (
    <Listbox value={selectedKey} onChange={onSelection}>
      {({open}) => (
        <div className="relative">
          <div id={id}>
            <Listbox.Button
              className={cx(
                styles,
                width !== undefined ? `w-${width}` : 'w-full',
                disabled ? 'cursor-not-allowed opacity-50 pointer-events-none' : '',
                primary ? primaryStyles : defaultStyles,
                {[className]: className.length > 0}
              )}
            >
              <div className={cx(icon != null ? '' : 'block overflow-x-hidden text-ellipsis')}>
                {selection?.key !== '' ? selection.element.active : placeholder}
              </div>
              <div className={cx(icon != null ? '' : 'pointer-events-none text-gray-400')}>
                {icon ?? <Icon icon="heroicons:chevron-up-down-20-solid" aria-hidden="true" />}
              </div>
            </Listbox.Button>
          </div>

          <Transition
            show={open}
            as={Fragment}
            leave="transition ease-in duration-100"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <Listbox.Options
              className={cx(
                'absolute z-50 mt-1 max-h-[50vh] w-max overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm',
                {[optionsClassname]: optionsClassname.length > 0}
              )}
            >
              {searchable && (
                <div className="sticky h-[45px] z-10 top-[-5px] border-b border-gray-200">
                  <input
                    type="text"
                    className="w-full px-6 h-full text-sm border-none rounded-none ring-0 outline-none bg-gray-50 dark:bg-gray-800 dark:text-white"
                    placeholder="Search..."
                    value={searchTerm}
                    onChange={e => setSearchTerm(e.target.value)}
                  />
                </div>
              )}
              {loading === true ? (
                <div className="w-[270px]">{loader}</div>
              ) : (
                filteredItems.length > 0 &&
                filteredItems.map(option => (
                  <Listbox.Option
                    id={`h-select-option-${option.key}`}
                    key={option.key}
                    disabled={option.disabled ?? false}
                    className={({active, disabled}) =>
                      cx(
                        active && 'bg-indigo-600 text-white',
                        'relative cursor-default select-none py-2 pl-3 pr-9',
                        disabled && 'opacity-50',
                        ''
                      )
                    }
                    value={option.key}
                  >
                    {({selected, active}) => (
                      <>
                        <div className="flex items-center">
                          <span className={cx(selected ? 'font-semibold' : 'font-normal', 'ml-3')}>
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
                ))
              )}
            </Listbox.Options>
          </Transition>
        </div>
      )}
    </Listbox>
  );
};

export default Select;
