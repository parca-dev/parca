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

import ReactDatePicker from 'react-datepicker';

import {convertLocalToUTCDate, convertUTCToLocalDate} from '@parca/utilities';

interface Props {
  selected: Date;
  onChange: (date: Date | null) => void;
}

const DateTimePicker = ({selected, onChange}: Props): JSX.Element => (
  <ReactDatePicker
    selected={selected}
    onChange={onChange}
    showTimeInput
    dateFormat="yyyy-MM-dd HH:mm:ss"
    className="w-full rounded-md border border-gray-200  bg-gray-50 p-2 text-sm dark:border-gray-600 dark:bg-gray-900"
  />
);

export const UTCDateTimePicker = ({selected, onChange}: Props): JSX.Element => (
  <ReactDatePicker
    selected={convertUTCToLocalDate(selected)}
    onChange={date => onChange(date != null ? convertLocalToUTCDate(date) : null)}
    showTimeInput
    dateFormat="yyyy-MM-dd HH:mm:ss"
    className="w-full rounded-md border border-gray-200  bg-gray-50 p-2 text-sm dark:border-gray-600 dark:bg-gray-900"
  />
);

export default DateTimePicker;
