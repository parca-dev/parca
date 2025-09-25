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

import {describe, expect, it} from 'vitest';

import {
  shiftTimeAcrossTimezones,
} from './time';

describe('shiftTimeAcrossTimezones', () => {
  it('shifts time correctly when converting between timezones shifting backwards', () => {
    const dateInIST = new Date('2024-06-10T00:00:00Z'); // 12:00 AM IST (UTC+5:30)
    const utcDate = shiftTimeAcrossTimezones(dateInIST, 'Asia/Kolkata', 'UTC');
    expect(utcDate.toISOString()).toBe('2024-06-10T05:30:00.000Z'); // 5:30 AM UTC
  });

  it('shifts time correctly when converting between timezones shifting forwards', () => {
    const dateInUTC = new Date('2024-06-10T05:30:00.000Z'); // 5:30 AM UTC
    const istDate = shiftTimeAcrossTimezones(dateInUTC, 'UTC', 'Asia/Kolkata');
    expect(istDate.toISOString()).toBe('2024-06-10T00:00:00.000Z'); // 12:00 AM IST (UTC+5:30)
  });

  it('returns the same date if both timezones are the same', () => {
    const dateInIST = new Date('2024-06-10T05:30:00Z');
    const sameDate = shiftTimeAcrossTimezones(dateInIST, 'Asia/Kolkata', 'Asia/Kolkata');
    expect(sameDate.toISOString()).toBe('2024-06-10T05:30:00.000Z');
  });
});
