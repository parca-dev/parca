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

import {Icon} from '@iconify/react';

import {AbsoluteDate, DateTimeRange, RelativeDate, UNITS, getHistoricalDate} from '../utils';

interface RelativeDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: RelativeDate | AbsoluteDate, to: RelativeDate | AbsoluteDate) => void;
}

interface UnitsMap {
  [key: string]: string;
}

const unitLong: UnitsMap = {m: UNITS.MINUTE, h: UNITS.HOUR, d: UNITS.DAY};
const unitShort: UnitsMap = {[UNITS.MINUTE]: 'm', [UNITS.HOUR]: 'h', [UNITS.DAY]: 'd'};

const presetRanges = [
  {value: 1, unit: UNITS.MINUTE},
  {value: 5, unit: UNITS.MINUTE},
  {value: 15, unit: UNITS.MINUTE},
  {value: 30, unit: UNITS.MINUTE},
  {value: 1, unit: UNITS.HOUR},
  {value: 3, unit: UNITS.HOUR},
  {value: 6, unit: UNITS.HOUR},
  {value: 12, unit: UNITS.HOUR},
  {value: 1, unit: UNITS.DAY},
  {value: 2, unit: UNITS.DAY},
];

const NOW = new RelativeDate(UNITS.MINUTE, 0);

const parseInput = (input: string): {value: number; unit: string} | null => {
  const match = input.match(/(\d+)([smhdwMy])/);
  if (match == null) {
    return null;
  }
  const value = parseInt(match[1], 10);
  const unit = match[2];
  return {value, unit: unitLong[unit]};
};

const RelativeDatePicker = ({
  range,
  onChange = () => null,
}: RelativeDatePickerProps): JSX.Element => {
  const date = range.from as RelativeDate;
  const [rangeInputString, setRangeInputString] = useState<string>(`${date.value}${date.unit}`);
  const [validRange, setValidRange] = useState<{value: number; unit: string}>({
    value: date.value,
    unit: date.unit,
  });

  useEffect(() => {
    setRangeInputString(`${validRange.value}${unitShort[validRange.unit]}`);
    onChange(new RelativeDate(validRange.unit, validRange.value), NOW);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validRange]);

  const currentRangeIndex = presetRanges.findIndex(
    ({value, unit}) => value === date.value && unit === date.unit
  );

  return (
    <div>
      <label htmlFor="range" className="text-xs">
        Range
      </label>
      <div className="flex rounded-md shadow-sm">
        <button
          type="button"
          disabled={currentRangeIndex === 0}
          className="rounded-l-md border border-r-0 border-gray-300 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400"
          onClick={() => {
            if (currentRangeIndex === -1) {
              return;
            }
            const previousRangeIndex = currentRangeIndex - 1;
            const previousRange = presetRanges[previousRangeIndex];
            setValidRange(previousRange);
          }}
        >
          <Icon icon="heroicons:minus-20-solid" />
        </button>
        <div className="flex flex-grow items-stretch focus-within:z-10">
          <input
            type="text"
            name="range"
            id="range"
            className="flex w-full text-center text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6"
            placeholder="6h"
            value={rangeInputString}
            onBlur={() => {
              // parse the input value and check whether it is valid
              // if it is valid, set the range
              // if it is not valid, set the range to the previous value

              const parsedInput = parseInput(rangeInputString);

              if (parsedInput === null) {
                setRangeInputString(`${validRange.value}${unitShort[validRange.unit]}`);
                return;
              }
              const {value, unit} = parsedInput;
              setValidRange({value, unit});
            }}
          />
        </div>
        <button
          type="button"
          disabled={currentRangeIndex === presetRanges.length - 1}
          className="rounded-r-md border border-l-0 border-gray-300 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400"
          onClick={() => {
            if (currentRangeIndex === -1) {
              return;
            }
            const nextRangeIndex = currentRangeIndex + 1;
            const nextRange = presetRanges[nextRangeIndex];
            setValidRange(nextRange);
          }}
        >
          <Icon icon="heroicons:plus-20-solid" />
        </button>
      </div>
      <button
        type="button"
        className="flex"
        onClick={() => {
          onChange(
            new AbsoluteDate(getHistoricalDate(validRange)),
            new AbsoluteDate(getHistoricalDate(NOW))
          );
        }}
      >
        <p className="text-center text-xs text-gray-500">Use absolute start time instead</p>
      </button>
    </div>
  );
};

export default RelativeDatePicker;
