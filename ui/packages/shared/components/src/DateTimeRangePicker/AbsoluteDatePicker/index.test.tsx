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

import {render, screen} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';

import {AbsoluteDate, DateTimeRange} from '../utils';
import AbsoluteDatePicker from './index';

describe('AbsoluteDatePicker', () => {
  it('resyncs when an existing DateTimeRange instance is mutated', () => {
    const range = new DateTimeRange(
      new AbsoluteDate(new Date('2023-12-01T10:00:00Z')),
      new AbsoluteDate(new Date('2023-12-01T15:30:00Z'))
    );

    const {rerender} = render(<AbsoluteDatePicker range={range} onChange={vi.fn()} />);

    const [startInput, endInput] = screen.getAllByRole('textbox') as HTMLInputElement[];
    expect(startInput.value).toBe('2023-12-01 10:00:00');
    expect(endInput.value).toBe('2023-12-01 15:30:00');

    range.from = new AbsoluteDate(new Date('2023-12-02T08:15:00Z'));
    range.to = new AbsoluteDate(new Date('2023-12-02T09:45:00Z'));

    rerender(<AbsoluteDatePicker range={range} onChange={vi.fn()} />);

    expect(startInput.value).toBe('2023-12-02 08:15:00');
    expect(endInput.value).toBe('2023-12-02 09:45:00');
  });
});
