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
import cx from 'classnames';
import {Popover} from '@headlessui/react';
import {DateTimeRange, DateUnion} from './utils';
import {useClickAway} from 'react-use';
import DateTimeRangePickerTrigger from './DateTimeRangePickerTrigger';
import DateTimeRangePickerPanel from './DateTimeRangePickerPanel';

interface DateTimeRangePickerProps {
  onRangeSelection: (range: DateTimeRange) => void;
  range: DateTimeRange;
}

const DateTimeRangePicker = ({onRangeSelection, range}: DateTimeRangePickerProps): JSX.Element => {
  const [isActive, setIsActive] = useState<boolean>(false);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickAway(containerRef, () => {
    setIsActive(false);
  });

  return (
    <Popover>
      <div ref={containerRef} className="items-center w-fit">
        <DateTimeRangePickerTrigger
          range={range}
          isActive={isActive}
          onClick={() => {
            setIsActive(!isActive);
          }}
        />
        {isActive ? (
          <Popover.Panel
            className={cx(
              'absolute z-10 w-fit mt-2 rounded shadow-lg ring-1 ring-black ring-opacity-5 border dark:border-gray-600'
            )}
            static
          >
            <DateTimeRangePickerPanel
              range={range}
              onChange={(from: DateUnion, to: DateUnion) => {
                onRangeSelection(new DateTimeRange(from, to));
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
