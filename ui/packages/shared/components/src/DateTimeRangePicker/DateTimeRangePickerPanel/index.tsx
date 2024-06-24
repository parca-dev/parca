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

import Tab from '../../Tab';
import AbsoluteDatePicker from '../AbsoluteDatePicker';
import {RelativeDatePickerForPanel} from '../RelativeDatePicker';
import {DateTimeRange, DateUnion} from '../utils';

interface DateTimeRangePickerProps {
  range: DateTimeRange;
  onRangeSelection: (range: DateTimeRange) => void;
  togglePopoverMenu: () => void;
}

const DateTimeRangePickerPanel = ({
  range,
  onRangeSelection,
  togglePopoverMenu,
}: DateTimeRangePickerProps): JSX.Element => {
  return (
    <div className="w-[300px] rounded bg-gray-100 py-4 text-gray-600 dark:bg-gray-800 dark:text-gray-300">
      <Tab
        tabs={['Relative', 'Absolute']}
        panels={[
          <RelativeDatePickerForPanel
            onChange={(from: DateUnion, to: DateUnion) => {
              onRangeSelection(new DateTimeRange(from, to));
            }}
            range={range}
            key={0}
            hidePopoverMenu={togglePopoverMenu}
          />,
          <AbsoluteDatePicker
            range={range}
            onChange={(from: DateUnion, to: DateUnion) => {
              onRangeSelection(new DateTimeRange(from, to));
            }}
            key={1}
          />,
        ]}
        defaultTabIndex={range.from.isRelative() ? 0 : 1}
      />
    </div>
  );
};

export default DateTimeRangePickerPanel;
