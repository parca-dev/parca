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
            'relative flex min-w-[200px] cursor-default justify-between rounded-md border-t border-r border-b border-l px-3 py-2 text-left text-gray-600 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:text-gray-300 sm:text-sm',
            {'bg-gray-50 dark:bg-gray-900': !isActive},
            {'!justify-center, bg-gray-100 dark:bg-gray-800': isActive}
          )}
        >
          <span className="w-[147px] overflow-hidden text-ellipsis whitespace-nowrap xl:w-auto">
            {dateString}
          </span>

          <span className="cursor-pointer px-2">{!isActive ? '▼' : '▲'}</span>
        </div>
      </Popover.Button>
    </>
  );
};

export default DateTimeRangePickerTrigger;
