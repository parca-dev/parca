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

/**
 * Centralized test IDs for Parca e2e testing
 * This ensures consistency between components and tests
 */

export const TEST_IDS = {
  // QueryControls Main Container
  QUERY_CONTROLS_CONTAINER: 'query-controls-container',

  // Profile Type Selector
  PROFILE_TYPE_SELECTOR: 'profile-type-selector',
  PROFILE_TYPE_LABEL: 'profile-type-label',

  // Query Browser Mode Switch
  QUERY_MODE_SWITCH: 'query-mode-switch',
  QUERY_MODE_LABEL: 'query-mode-label',
  ADVANCED_MODE_SWITCH: 'advanced-mode-switch',

  // Query Input Section
  QUERY_BROWSER_CONTAINER: 'query-browser-container',
  QUERY_LABEL: 'query-label',

  // MatchersInput (Advanced Mode)
  MATCHERS_INPUT_CONTAINER: 'matchers-input-container',
  MATCHERS_TEXTAREA: 'matchers-textarea',

  // SimpleMatchers (Simple Mode)
  SIMPLE_MATCHERS_CONTAINER: 'simple-matchers-container',
  SIMPLE_MATCHER_ROW: 'simple-matcher-row',
  LABEL_NAME_SELECT: 'label-name-select',
  OPERATOR_SELECT: 'operator-select',
  LABEL_VALUE_SELECT: 'label-value-select',
  REMOVE_MATCHER_BUTTON: 'remove-matcher-button',
  ADD_MATCHER_BUTTON: 'add-matcher-button',
  SHOW_MORE_BUTTON: 'show-more-button',
  SHOW_LESS_BUTTON: 'show-less-button',

  // ViewMatchers
  VIEW_MATCHERS_CONTAINER: 'view-matchers-container',

  // Sum By Selector
  SUM_BY_CONTAINER: 'sum-by-container',
  SUM_BY_LABEL: 'sum-by-label',
  SUM_BY_SELECT: 'sum-by-select',

  // Date Time Range Picker
  DATE_TIME_RANGE_PICKER: 'date-time-range-picker',
  DATE_TIME_RANGE_PICKER_CONTAINER: 'date-time-range-picker-container',
  DATE_TIME_RANGE_PICKER_TEXT: 'date-time-range-picker-text',
  DATE_TIME_RANGE_PICKER_BUTTON: 'date-time-range-picker-button',
  DATE_TIME_RANGE_PICKER_PANEL: 'date-time-range-picker-panel',
  DATE_TIME_RANGE_LABEL: 'date-time-range-label',

  // Date Time Range Picker - Tabs
  RELATIVE_TAB: 'relative-tab',
  ABSOLUTE_TAB: 'absolute-tab',

  // Relative Date Picker
  RELATIVE_DATE_PICKER: 'relative-date-picker',
  RELATIVE_DATE_SELECT: 'relative-date-select',
  RELATIVE_TIME_INPUT: 'relative-time-input',
  RELATIVE_UNIT_SELECT: 'relative-unit-select',

  // Absolute Date Picker
  ABSOLUTE_DATE_PICKER: 'absolute-date-picker',
  FROM_DATE_INPUT: 'from-date-input',
  TO_DATE_INPUT: 'to-date-input',

  // Search Button
  SEARCH_BUTTON: 'search-button',
  SEARCH_BUTTON_LABEL: 'search-button-label',

  // Flamegraph Container
  FLAMEGRAPH_CONTAINER: 'flamegraph-container',
  FLAMEGRAPH_RESET_BUTTON: 'flamegraph-reset-button',

  // Metrics Graph
  METRICS_GRAPH: 'metrics-graph',
  METRICS_GRAPH_TOOLTIP: 'metrics-graph-tooltip',
  TOOLTIP_LABEL: 'tooltip-label',
  TOOLTIP_LABEL_KEY: 'tooltip-label-key',
  TOOLTIP_LABEL_VALUE: 'tooltip-label-value',
  METRICS_GRAPH_CONTEXT_MENU: 'metrics-graph-context-menu',

  // Profile Filters
  PROFILE_FILTERS_CONTAINER: 'profile-filters-container',
  FILTER_TYPE_SELECT: 'filter-type-select',
  FILTER_TYPE_SELECT_FLYOUT: 'filter-type-select-flyout',
  FILTER_FIELD_SELECT: 'filter-field-select',
  FILTER_FIELD_SELECT_FLYOUT: 'filter-field-select-flyout',
  FILTER_MATCH_TYPE_SELECT: 'filter-match-type-select',
  FILTER_MATCH_TYPE_SELECT_FLYOUT: 'filter-match-type-select-flyout',
  FILTER_VALUE_INPUT: 'filter-value-input',
  FILTER_REMOVE_BUTTON: 'filter-remove-button',
  ADD_FILTER_BUTTON: 'add-filter-button',
  APPLY_FILTERS_BUTTON: 'apply-filters-button',

  // Group By Controls
  GROUP_BY_CONTAINER: 'group-by-container',
  GROUP_BY_LABEL: 'group-by-label',
  GROUP_BY_SELECT_FLYOUT: 'group-by-select-flyout',

  // Compare Mode
  COMPARE_CONTAINER: 'compare-container',
  COMPARE_SIDE_A: 'compare-side-a',
  COMPARE_SIDE_B: 'compare-side-b',
  COMPARE_PROFILE_VIEW: 'compare-profile-view',

  // Diff Legend
  DIFF_LEGEND: 'diff-legend',

  // Common Interactive Elements
  SELECT_DROPDOWN: 'select-dropdown',
  SELECT_OPTION: 'select-option',
  BUTTON: 'button',
  SWITCH: 'switch',
  TEXTAREA: 'textarea',
  INPUT: 'input',
  LABEL: 'label',
} as const;

// Type-safe helper function to get test IDs
export const getTestId = (key: keyof typeof TEST_IDS): string => {
  return TEST_IDS[key];
};

// Helper function to create data-testid attribute object
export const testId = (id: (typeof TEST_IDS)[keyof typeof TEST_IDS]): Record<string, string> => ({
  'data-testid': id,
});
