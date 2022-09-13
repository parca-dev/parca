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
import {capitalize} from '@parca/functions';

import Input from '../../../Input';
import {DateTimeRange, RelativeDate, UNITS, UNIT_TYPE} from '../../utils';
import Button from '../../../Button';
import Select, {contructItemsFromArray} from '../../../Select';

const constructKeyAndLabels = (UNITS: UNIT_TYPE[]): Array<{key: string; label: string}> => {
  return UNITS.map(unit => ({
    key: unit,
    label: `${capitalize(unit)}s`,
  }));
};

interface RelativeDatePickerProps {
  range: DateTimeRange;
  onChange?: (from: RelativeDate, to: RelativeDate) => void;
}

const quickPresetRanges = [
  {
    title: 'Last 15 min',
    unit: UNITS.MINUTE,
    value: 15,
  },
  {
    title: 'Last 1 hour',
    unit: UNITS.HOUR,
    value: 1,
  },
  {
    title: 'Last 3 hours',
    unit: UNITS.HOUR,
    value: 3,
  },
  {
    title: 'Last 6 hours',
    unit: UNITS.HOUR,
    value: 6,
  },
  {
    title: 'Last 12 hours',
    unit: UNITS.HOUR,
    value: 12,
  },
  {
    title: 'Last 1 day',
    unit: UNITS.DAY,
    value: 1,
  },
];

const NOW = new RelativeDate(UNITS.MINUTE, 0);

const RelativeDatePicker = ({
  range,
  onChange = () => null,
}: RelativeDatePickerProps): JSX.Element => {
  const date = range.from as RelativeDate;
  const [unit, setUnit] = useState<UNIT_TYPE>(date.isRelative() ? date.unit : UNITS.MINUTE);
  const [value, setValue] = useState<number>(date.isRelative() ? date.value : 15);
  return (
    <div className="p-4 w-[300px]">
      <div className="pb-2">
        <div className="mb-4 hidden">
          <span className="uppercase text-xs text-gray-500">Quick Ranges</span>
        </div>
        <div className="grid grid-rows-3 grid-flow-col gap-2">
          {quickPresetRanges.map(({title, unit, value}) => (
            <Button
              onClick={() => {
                onChange(new RelativeDate(unit, value), NOW);
              }}
              color="link"
              key={title}
            >
              {title}
            </Button>
          ))}
        </div>
      </div>
      <div>
        <div className="my-4 border-b-[1px] border-gray-200 text-xs leading-[0px] mx-auto text-center">
          <span className="bg-gray-100 dark:bg-gray-800 px-1">OR</span>
        </div>
        <div className="flex items-center justify-center p-1 my-4">
          <span className="uppercase text-xs mr-4">Last</span>
          <Input
            type="number"
            className="w-16 mr-2 text-sm"
            value={value}
            min={0}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
              setValue(parseInt(e.target.value, 10))
            }
          />
          <Select
            className="w-32"
            items={contructItemsFromArray(constructKeyAndLabels(Object.values(UNITS)))}
            selectedKey={unit}
            onSelection={key => setUnit(key as UNIT_TYPE)}
          />
        </div>
        <div className="w-32 mx-auto pb-2">
          <Button
            onClick={() => {
              onChange(new RelativeDate(unit, value), NOW);
            }}
          >
            Apply
          </Button>
        </div>
      </div>
      <p className="text-gray-500 text-xs italic text-center mx-4">
        Note: Setting a relative time means that on every search the time will be set to the time of
        the search.
      </p>
    </div>
  );
};

export default RelativeDatePicker;
