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

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';

const RuntimeToggle = ({
  id,
  state,
  toggle,
  label,
  description,
}: {
  id: string;
  state: boolean;
  toggle: () => void;
  label: string;
  description: string;
}): JSX.Element => {
  return (
    <div key={id} className="relative flex items-start">
      <div className="flex h-6 items-center">
        <input
          id={id}
          name={id}
          type="checkbox"
          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
          checked={state}
          onChange={() => toggle()}
        />
      </div>
      <div className="ml-3 text-sm leading-6">
        <label htmlFor={id} className="font-medium text-gray-900 dark:text-gray-200">
          {label}
        </label>
        <p className="text-gray-500 dark:text-gray-400">{description}</p>
      </div>
    </div>
  );
};

const RuntimeFilterDropdown = ({
  showRuntimeRuby,
  toggleShowRuntimeRuby,
  showRuntimePython,
  toggleShowRuntimePython,
  showInterpretedOnly,
  toggleShowInterpretedOnly,
}: {
  showRuntimeRuby: boolean;
  toggleShowRuntimeRuby: () => void;
  showRuntimePython: boolean;
  toggleShowRuntimePython: () => void;
  showInterpretedOnly: boolean;
  toggleShowInterpretedOnly: () => void;
}): React.JSX.Element => {
  return (
    <div>
      <label className="text-sm">Runtimes</label>
      <Menu as="div" className="relative text-left">
        <div>
          <Menu.Button
            className="relative w-full cursor-default rounded-md border bg-white py-2 pl-3 pr-[1.7rem] text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm"
            id="h-runtimes-filter"
          >
            <span className="block overflow-x-hidden text-ellipsis">Runtimes</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 ml-3 flex items-center pr-2 text-gray-400">
              <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
            </span>
          </Menu.Button>
        </div>

        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Menu.Items className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  <RuntimeToggle
                    id="show-runtime-ruby"
                    state={showRuntimeRuby}
                    toggle={toggleShowRuntimeRuby}
                    label="Ruby"
                    description="Show Ruby runtime functions."
                  />
                  <RuntimeToggle
                    id="show-runtime-python"
                    state={showRuntimePython}
                    toggle={toggleShowRuntimePython}
                    label="Python"
                    description="Show Python runtime functions."
                  />
                  <RuntimeToggle
                    id="show-interpreted-only"
                    state={showInterpretedOnly}
                    toggle={toggleShowInterpretedOnly}
                    label="Interpreted Only"
                    description="Show only interpreted functions."
                  />
                </div>
              </fieldset>
            </div>
          </Menu.Items>
        </Transition>
      </Menu>
    </div>
  );
};

export default RuntimeFilterDropdown;
