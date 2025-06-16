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

import React, {Fragment, useEffect, useRef, useState} from 'react';

import {Listbox, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';

import {type SelectElement, type SelectItem} from '@parca/components';

import {groupByOptions} from '../ActionButtons/GroupByDropdown';

interface Props {
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
  groupByOptions: Array<{
    value: string;
    label: string;
    disabled?: boolean;
  }>;
}

function group(name: string): SelectElement {
  const groupByName = groupByOptions.find(option => option.value === name);

  const title = groupByName?.label.replace(/ /g, '\u00a0');
  return {
    active: <>{title}</>,
    expanded: (
      <>
        <span>{title}</span>
        <br />
      </>
    ),
  };
}

const LevelsDropdownSelect: React.FC<Props> = ({groupBy, toggleGroupBy, groupByOptions}) => {
  const groupByOptionsItems = groupByOptions.map(option => ({
    key: option.value,
    element: group(option.value),
    disabled: option.disabled,
    humanReadableKey: option.label,
  }));

  const getSelection = (): SelectItem | null => {
    if (groupBy.length === 0) {
      return null;
    }
    if (groupBy.length === 1) {
      const item = groupByOptionsItems.find(v => v.key === groupBy[0]);
      return (
        item ?? {
          key: groupBy[0],
          element: {active: <>{groupBy[0]}</>, expanded: <>{groupBy[0]}</>},
        }
      );
    }
    // Multiple items selected - show comma-separated human readable names
    const displayText = 'Multiple Selection';
    return {
      key: displayText,
      element: {active: <>{displayText}</>, expanded: <>{displayText}</>},
    };
  };

  const selection = getSelection();

  const styles =
    'relative border rounded-md shadow-sm px-4 py-2 text-left cursor-default focus:outline-none focus:ring-1 items-center focus:ring-indigo-500 focus:border-indigo-500 text-sm flex gap-2 flex items-center justify-between';
  const defaultStyles = 'bg-white dark:bg-gray-900 dark:border-gray-600';

  const labelsButton = useRef<HTMLDivElement>(null);
  const labelSelectorRef = useRef<HTMLDivElement>(null);
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent): void => {
      if (
        isOpen &&
        labelSelectorRef.current !== null &&
        !labelSelectorRef.current.contains(event.target as Node) &&
        labelsButton.current !== null &&
        !labelsButton.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  return (
    <div className="flex flex-col relative">
      <span className="text-sm">Levels</span>
      <Listbox value={groupBy}>
        {({open}) => (
          <div className="relative">
            <div id="parca-group-by-dropdown-select">
              <Listbox.Button className={cx(styles, 'w-full', defaultStyles)}>
                <div className={cx('block overflow-x-hidden text-ellipsis whitespace-nowrap')}>
                  {selection != null && selection.key !== ''
                    ? selection.element.active
                    : 'Select group by'}
                </div>
                <div className={cx('pointer-events-none text-gray-400')}>
                  <Icon icon="heroicons:chevron-up-down-20-solid" aria-hidden="true" />
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
                  'absolute z-50 mt-1 pt-0 max-h-[50vh] w-max overflow-hidden rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm'
                )}
              >
                <div className="relative">
                  <div>
                    {groupByOptionsItems.length > 0 &&
                      groupByOptionsItems.map(option => (
                        <Listbox.Option
                          id={`h-select-option-${option.key}`}
                          key={option.key}
                          disabled={option.disabled ?? false}
                          className={({active, disabled}) =>
                            cx(
                              active && 'bg-indigo-600 text-white',
                              'relative cursor-default select-none py-2 pl-3 pr-9',
                              disabled && 'opacity-50'
                            )
                          }
                          value={option.key}
                          onClick={() => toggleGroupBy(option.key)}
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
                              {Array.isArray(groupBy) && groupBy.includes(option.key) ? (
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
                </div>
              </Listbox.Options>
            </Transition>
          </div>
        )}
      </Listbox>
    </div>
  );
};

export default LevelsDropdownSelect;
