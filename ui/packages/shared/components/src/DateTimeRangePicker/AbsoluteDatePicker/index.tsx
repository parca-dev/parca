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

import {useEffect, useState} from 'react';

import {UTCDateTimePicker} from '../../DateTimePicker';
import {AbsoluteDate, DateTimeRange, RelativeDate, UNITS, getDateHoursAgo} from '../utils';

interface AbsoluteDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: AbsoluteDate | RelativeDate, to: AbsoluteDate | RelativeDate) => void;
}

const AbsoluteDatePicker = ({
  range,
  onChange = () => null,
}: AbsoluteDatePickerProps): JSX.Element => {
  const [from, setFrom] = useState<Date>(
    range.from.isRelative() ? getDateHoursAgo(1) : (range.from as AbsoluteDate).value
  );
  const [to, setTo] = useState<Date>(
    range.to.isRelative() ? getDateHoursAgo(0) : (range.to as AbsoluteDate).value
  );

  useEffect(() => {
    onChange(new AbsoluteDate(from), new AbsoluteDate(to));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to]);

  return (
    <div className="flex flex-col">
      <div className="flex justify-center gap-x-2">
        <div>
          <div className="mb-0.5 mt-1.5 text-xs">Start</div>
          <UTCDateTimePicker selected={from} onChange={date => date != null && setFrom(date)} />
        </div>
        <div>
          <div className="mb-0.5 mt-1.5 text-xs">End</div>
          <UTCDateTimePicker selected={to} onChange={date => date != null && setTo(date)} />
        </div>
      </div>
      <button
        type="button"
        className="flex"
        onClick={() => {
          const getRelativeTimeRangeBetweenDates = (timeRange: number) => {
            const roundToHundredth = value => {
              return Number(value.toFixed(2));
            };

            if (timeRange < 1000 * 60 * 60) {
              const timeRangeToMinutes = timeRange / 1000 / 60;
              return {unit: UNITS.MINUTE, value: roundToHundredth(timeRangeToMinutes)};
            }
            if (timeRange < 1000 * 60 * 60 * 24) {
              const timeRangeToHours = timeRange / 1000 / 60 / 60;
              return {unit: UNITS.HOUR, value: roundToHundredth(timeRangeToHours)};
            }
            const timeRangeToDays = timeRange / 1000 / 60 / 60 / 24;
            return {unit: UNITS.DAY, value: roundToHundredth(timeRangeToDays)};
          };

          const {unit, value} = getRelativeTimeRangeBetweenDates(to.getTime() - from.getTime());
          onChange(new RelativeDate(unit, value), new RelativeDate(unit, 0));
        }}
      >
        <p className="my-1 ml-1 text-xs text-gray-500 hover:text-indigo-600 dark:text-gray-400">
          Use relative range instead
        </p>
      </button>
    </div>
  );
};

export default AbsoluteDatePicker;
