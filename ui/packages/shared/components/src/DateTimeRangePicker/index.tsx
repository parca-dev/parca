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

import {useRef, useState} from 'react';

import {Popover} from '@headlessui/react';
import {useClickAway} from 'react-use';

import {testId, TEST_IDS} from '@parca/test-utils';

import DateTimeRangePickerPanel from './DateTimeRangePickerPanel';
import DateTimeRangePickerText from './DateTimeRangePickerText';
import {DateTimeRange} from './utils';

interface DateTimeRangePickerProps {
  onRangeSelection: (range: DateTimeRange) => void;
  range: DateTimeRange;
  timezone?: string;
}

const DateTimeRangePicker = ({
  onRangeSelection,
  range,
  timezone,
}: DateTimeRangePickerProps): JSX.Element => {
  const [isActive, setIsActive] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useClickAway(containerRef, () => {
    setIsActive(false);
  });

  const toggleActive = (): void => {
    setIsActive(true);
  };

  return (
    <Popover>
      <div
        ref={containerRef}
        className="relative w-fit items-center"
        {...testId(TEST_IDS.DATE_TIME_RANGE_PICKER_CONTAINER)}
      >
        <DateTimeRangePickerText
          range={range}
          onClick={toggleActive}
          isActive={isActive}
          onRangeSelection={(range: DateTimeRange) => {
            onRangeSelection(range);
          }}
          timezone={timezone}
        />

        {isActive ? (
          <Popover.Panel
            className="absolute left-[50%] translate-x-[-50%] z-10 mt-2 w-fit rounded border shadow-lg dark:border-gray-600"
            static
            {...testId(TEST_IDS.DATE_TIME_RANGE_PICKER_PANEL)}
          >
            <DateTimeRangePickerPanel
              range={range}
              onRangeSelection={(range: DateTimeRange) => {
                onRangeSelection(range);
              }}
              togglePopoverMenu={() => {
                setIsActive(false);
              }}
            />
          </Popover.Panel>
        ) : null}
      </div>
    </Popover>
  );
};

export default DateTimeRangePicker;
export {DateTimeRange};
