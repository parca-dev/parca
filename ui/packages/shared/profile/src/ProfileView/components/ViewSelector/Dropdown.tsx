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

import {Button, useParcaContext} from '@parca/components';

export interface DropdownElement {
  active: JSX.Element;
  expanded: JSX.Element;
}

export interface DropdownItem {
  key: string;
  disabled?: boolean;
  element: DropdownElement;
  innerAction?: InnerAction;
}

export interface InnerAction {
  text: string;
  onClick: () => void;
}

export function contructItemsFromArray(items: any[]): DropdownItem[] {
  return items.map(item => ({
    key: item.key,
    element: {active: <>{item.label}</>, expanded: <>{item.label}</>},
    innerAction: item.innerAction,
  }));
}

const Dropdown = ({
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
}: {
  items: DropdownItem[];
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
}): JSX.Element => {
  const selection = items.find(v => v.key === selectedKey) ?? {
    key: selectedKey,
    element: {active: <>{selectedKey}</>, expanded: <>{selectedKey}</>},
  };
  const {loader} = useParcaContext();
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
            <Listbox.Options className="absolute w-[246px] right-0 z-50 mt-1 max-h-[50vh] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
              {loading === true ? (
                <div className="w-[270px]">{loader}</div>
              ) : (
                items.length > 0 &&
                items.map(option => (
                  <Listbox.Option
                    id={`h-select-option-${option.key}`}
                    key={option.key}
                    disabled={option.disabled ?? false}
                    className={({active}) =>
                      cx(
                        active && 'bg-indigo-600 text-white',
                        'relative flex cursor-default select-none py-2 px-3'
                      )
                    }
                    value={option.key}
                  >
                    {({selected, active, disabled}) => (
                      <div className="flex items-center w-full justify-between">
                        <div className="flex items-center">
                          <span
                            className={cx(
                              selected ? 'font-semibold' : 'font-normal',
                              disabled && 'opacity-50'
                            )}
                          >
                            {option.element.expanded}
                          </span>
                        </div>
                        {option.innerAction !== undefined && (
                          <Button
                            variant="neutral"
                            className="p-[6px]"
                            onClick={e => {
                              e.stopPropagation();
                              option.innerAction?.onClick();
                            }}
                            disabled={disabled}
                          >
                            {option.innerAction.text}
                            {option.innerAction.text === 'Add Panel' && (
                              <Icon icon="ic:baseline-plus" className="w-[14px] h-[14px] ml-2" />
                            )}
                          </Button>
                        )}
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
                      </div>
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

export default Dropdown;
