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

import DateTimeRangePicker from '../index';
import {AbsoluteDate, DateTimeRange, DateUnion, RelativeDate} from '../utils';

const toString = (date: DateUnion): string => {
  return date.isRelative()
    ? date.value.toString() + ' ' + (date as RelativeDate).unit
    : (date as AbsoluteDate).getTime().toISOString();
};

const StateWrappedComponent = (props: {key: string; value: any}): JSX.Element => {
  const [range, setRange] = useState(new DateTimeRange());
  return (
    <div className="flex flex-col gap-4 w-fit">
      <DateTimeRangePicker range={range} onRangeSelection={setRange} {...props} />
      <div className="flex flex-col gap-1 border-t-2 text-sm pt-1 mt-4">
        Evaluated values:
        <span>from: {toString(range.from)}</span>
        <span>to: {toString(range.to)}</span>
      </div>
    </div>
  );
};

export default StateWrappedComponent;
