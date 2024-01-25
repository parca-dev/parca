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

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../IcicleGraphArrow';

const groupByOptions = [
  {
    value: FIELD_FUNCTION_NAME,
    label: 'Function Name',
    description: 'Stacktraces are grouped by function names.',
    disabled: true,
  },
  {
    value: FIELD_LABELS,
    label: 'Labels',
    description: 'Stacktraces are grouped by pprof labels.',
    disabled: false,
  },
  {
    value: FIELD_FUNCTION_FILE_NAME,
    label: 'Filename',
    description: 'Stacktraces are grouped by filenames.',
    disabled: false,
  },
  {
    value: FIELD_LOCATION_ADDRESS,
    label: 'Address',
    description: 'Stacktraces are grouped by addresses.',
    disabled: false,
  },
  {
    value: FIELD_MAPPING_FILE,
    label: 'Binary',
    description: 'Stacktraces are grouped by binaries.',
    disabled: false,
  },
];

const GroupByDropdown = ({
  groupBy,
  toggleGroupBy,
}: {
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
}): React.JSX.Element => {
  const label =
    groupBy.length === 0
      ? 'Nothing'
      : groupBy.length === 1
      ? groupByOptions.find(option => option.value === groupBy[0])?.label
      : 'Multiple';

  return (
    <div className="relative">
      <label className="text-sm">Group</label>
      <Menu as="div" className="relative text-left">
        <Menu.Button className="relative w-max cursor-default rounded-md border bg-white py-2 pl-3 pr-[1.7rem] text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm">
          <span className="block overflow-x-hidden text-ellipsis">{label}</span>
          <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2 text-gray-400">
            <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
          </span>
        </Menu.Button>

        <Transition
          as="div"
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Menu.Items className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  {groupByOptions.map(({value, label, description, disabled}) => (
                    <div key={value} className="relative flex items-start">
                      <div className="flex h-6 items-center">
                        <input
                          id={value}
                          name={value}
                          type="checkbox"
                          disabled={disabled}
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                          checked={groupBy.includes(value)}
                          onChange={() => toggleGroupBy(value)}
                        />
                      </div>
                      <div className="ml-3 text-sm leading-6">
                        <label
                          htmlFor={value}
                          className="font-medium text-gray-900 dark:text-gray-200"
                        >
                          {label}
                        </label>
                        <p className="text-gray-500 dark:text-gray-400">{description}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </fieldset>
            </div>
          </Menu.Items>
        </Transition>
      </Menu>
    </div>
  );
};

export default GroupByDropdown;
