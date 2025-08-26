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
import {beforeEach, describe, expect, it, vi} from 'vitest';

import {AbsoluteDate, DateTimeRange} from '../utils';
import DateTimeRangePickerText from './index';

describe('DateTimeRangePickerText', () => {
  const mockOnClick = vi.fn();
  const mockOnRangeSelection = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('timezone handling with absolute dates', () => {
    it('should display times in UTC format when no timezone is provided (advanced mode)', () => {
      // Create absolute date range for testing: 2023-12-01 10:00:00 to 2023-12-01 15:30:00 UTC
      const fromDate = new Date('2023-12-01T10:00:00Z'); // UTC time
      const toDate = new Date('2023-12-01T15:30:00Z'); // UTC time

      const range = new DateTimeRange(new AbsoluteDate(fromDate), new AbsoluteDate(toDate));

      render(
        <DateTimeRangePickerText
          range={range}
          onClick={mockOnClick}
          isActive={false}
          onRangeSelection={mockOnRangeSelection}
          timezone={undefined} // Advanced mode - should show UTC
        />
      );

      // Find the button that displays the date range
      const button = screen.getByTestId('date-time-range-picker-button');

      // In UTC mode (no timezone), times should be displayed in UTC format
      // Expected format: "YYYY-MM-DD HH:mm:ss → HH:mm:ss"
      expect(button).toBeInTheDocument();

      // The text content should contain UTC formatted times
      const buttonText = button.textContent || '';

      // Should contain the full from date in UTC format
      expect(buttonText).toMatch(/2023-12-01 10:00:00/);

      // Should contain the to time (same date, so just time portion)
      expect(buttonText).toMatch(/15:30:00/);

      // Should contain the arrow separator
      expect(buttonText).toMatch(/→/);

      // Verify the complete expected format (removing the dropdown arrow character)
      const cleanButtonText = buttonText.replace(/[▼▲]/g, '').trim();
      expect(cleanButtonText).toBe('2023-12-01 10:00:00 → 15:30:00');
    });

    it('should display times in specified timezone when timezone is provided', () => {
      // Create test dates in UTC
      const fromDate = new Date('2023-12-01T15:30:00Z'); // 15:30 UTC
      const toDate = new Date('2023-12-01T20:00:00Z'); // 20:00 UTC

      const range = new DateTimeRange(new AbsoluteDate(fromDate), new AbsoluteDate(toDate));

      render(
        <DateTimeRangePickerText
          range={range}
          onClick={mockOnClick}
          isActive={false}
          onRangeSelection={mockOnRangeSelection}
          timezone="America/New_York" // EST/EDT timezone
        />
      );

      const button = screen.getByTestId('date-time-range-picker-button');
      const buttonText = button.textContent || '';

      // In America/New_York timezone:
      // 15:30 UTC = 10:30 EST (UTC-5) or 11:30 EDT (UTC-4)
      // 20:00 UTC = 15:00 EST (UTC-5) or 16:00 EDT (UTC-4)

      // The exact time will depend on whether the date falls in EST or EDT
      // But we can verify it's different from UTC and contains timezone-adjusted times
      expect(buttonText).not.toMatch(/15:30:00/); // Should not show UTC time
      expect(buttonText).not.toMatch(/20:00:00/); // Should not show UTC time

      // Should contain timezone-adjusted times (either EST or EDT)
      expect(buttonText).toMatch(/10:30:00|11:30:00/); // From time in EST/EDT
      expect(buttonText).toMatch(/15:00:00|16:00:00/); // To time in EST/EDT

      // Should still contain the arrow separator
      expect(buttonText).toMatch(/→/);
    });

    it('should display different times for different timezones', () => {
      // Test with the same UTC time but different timezones
      const fromDate = new Date('2023-06-15T12:00:00Z'); // Noon UTC (summer date for DST)
      const toDate = new Date('2023-06-15T18:00:00Z'); // 6 PM UTC

      const range = new DateTimeRange(new AbsoluteDate(fromDate), new AbsoluteDate(toDate));

      // Test with Pacific timezone
      const {rerender} = render(
        <DateTimeRangePickerText
          range={range}
          onClick={mockOnClick}
          isActive={false}
          onRangeSelection={mockOnRangeSelection}
          timezone="America/Los_Angeles" // PDT (UTC-7 in summer)
        />
      );

      const button = screen.getByTestId('date-time-range-picker-button');
      const pacificTime = button.textContent || '';

      // 12:00 UTC = 05:00 PDT, 18:00 UTC = 11:00 PDT
      expect(pacificTime).toMatch(/05:00:00/);
      expect(pacificTime).toMatch(/11:00:00/);

      // Test with Tokyo timezone
      rerender(
        <DateTimeRangePickerText
          range={range}
          onClick={mockOnClick}
          isActive={false}
          onRangeSelection={mockOnRangeSelection}
          timezone="Asia/Tokyo" // JST (UTC+9)
        />
      );

      const tokyoTime = button.textContent || '';

      // 12:00 UTC = 21:00 JST, 18:00 UTC = 03:00 JST (next day)
      const cleanTokyoTime = tokyoTime.replace(/[▼▲]/g, '').trim();
      expect(cleanTokyoTime).toBe('2023-06-15 21:00:00 → 2023-06-16 03:00:00');

      // Verify they're different
      expect(pacificTime).not.toBe(tokyoTime);
    });

    it('should handle edge case of date crossing with timezone conversion', () => {
      // Test UTC time that crosses date boundary when converted to timezone
      const fromDate = new Date('2023-12-01T02:00:00Z'); // 2 AM UTC
      const toDate = new Date('2023-12-01T04:00:00Z'); // 4 AM UTC

      const range = new DateTimeRange(new AbsoluteDate(fromDate), new AbsoluteDate(toDate));

      render(
        <DateTimeRangePickerText
          range={range}
          onClick={mockOnClick}
          isActive={false}
          onRangeSelection={mockOnRangeSelection}
          timezone="America/Los_Angeles" // PST (UTC-8), should be previous day
        />
      );

      const button = screen.getByTestId('date-time-range-picker-button');
      const buttonText = button.textContent || '';

      // 02:00 UTC on 2023-12-01 = 18:00 PST on 2023-11-30
      // 04:00 UTC on 2023-12-01 = 20:00 PST on 2023-11-30
      const cleanButtonText = buttonText.replace(/[▼▲]/g, '').trim();
      expect(cleanButtonText).toBe('2023-11-30 18:00:00 → 20:00:00');
    });
  });
});
