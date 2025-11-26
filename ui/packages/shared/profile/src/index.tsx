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

import type {ParamPreferences} from '@parca/components';

import MetricsGraph, {type ContextMenuItemOrSubmenu, type Series} from './MetricsGraph';
import ProfileExplorer from './ProfileExplorer';
import ProfileTypeSelector from './ProfileTypeSelector';
import {SelectWithRefresh} from './SelectWithRefresh';
import CustomSelect from './SimpleMatchers/Select';
import {
  LabelsQueryProvider,
  useLabelsQueryProvider,
  type LabelsQueryProviderContextType,
} from './contexts/LabelsQueryProvider';
import {UnifiedLabelsProvider, useUnifiedLabels} from './contexts/UnifiedLabelsContext';
import {useLabelNames} from './hooks/useLabels';
import {useQueryState} from './hooks/useQueryState';

export {useMetricsGraphDimensions} from './MetricsGraph/useMetricsGraphDimensions';

export * from './ProfileFlameGraph';
export * from './ProfileSource';
export {
  convertToProtoFilters,
  convertFromProtoFilters,
} from './ProfileView/components/ProfileFilters/useProfileFilters';
export * from './ProfileView';
export * from './ProfileViewWithData';
export * from './utils';
export * from './ProfileTypeSelector';
export * from './SourceView';
export * from './ProfileMetricsGraph';
export * from './useSumBy';
export {QueryControls} from './QueryControls';

export {default as ProfileFilters} from './ProfileView/components/ProfileFilters';
export {useProfileFiltersUrlState} from './ProfileView/components/ProfileFilters/useProfileFiltersUrlState';

export const DEFAULT_PROFILE_EXPLORER_PARAM_VALUES: ParamPreferences = {
  dashboard_items: {
    defaultValue: 'flamegraph',
    splitOnCommas: true, // This param should split on commas for array values
  },
};

export {
  ProfileExplorer,
  ProfileTypeSelector,
  CustomSelect,
  SelectWithRefresh,
  useLabelNames,
  MetricsGraph,
  type ContextMenuItemOrSubmenu,
  type Series,
  LabelsQueryProvider,
  useLabelsQueryProvider,
  UnifiedLabelsProvider,
  useUnifiedLabels,
  useQueryState,
  type LabelsQueryProviderContextType,
};
