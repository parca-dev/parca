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
 * UnifiedLabelsProvider - UI Presentation Layer
 *
 * This context provider transforms raw label data into a format optimized for
 * UI components (QueryControls, SimpleMatchers, etc.).
 *
 * Purpose:
 * - Transforms label arrays into structured formats for dropdowns and selectors
 * - Groups labels by type (e.g., 'cpu', 'gpu') for organized display
 * - Handles label name prefixes and mappings for user-friendly display
 * - Provides a unified interface regardless of data source(s)
 *
 * Architecture Pattern:
 * This is the final layer in a three-layer architecture:
 * 1. LabelsQueryProvider - Fetches data from API
 * 2. LabelsSource (in ProfileSelector) - Transforms/merges data
 * 3. UnifiedLabelsProvider (this file) - Presents data to UI components
 *
 * Consumer Hook: useUnifiedLabels()
 */

import {createContext, useContext} from 'react';

import {QueryServiceClient} from '@parca/client';
import {Query} from '@parca/parser';

import {transformLabelsForSelect} from '../SimpleMatchers';
import type {SelectItem} from '../SimpleMatchers/Select';
import {useLabelNameMappings, type LabelNameMapping} from './utils';

interface LabelNameSection {
  type: string;
  values: SelectItem[];
}

interface UnifiedLabelsContextType {
  labelNameMappingsForMatchersInput: LabelNameMapping[];
  labelNameMappingsForSimpleMatchers: LabelNameSection[];
  labelNames: string[];
  labelValues: string[];

  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;
  currentLabelName: string | null;
  setCurrentLabelName: (name: string | null) => void;
  shouldHandlePrefixes: boolean;
  refetchLabelValues: () => Promise<void>;
  refetchLabelNames: () => Promise<void>;
  labelNameFromMatchers: string[];

  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  start?: number;
  end?: number;
}

const UnifiedLabelsContext = createContext<UnifiedLabelsContextType | null>(null);

interface UnifiedLabelsProviderProps {
  children: React.ReactNode;

  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  start?: number;
  end?: number;

  currentLabelName: string | null;
  setCurrentLabelName: (name: string | null) => void;

  labelNames: string[];
  labelValues: string[];
  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;

  refetchLabelValues: () => Promise<void>;
  refetchLabelNames: () => Promise<void>;
}

export function UnifiedLabelsProvider({
  children,
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
  profileType,
  start,
  end,
  labelNames,
  isLabelNamesLoading,
  isLabelValuesLoading,
  refetchLabelValues,
  refetchLabelNames,
  currentLabelName,
  setCurrentLabelName,
  labelValues,
}: UnifiedLabelsProviderProps): JSX.Element {
  const labelNameFromMatchers: string[] = [];

  const labelNamesFromAPI = labelNames;

  const labelNameMappingsForMatchersInput = useLabelNameMappings(labelNamesFromAPI);

  const allLabelNames = new Set(labelNamesFromAPI);

  const nonMatchingLabels = labelNameFromMatchers.filter(label => !allLabelNames.has(label));

  const labelNameMappingsForSimpleMatchers: LabelNameSection[] = [];

  const labels = {
    type: 'cpu',
    labelNames: labelNamesFromAPI,
    isLoading: isLabelNamesLoading,
  };

  labelNameMappingsForSimpleMatchers.push({
    type: labels.type,
    values: transformLabelsForSelect(labels.labelNames),
  });

  if (nonMatchingLabels.length > 0) {
    const uniqueNonMatchingLabels = Array.from(new Set(nonMatchingLabels));
    labelNameMappingsForSimpleMatchers.push({
      type: '',
      values: transformLabelsForSelect(uniqueNonMatchingLabels),
    });
  }

  const value = {
    labelNames: labelNamesFromAPI,
    labelNameMappingsForMatchersInput,
    isLabelNamesLoading,
    isLabelValuesLoading,
    currentLabelName,
    labelValues,
    setCurrentLabelName,
    shouldHandlePrefixes: false,
    refetchLabelValues: async () => {
      await refetchLabelValues();
    },
    refetchLabelNames: async () => {
      await refetchLabelNames();
    },
    labelNameFromMatchers,
    labelNameMappingsForSimpleMatchers,
    queryClient,
    setMatchersString,
    runQuery,
    currentQuery,
    profileType,
    start,
    end,
  };

  return <UnifiedLabelsContext.Provider value={value}>{children}</UnifiedLabelsContext.Provider>;
}

export function useUnifiedLabels(): UnifiedLabelsContextType {
  const context = useContext(UnifiedLabelsContext);
  if (context === null) {
    throw new Error('useUnifiedLabels must be used within a UnifiedLabelsProvider');
  }
  return context;
}
