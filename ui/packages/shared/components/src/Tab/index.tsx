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

import cx from 'classnames';
import {Tab as HeadlessTab} from '@headlessui/react';

const Tab = ({tabs, panels, defaultTabIndex = 0}) => {
  return (
    <HeadlessTab.Group defaultIndex={defaultTabIndex}>
      <HeadlessTab.List className="flex p-1 space-x-1 bg-gray-50 dark:bg-gray-700 text-gray-500 dark:text-gray-400 rounded-xl w-[80%] mx-auto border border-gray-300 dark:border-gray-400">
        {tabs.map((tab, idx) => (
          <HeadlessTab
            key={idx}
            className={({selected}) =>
              cx(
                'w-full py-2.5 text-sm leading-5 font-medium rounded-lg',
                'focus:outline-none focus:ring-2 ring-offset-2 ring-offset-blue-400 ring-white ring-opacity-60',
                selected
                  ? 'font-medium bg-gray-200 dark:bg-gray-900 text-gray-900 dark:text-gray-200 shadow'
                  : 'font-normal hover:bg-gray-200 hover:dark:bg-white/[0.12] hover:text-gray-800 hover:dark:text-gray-300'
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
              'focus:outline-none focus:ring-2 ring-offset-2 ring-offset-blue-400 ring-white ring-opacity-60'
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
