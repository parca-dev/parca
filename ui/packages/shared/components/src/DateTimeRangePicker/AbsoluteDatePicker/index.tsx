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

import {useState} from 'react';

import {DateTimePicker} from '../../DateTimePicker';
import {AbsoluteDate, DateTimeRange, RelativeDate, getHistoricalDate} from '../utils';

interface AbsoluteDatePickerProps {
  range: DateTimeRange;
  onChange: (from: AbsoluteDate | RelativeDate, to: AbsoluteDate | RelativeDate) => void;
}

const AbsoluteDatePicker = ({range, onChange}: AbsoluteDatePickerProps): JSX.Element => {
  const computeFrom = (d: RelativeDate | AbsoluteDate): AbsoluteDate =>
    d.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({unit: (d as RelativeDate).unit, value: (d as RelativeDate).value})
        )
      : (d as AbsoluteDate);

  const computeTo = (d: RelativeDate | AbsoluteDate): AbsoluteDate =>
    d.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({unit: (d as RelativeDate).unit, value: (d as RelativeDate).value})
        )
      : (d as AbsoluteDate);

  const [from, setFrom] = useState<AbsoluteDate>(() => computeFrom(range.from));
  const [to, setTo] = useState<AbsoluteDate>(() => computeTo(range.to));
  const [prevRangeFrom, setPrevRangeFrom] = useState(range.from);
  const [prevRangeTo, setPrevRangeTo] = useState(range.to);

  if (prevRangeFrom !== range.from || prevRangeTo !== range.to) {
    setPrevRangeFrom(range.from);
    setPrevRangeTo(range.to);
    setFrom(computeFrom(range.from));
    setTo(computeTo(range.to));
  }

  return (
    <div className="flex flex-col w-[80%] mx-auto">
      <div className="flex flex-col justify-center gap-x-2">
        <div>
          <div className="mb-0.5 mt-1.5 text-xs">Start</div>
          <DateTimePicker
            selected={from}
            onChange={date => {
              if (date == null) {
                return;
              }
              setFrom(date);
              onChange(date, to);
            }}
          />
        </div>
        <div>
          <div className="mb-0.5 mt-1.5 text-xs">End</div>
          <DateTimePicker
            selected={to}
            onChange={date => {
              if (date == null) {
                return;
              }
              setTo(date);
              onChange(from, date);
            }}
          />
        </div>
      </div>
    </div>
  );
};

export default AbsoluteDatePicker;
