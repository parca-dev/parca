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

import {capitalize} from '@parca/utilities';

import {Button} from '../../../Button';
import Input from '../../../Input';
import Select, {contructItemsFromArray} from '../../../Select';
import {DateTimeRange, RelativeDate, UNITS, UNIT_TYPE} from '../../utils';

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
    <div className="w-[300px] p-4">
      <div className="pb-2">
        <div className="mb-4 hidden">
          <span className="text-xs uppercase text-gray-500">Quick Ranges</span>
        </div>
        <div className="grid grid-flow-col grid-rows-3 gap-2">
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
        <div className="my-4 mx-auto border-b-[1px] border-gray-200 text-center text-xs leading-[0px]">
          <span className="bg-gray-100 px-1 dark:bg-gray-800">OR</span>
        </div>
        <div className="my-4 flex items-center justify-center p-1">
          <span className="mr-4 text-xs uppercase">Last</span>
          <Input
            type="number"
            className="mr-2 w-16 text-sm"
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
            onSelection={key => setUnit(key)}
          />
        </div>
        <div className="mt-4 flex">
          <div className="mx-auto">
            <Button
              onClick={() => {
                onChange(new RelativeDate(unit, value), NOW);
              }}
            >
              Apply
            </Button>
          </div>
        </div>
      </div>
      <p className="m-4 text-center text-xs italic text-gray-500">
        Note: Setting a relative time means that on every search the time will be set to the time of
        the search.
      </p>
    </div>
  );
};

export default RelativeDatePicker;
