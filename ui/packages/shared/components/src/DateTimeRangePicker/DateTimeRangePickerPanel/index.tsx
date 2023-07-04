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
import type {DateTimeRange, DateUnion} from '../utils';
import AbsoluteDatePicker from './AbsoluteDatePicker';
import RelativeDatePicker from './RelativeDatePicker';

interface DateTimeRangePickerProps {
  range: DateTimeRange;
  onChange?: (from: DateUnion, to: DateUnion) => void;
}

const DateTimeRangePickerPanel = ({
  range,
  onChange = () => null,
}: DateTimeRangePickerProps): JSX.Element => {
  return (
    <div className="w-[300px] rounded bg-gray-100 pt-4 text-gray-600 dark:bg-gray-800 dark:text-gray-300">
      <Tab
        tabs={['Relative', 'Absolute']}
        panels={[
          <RelativeDatePicker
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
            key={0}
          />,
          <AbsoluteDatePicker
            range={range}
            onChange={(from, to) => onChange(from as DateUnion, to as DateUnion)}
            key={1}
          />,
        ]}
        defaultTabIndex={range.from.isRelative() ? 0 : 1}
      />
    </div>
  );
};

export default DateTimeRangePickerPanel;
