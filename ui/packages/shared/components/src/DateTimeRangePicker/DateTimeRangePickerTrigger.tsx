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

import {Popover} from '@headlessui/react';
import cx from 'classnames';

import {DateTimeRange, formatDateStringForUI} from './utils';

interface DateTimeRangePickerTriggerProps {
  range: DateTimeRange;
  onClick: () => void;
  isActive: boolean;
}

const DateTimeRangePickerTrigger = ({
  range,
  onClick,
  isActive,
}: DateTimeRangePickerTriggerProps): JSX.Element => {
  const dateString =
    isActive && range.from.isRelative()
      ? `${formatDateStringForUI(range.from)} → ${formatDateStringForUI(range.to)}`
      : range.getRangeStringForUI();
  return (
    <>
      <Popover.Button onClick={onClick} title={dateString}>
        <div
          onClick={onClick}
          className={cx(
            'text-gray-600 dark:text-gray-300 relative flex justify-between min-w-[200px] border-t border-r border-b border-l dark:border-gray-600 rounded-md shadow-sm px-3 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm',
            {'bg-gray-50 dark:bg-gray-900': !isActive},
            {'!justify-center, bg-gray-100 dark:bg-gray-800': isActive}
          )}
        >
          <span className="w-[147px] xl:w-auto text-ellipsis overflow-hidden whitespace-nowrap">
            {dateString}
          </span>

          <span className="px-2 cursor-pointer">{!isActive ? '▼' : '▲'}</span>
        </div>
      </Popover.Button>
    </>
  );
};

export default DateTimeRangePickerTrigger;
