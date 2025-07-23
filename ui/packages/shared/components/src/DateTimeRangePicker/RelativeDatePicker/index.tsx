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

import {Icon} from '@iconify/react';

import {Button} from '../../Button';
import {
  AbsoluteDate,
  DateTimeRange,
  RelativeDate,
  UNITS,
  UNIT_TYPE,
  createNow,
  formatRange,
  getHistoricalDate,
  getRelativeTimeRangeBetweenDates,
  parseInput,
  presetRanges,
  unitShort,
} from '../utils';

interface RelativeDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: RelativeDate | AbsoluteDate, to: RelativeDate | AbsoluteDate) => void;
  toggleRangePickerPanel: () => void;
}

export const RelativeDatePickerForPanel = ({
  onChange = () => null,
  range,
  hidePopoverMenu,
}: {
  onChange: (from: RelativeDate | AbsoluteDate, to: RelativeDate | AbsoluteDate) => void;
  range: DateTimeRange;
  hidePopoverMenu: () => void;
}): JSX.Element => {
  const dateFromInRelative = range.from as RelativeDate;
  const dateToInRelative = range.to as RelativeDate;

  const [from] = useState<AbsoluteDate>(
    range.from.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({
            unit: dateFromInRelative.unit,
            value: dateFromInRelative.value,
          })
        )
      : (range.from as AbsoluteDate)
  );
  const [to] = useState<AbsoluteDate>(
    range.to.isRelative()
      ? new AbsoluteDate(
          getHistoricalDate({
            unit: dateToInRelative.unit,
            value: dateToInRelative.value,
          })
        )
      : (range.to as AbsoluteDate)
  );

  const {unit, value} = useMemo(
    () => getRelativeTimeRangeBetweenDates(to.getTime().getTime() - from.getTime().getTime()),
    [from, to]
  );

  // When the list of presets is shown in the popover panel, we use this effect here to ensure that the
  // absolute date range is converted to a relative date range and we then use the `onChange` prop to
  // update the range in the `RelativeDatePicker` component below.
  useEffect(() => {
    onChange(new RelativeDate(unit, value), createNow());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unit, value]);

  const presetRanges = [
    {value: 15, unit: UNITS.MINUTE, text: 'Last 15 minutes'},
    {value: 1, unit: UNITS.HOUR, text: 'Last 1 hour'},
    {value: 3, unit: UNITS.HOUR, text: 'Last 3 hours'},
    {value: 6, unit: UNITS.HOUR, text: 'Last 6 hours'},
    {value: 12, unit: UNITS.HOUR, text: 'Last 12 hours'},
    {value: 1, unit: UNITS.DAY, text: 'Last 1 day'},
  ];

  return (
    <div className="flex flex-col gap-3 items-center text-sm p-4">
      {presetRanges.map(({value, unit, text}) => (
        <Button
          key={`${value}-${unit}`}
          onClick={() => {
            onChange(new RelativeDate(unit, value), createNow());
            hidePopoverMenu();
          }}
          className="min-w-[142px]"
          variant={
            value === dateFromInRelative.value && unit === dateFromInRelative.unit
              ? 'primary'
              : 'neutral'
          }
        >
          {text}
        </Button>
      ))}
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
    unit: UNIT_TYPE;
  }>({
    value: date.value,
    unit: date.unit,
  });

  useEffect(() => {
    setRangeInputString(formatRange(date.value, date.unit));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range]);

  useEffect(() => {
    const formattedRange = formatRange(validRange.value, validRange.unit);
    setRangeInputString(formattedRange);
    onChange(new RelativeDate(validRange.unit, validRange.value), createNow());
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
      <div className="flex h-[38px] w-[300px] rounded-md shadow-sm">
        <button
          type="button"
          disabled={currentPresetIndex === 0}
          className="rounded-l-md border border-r-0 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400 border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-700"
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
          className="flex w-full flex-grow items-stretch border text-center text-gray-900 placeholder:text-gray-400 focus-within:z-10 focus:border-indigo-500 focus:ring-1 focus:ring-inset focus:ring-indigo-500 border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 sm:text-sm sm:leading-6 outline-none"
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
              setRangeInputString(formatRange(validRange.value, validRange.unit));
            } else {
              setValidRange(parsedInput);
            }
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
          className="rounded-r-md border border-l-0 bg-gray-100 p-3 text-sm font-semibold text-gray-900 hover:bg-gray-200 disabled:bg-white disabled:text-gray-400 border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-200 dark:hover:bg-gray-700"
          onClick={() => {
            const previousRangeIndex = currentPresetIndex + 1;
            const previousRange = presetRanges[previousRangeIndex];
            setValidRange(previousRange);
          }}
        >
          <Icon icon="heroicons:plus-20-solid" />
        </button>
      </div>
    </div>
  );
};

export default RelativeDatePicker;
