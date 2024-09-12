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

import {useEffect, useMemo, useState} from 'react';

import {DateTimePicker} from '../../DateTimePicker';
import {AbsoluteDate, DateTimeRange, RelativeDate, getHistoricalDate} from '../utils';

interface AbsoluteDatePickerProps {
  range: DateTimeRange;
  onChange: (from: AbsoluteDate | RelativeDate, to: AbsoluteDate | RelativeDate) => void;
}

const AbsoluteDatePicker = ({range, onChange}: AbsoluteDatePickerProps): JSX.Element => {
  const dateFromInRelative = useMemo(() => range.from as RelativeDate, [range.from]);
  const dateToInRelative = useMemo(() => range.to as RelativeDate, [range.to]);

  const [from, setFrom] = useState<AbsoluteDate>(
    range.from.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({
            unit: dateFromInRelative.unit,
            value: dateFromInRelative.value,
          })
        )
      : (range.from as AbsoluteDate)
  );
  const [to, setTo] = useState<AbsoluteDate>(
    range.to.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({
            unit: dateToInRelative.unit,
            value: dateToInRelative.value,
          })
        )
      : (range.to as AbsoluteDate)
  );

  useEffect(() => {
    setFrom(
      range.from.isRelative()
        ? new AbsoluteDate(
            getHistoricalDate({
              unit: dateFromInRelative.unit,
              value: dateFromInRelative.value,
            })
          )
        : (range.from as AbsoluteDate)
    );
    setTo(
      range.to.isRelative()
        ? new AbsoluteDate(
            getHistoricalDate({
              unit: dateToInRelative.unit,
              value: dateToInRelative.value,
            })
          )
        : (range.to as AbsoluteDate)
    );
  }, [dateFromInRelative, dateToInRelative, range.from, range.to]);

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
