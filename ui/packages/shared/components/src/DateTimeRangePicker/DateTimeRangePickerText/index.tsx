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

import { testId } from '@parca/test-utils';

import RelativeDatePicker from '../RelativeDatePicker';
import {DateTimeRange, DateUnion} from '../utils';

interface DateTimeRangePickerTextProps {
  range: DateTimeRange;
  onClick: () => void;
  isActive: boolean;
  onRangeSelection: (range: DateTimeRange) => void;
}

const DateTimeRangePickerText = ({
  range,
  onClick,
  isActive,
  onRangeSelection,
}: DateTimeRangePickerTextProps): JSX.Element => {
  const isRelativeRange = range.from.isRelative();
  const dateString = range.getRangeStringForUI();

  return (
    <div {...testId('DATE_TIME_RANGE_PICKER_TEXT')}>
      {isRelativeRange ? (
        <RelativeDatePicker
          range={range}
          onChange={(from: DateUnion, to: DateUnion) => {
            onRangeSelection(new DateTimeRange(from, to));
          }}
          toggleRangePickerPanel={onClick}
        />
      ) : (
        <div>
          <label htmlFor="range" className="text-xs" {...testId('DATE_TIME_RANGE_LABEL')}>
            Range
          </label>
          <div
            onClick={onClick}
            className={cx(
              'relative flex min-w-[300px] cursor-default justify-between rounded-md border px-3 py-2 text-left text-gray-600 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:text-gray-300 sm:text-sm',
              {'bg-white dark:bg-gray-900': !isActive},
              {'!justify-center, bg-gray-100 dark:bg-gray-800 ': isActive}
            )}
            {...testId('DATE_TIME_RANGE_PICKER_BUTTON')}
          >
            <span className="w-[147px] overflow-hidden text-ellipsis whitespace-nowrap xl:w-auto">
              {dateString}
            </span>

            <span className="cursor-pointer px-2">{!isActive ? '▼' : '▲'}</span>
          </div>
        </div>
      )}
    </div>
  );
};

export default DateTimeRangePickerText;
