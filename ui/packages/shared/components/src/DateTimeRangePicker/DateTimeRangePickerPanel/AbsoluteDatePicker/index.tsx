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

import {Button} from '../../../Button';
import {UTCDateTimePicker} from '../../../DateTimePicker';
import {AbsoluteDate, DateTimeRange, getDateHoursAgo} from '../../utils';

interface AbsoluteDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: AbsoluteDate, to: AbsoluteDate) => void;
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
  return (
    <div className="p-4">
      <div className="mb-2 hidden">
        <span className="text-xs uppercase">Absolute Range</span>
      </div>
      <div className="flex flex-col justify-center">
        <div className="mb-2">
          <div className="mb-2">
            <span className="text-xs uppercase">From:</span>
          </div>
          <UTCDateTimePicker selected={from} onChange={date => date != null && setFrom(date)} />
        </div>
        <div className="mb-1">
          <div className="mb-2">
            <span className="text-xs uppercase">To:</span>
          </div>
          <UTCDateTimePicker selected={to} onChange={date => date != null && setTo(date)} />
        </div>
      </div>
      <div className="mt-4 flex">
        <div className="mx-auto">
          <Button
            onClick={() => {
              onChange(new AbsoluteDate(from), new AbsoluteDate(to));
            }}
          >
            Apply
          </Button>
        </div>
      </div>
      <p className="m-4 text-center text-xs italic text-gray-500">
        Note: All date and time values are in UTC.
      </p>
    </div>
  );
};

export default AbsoluteDatePicker;
