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

import isChromatic from 'chromatic/isChromatic';

import {AbsoluteDate, DateTimeRange, getDateHoursAgo} from '../../DateTimeRangePicker/utils';
import DateTimeRangePicker from './StateWrappedComponent';

if (isChromatic()) {
  // Freeze time to a constant value to make snapshots predictable
  window.Date = new Proxy(window.Date, {
    construct(target, args) {
      if (args.length === 0) {
        return new Date(1708514847145);
      }
      /* @ts-expect-error */ // eslint-disable-next-line new-cap
      return new target(...args);
    },
  });
}

export default {
  title: 'Components/DateTimeRangePicker ',
  component: DateTimeRangePicker,
};

export const Default = {args: {}};

export const Last1HWithAlias = {
  args: {
    range: new DateTimeRange(new AbsoluteDate(getDateHoursAgo(1)), new AbsoluteDate('now')),
  },
};
