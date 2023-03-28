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

import {Tab as HeadlessTab} from '@headlessui/react';
import cx from 'classnames';

const Tab = ({
  tabs,
  panels,
  defaultTabIndex = 0,
}: {
  tabs: string[];
  panels: JSX.Element[];
  defaultTabIndex: number;
}): JSX.Element => {
  return (
    <HeadlessTab.Group defaultIndex={defaultTabIndex}>
      <HeadlessTab.List className="mx-auto flex w-[80%] space-x-1 rounded-xl border border-gray-300 bg-gray-50 p-1 text-gray-500 dark:border-gray-400 dark:bg-gray-700 dark:text-gray-400">
        {tabs.map((tab, idx) => (
          <HeadlessTab
            key={idx}
            className={({selected}) =>
              cx(
                'w-full rounded-lg py-2.5 text-sm font-medium leading-5',
                'ring-white ring-opacity-60 ring-offset-2 ring-offset-blue-400 focus:outline-none focus:ring-2',
                selected
                  ? 'bg-gray-200 font-medium text-gray-900 shadow dark:bg-gray-900 dark:text-gray-200'
                  : 'font-normal hover:bg-gray-200 hover:text-gray-800 hover:dark:bg-white/[0.12] hover:dark:text-gray-300'
              )
            }
          >
            {tab}
          </HeadlessTab>
        ))}
      </HeadlessTab.List>
      <HeadlessTab.Panels className="mt-2">
        {panels.map((panel, idx) => (
          <HeadlessTab.Panel
            key={idx}
            className={cx(
              'rounded',
              'ring-white ring-opacity-60 ring-offset-2 ring-offset-blue-400 focus:outline-none focus:ring-2'
            )}
          >
            {panel}
          </HeadlessTab.Panel>
        ))}
      </HeadlessTab.Panels>
    </HeadlessTab.Group>
  );
};

export default Tab;
