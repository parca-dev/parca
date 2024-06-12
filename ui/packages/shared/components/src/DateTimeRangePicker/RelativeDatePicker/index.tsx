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

import {
  AbsoluteDate,
  DateTimeRange,
  RelativeDate,
  UNITS,
  UNIT_TYPE,
  getHistoricalDate,
} from '../utils';

interface RelativeDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: RelativeDate | AbsoluteDate, to: RelativeDate | AbsoluteDate) => void;
  toggleRangePickerPanel: () => void;
}

interface UnitsMap {
  [key: string]: string;
}

const unitLong: UnitsMap = {
  m: UNITS.MINUTE,
  h: UNITS.HOUR,
  d: UNITS.DAY,
  w: UNITS.WEEK,
  y: UNITS.YEAR,
};
const unitShort: UnitsMap = {
  [UNITS.MINUTE]: 'm',
  [UNITS.HOUR]: 'h',
  [UNITS.DAY]: 'd',
  [UNITS.WEEK]: 'w',
  [UNITS.YEAR]: 'y',
};

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
  {value: 1, unit: UNITS.WEEK},
  {value: 2, unit: UNITS.WEEK},
  {value: 4, unit: UNITS.WEEK},
  {value: 8, unit: UNITS.WEEK},
  {value: 16, unit: UNITS.WEEK},
  {value: 26, unit: UNITS.WEEK},
  {value: 1, unit: UNITS.YEAR},
  {value: 2, unit: UNITS.YEAR},
];

const NOW = new RelativeDate(UNITS.MINUTE, 0);

const parseInput = (input: string): {value: number; unit: string} | null => {
  const value = parseFloat(input);
  const match = input.match(/(\d+)([mhd])/);

  // handle parseFloat edge cases and non-valid input
  if (Number.isNaN(value) || input.includes('Infinity') || input.includes('e') || match == null) {
    return null;
  }

  const unit = match[2];
  return {value, unit: unitLong[unit]};
};

export const RelativeDatePickerForPanel = ({
  onChange = () => null,
}: {
  onChange: (from: RelativeDate | AbsoluteDate, to: RelativeDate | AbsoluteDate) => void;
}): JSX.Element => {
  return (
    <div className="flex flex-col gap-4 items-center text-sm p-4">
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.MINUTE, 15), NOW);
        }}
      >
        Last 15 minutes
      </div>
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.HOUR, 1), NOW);
        }}
      >
        Last 1 hour
      </div>
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.HOUR, 3), NOW);
        }}
      >
        Last 3 hours
      </div>
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.HOUR, 6), NOW);
        }}
      >
        Last 6 hours
      </div>
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.HOUR, 12), NOW);
        }}
      >
        Last 12 hours
      </div>
      <div
        className="cursor-pointer"
        onClick={() => {
          onChange(new RelativeDate(UNITS.DAY, 1), NOW);
        }}
      >
        Last 1 day
      </div>
    </div>
  );
};

const RelativeDatePicker = ({
  range,
  onChange = () => null,
  toggleRangePickerPanel,
}: RelativeDatePickerProps): JSX.Element => {
  const date = range.from as RelativeDate;
  const [rangeInputString, setRangeInputString] = useState<string>(
    `${date.value}${unitShort[date.unit]}`
  );
  const [validRange, setValidRange] = useState<{
    value: number;
    unit: string;
  }>({
    value: date.value,
    unit: date.unit,
  });

  useEffect(() => {
    if (date.value === validRange.value && date.unit === validRange.unit) {
      return;
    }
    setRangeInputString(`${validRange.value}${unitShort[validRange.unit]}`);
    onChange(new RelativeDate(validRange.unit, validRange.value), NOW);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validRange]);

  const getMultiplyFactor = (unit: UNIT_TYPE): number => {
    switch (unit) {
      case UNITS.HOUR:
        return 60;
      case UNITS.DAY:
        return 24 * 60;
      case UNITS.WEEK:
        return 7 * 24 * 60;
      case UNITS.YEAR:
        return 365 * 24 * 60;
      case UNITS.MINUTE:
      default:
        return 1;
    }
  };

  const getClosestPresetIndex = (): number => {
    const currentPresetIndex = presetRanges.findIndex(
      ({value, unit}) => value === date.value && unit === date.unit
    );

    if (currentPresetIndex !== -1) {
      return currentPresetIndex;
    }

    const presetRangesTotalMinutes = presetRanges.map(({value, unit}) => {
      const multiplyFactor = getMultiplyFactor(unit);
      return value * multiplyFactor;
    });

    const currentTotalMinutes = getMultiplyFactor(date.unit) * date.value;
    const closestPresetIndex = [...presetRangesTotalMinutes, currentTotalMinutes]
      .sort((a, b) => a - b)
      .findIndex(totalMinutes => {
        return totalMinutes === currentTotalMinutes;
      });

    return closestPresetIndex;
  };

  const currentPresetIndex = getClosestPresetIndex();

  return (
    <div>
      <label htmlFor="range" className="text-xs">
        Range
      </label>
      <div className="flex h-[38px] rounded-md shadow-sm">
        <button
          type="button"
          disabled={currentPresetIndex === 0}
          className="rounded-l-md border border-r-0 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-700"
          onClick={() => {
            const previousRangeIndex = currentPresetIndex - 1;
            const previousRange = presetRanges[previousRangeIndex];
            setValidRange(previousRange);
          }}
        >
          <Icon icon="heroicons:minus-20-solid" />
        </button>
        <input
          type="text"
          name="range"
          id="range"
          className="flex w-full flex-grow items-stretch border text-center text-gray-900 placeholder:text-gray-400 focus-within:z-10 focus:ring-1 focus:ring-inset focus:ring-indigo-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 sm:text-sm sm:leading-6"
          placeholder="6h"
          value={rangeInputString}
          onClick={() => {
            toggleRangePickerPanel();
          }}
          onChange={e => {
            setRangeInputString(e.target.value);
          }}
          onBlur={() => {
            const parsedInput = parseInput(rangeInputString);

            // if parsed input is not valid, set input to the previous valid value
            if (parsedInput === null) {
              setRangeInputString(`${validRange.value}${unitShort[validRange.unit]}`);
              return;
            }

            // if parsed input is valid, set valid range state
            const {value, unit} = parsedInput;
            setValidRange({value, unit});
          }}
          onKeyDown={e => {
            // if enter key is pressed, blur the input
            if (e.key === 'Enter') {
              e.currentTarget.blur();
            }
          }}
        />
        <button
          type="button"
          disabled={currentPresetIndex >= presetRanges.length - 1}
          className="rounded-r-md border border-l-0 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-700"
          onClick={() => {
            const previousRangeIndex = currentPresetIndex + 1;
            const previousRange = presetRanges[previousRangeIndex];
            setValidRange(previousRange);
          }}
        >
          <Icon icon="heroicons:plus-20-solid" />
        </button>
      </div>
      <button
        type="button"
        className="flex w-fit"
        onClick={() => {
          onChange(
            new AbsoluteDate(getHistoricalDate(validRange)),
            new AbsoluteDate(getHistoricalDate(NOW))
          );
        }}
      >
        <p className="my-1 ml-1 text-center text-xs text-gray-500 hover:text-indigo-600 dark:text-gray-400">
          Use absolute start time instead
        </p>
      </button>
    </div>
  );
};

export default RelativeDatePicker;
