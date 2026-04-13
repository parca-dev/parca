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

import {Menu} from '@headlessui/react';
import {Icon} from '@iconify/react';

import {useParcaContext, useURLState, useURLStateBatch} from '@parca/components';

const MoreDropdown = ({functionName}: {functionName: string}): React.JSX.Element | null => {
  const [_, setSandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const {enableSandwichView} = useParcaContext();
  const batchUpdates = useURLStateBatch();

  const onSandwichViewSelect = (): void => {
    // Batch updates to combine setSandwichFunctionName + setDashboardItems into single URL navigation
    batchUpdates(() => {
      setSandwichFunctionName(functionName.trim());
      setDashboardItems([...dashboardItems, 'sandwich']);
    });
  };

  const menuItems: Array<{label: string; action: () => void}> = [];

  if (enableSandwichView === true) {
    menuItems.push({
      label: 'Show in Sandwich view',
      action: () => onSandwichViewSelect(),
    });
  }

  if (menuItems.length === 0) {
    return null;
  }

  return (
    <div className="relative">
      <Menu>
        {({open, close}) => (
          <>
            <Menu.Button
              onClick={() => {
                if (open) {
                  close();
                }
              }}
              className="inline-flex font-sans dark:bg-gray-900 dark:border-gray-600 justify-center w-full text-sm font-normal text-gray-600 dark:text-gray-200 bg-white rounded-md focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-opacity-75"
            >
              <Icon icon="mdi:dots-horizontal" />
            </Menu.Button>
            {open && (
              <Menu.Items className="font-sans absolute z-30 left-0 w-56 mt-2 py-2 origin-top-right bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none border dark:bg-gray-900 dark:border-gray-600">
                <span className="text-xs text-gray-400 capitalize px-4 py-3">actions</span>

                {menuItems.map(item => (
                  <Menu.Button
                    key={item.label}
                    className="group mb-px flex w-full items-center rounded-md px-4 py-2 text-sm text-gray-900 dark:text-white hover:bg-indigo-500 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
                    onClick={item.action}
                  >
                    {item.label}
                  </Menu.Button>
                ))}
              </Menu.Items>
            )}
          </>
        )}
      </Menu>
    </div>
  );
};

export default MoreDropdown;
