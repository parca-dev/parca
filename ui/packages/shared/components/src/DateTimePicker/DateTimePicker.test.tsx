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

import {AbsoluteDate} from '../DateTimeRangePicker/utils';
import ParcaContext from '../ParcaContext';
import {DateTimePicker} from './index';

describe('DateTimePicker', () => {
  it('should display times in UTC when in advanced mode (no timezone)', () => {
    // Create a specific date for testing: 2023-12-01 15:30:00
    const testDate = new Date('2023-12-01T15:30:00Z'); // UTC time
    const absoluteDate = new AbsoluteDate(testDate);

    const mockOnChange = vi.fn();

    // Render with no timezone context (advanced mode)
    render(
      <ParcaContext.Provider
        value={{
          timezone: undefined, // This triggers advanced mode
          dark: false,
        }}
      >
        <DateTimePicker selected={absoluteDate} onChange={mockOnChange} />
      </ParcaContext.Provider>
    );

    // In advanced mode, the component should display UTC time
    // The input field should show the UTC formatted time
    const input = screen.getByRole('textbox');

    // The time should be displayed in UTC format: YYYY-MM-DD HH:mm:ss
    expect(input).toHaveValue('2023-12-01 15:30:00');

    // Verify that the displayed time matches the UTC time exactly
    // (not adjusted for local timezone)
    const expectedUtcString = absoluteDate.getUIString(); // No timezone means UTC
    expect(input.value).toBe(expectedUtcString);
  });

  it('should display times in specified timezone when timezone is provided', () => {
    // Create a test date
    const testDate = new Date('2023-12-01T15:30:00Z'); // UTC time
    const absoluteDate = new AbsoluteDate(testDate);

    const mockOnChange = vi.fn();

    // Render with a specific timezone (e.g., America/New_York)
    render(
      <ParcaContext.Provider
        value={{
          timezone: 'America/New_York',
          dark: false,
        }}
      >
        <DateTimePicker selected={absoluteDate} onChange={mockOnChange} />
      </ParcaContext.Provider>
    );

    const input = screen.getByRole('textbox');

    // The time should be displayed in the specified timezone
    // America/New_York is UTC-5 (or UTC-4 with DST), so 15:30 UTC should be adjusted
    const expectedTimezoneString = absoluteDate.getUIString('America/New_York');
    expect(input.value).toBe(expectedTimezoneString);

    // Verify it's different from UTC display
    const utcString = absoluteDate.getUIString();
    expect(input.value).not.toBe(utcString);
  });

  it('should maintain UTC time consistency when timezone is undefined', () => {
    // Test multiple dates to ensure consistent UTC behavior
    const testDates = [
      new Date('2023-01-01T00:00:00Z'), // Start of year
      new Date('2023-06-15T12:00:00Z'), // Mid-year (different DST)
      new Date('2023-12-31T23:59:59Z'), // End of year
    ];

    const mockOnChange = vi.fn();

    testDates.forEach((testDate, index) => {
      const absoluteDate = new AbsoluteDate(testDate);

      const {unmount} = render(
        <ParcaContext.Provider
          value={{
            timezone: undefined, // Advanced mode - should always show UTC
            dark: false,
          }}
        >
          <DateTimePicker selected={absoluteDate} onChange={mockOnChange} />
        </ParcaContext.Provider>
      );

      const input = screen.getByRole('textbox');

      // Each date should display in UTC format
      const expectedUtcString = absoluteDate.getUIString();
      expect(input.value).toBe(expectedUtcString);

      // Verify the format matches the expected UTC pattern (YYYY-MM-DD HH:mm:ss)
      expect(input.value).toMatch(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/);

      // Clean up after each iteration
      unmount();
    });
  });

  it('should handle "now" value correctly in advanced mode', () => {
    const absoluteDate = new AbsoluteDate('now');
    const mockOnChange = vi.fn();

    render(
      <ParcaContext.Provider
        value={{
          timezone: undefined, // Advanced mode
          dark: false,
        }}
      >
        <DateTimePicker selected={absoluteDate} onChange={mockOnChange} />
      </ParcaContext.Provider>
    );

    const input = screen.getByRole('textbox');

    // When value is "now", it should display as "now" regardless of timezone mode
    expect(input.value).toBe('now');
  });
});
