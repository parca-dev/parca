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

import {createContext, useContext, useMemo} from 'react';

import {transformLabelsForSelect} from '../SimpleMatchers';
import type {SelectItem} from '../SimpleMatchers/Select';
import {aggregateLoadingState, findFirstError} from './utils';

interface LabelNameSection {
  type: string;
  values: SelectItem[];
}

interface LabelContextValue {
  labelNameOptions: LabelNameSection[];
  isLoading: boolean;
  error: Error | null;
  fetchLabelValues?: (labelName: string) => Promise<string[]>;
  refetchLabelValues?: (labelName?: string) => Promise<void>;
  refetchLabelNames?: () => Promise<void>;
}

const LabelContext = createContext<LabelContextValue | null>(null);

export interface LabelSource {
  type: string;
  labelNames: string[];
  isLoading: boolean;
  error?: Error | null;
  fetchLabelValues?: (labelName: string) => Promise<string[]>;
}

interface LabelProviderProps {
  children: React.ReactNode;
  labelSources: LabelSource[];
  labelNameFromMatchers: string[];
  refetchLabelValues?: (labelName?: string) => Promise<void>;
  refetchLabelNames?: () => Promise<void>;
}

export function LabelProvider({
  children,
  labelSources,
  labelNameFromMatchers,
  refetchLabelValues,
  refetchLabelNames,
}: LabelProviderProps): JSX.Element {
  const value = useMemo(() => {
    const isLoading = aggregateLoadingState(labelSources);
    const error = findFirstError(labelSources);

    const unifiedFetchLabelValues = (() => {
      const sourcesWithFetchers = labelSources.filter(s => s.fetchLabelValues != null);

      if (sourcesWithFetchers.length === 0) return undefined;

      return async (labelName: string): Promise<string[]> => {
        for (const source of sourcesWithFetchers) {
          const fetcher = source.fetchLabelValues;
          if (fetcher == null) continue;

          try {
            const values = await fetcher(labelName);
            if (values.length > 0) return values;
          } catch (error) {
            console.error('Error fetching from external label source:', error);
          }
        }
        return [];
      };
    })();

    if (isLoading || error != null) {
      return {
        labelNameOptions: [],
        isLoading,
        error,
        fetchLabelValues: unifiedFetchLabelValues,
        refetchLabelValues,
        refetchLabelNames,
      };
    }

    const allLabelNames = new Set<string>();
    labelSources.forEach(source => {
      source.labelNames.forEach(name => allLabelNames.add(name));
    });

    const nonMatchingLabels = labelNameFromMatchers.filter(label => !allLabelNames.has(label));

    const options: LabelNameSection[] = [];

    labelSources.forEach(source => {
      if (source.labelNames.length > 0) {
        options.push({
          type: source.type,
          values: transformLabelsForSelect(source.labelNames),
        });
      }
    });

    if (nonMatchingLabels.length > 0) {
      const uniqueNonMatchingLabels = Array.from(new Set(nonMatchingLabels));
      options.push({
        type: '',
        values: transformLabelsForSelect(uniqueNonMatchingLabels),
      });
    }

    return {
      labelNameOptions: options,
      isLoading: false,
      error: null,
      fetchLabelValues: unifiedFetchLabelValues,
      refetchLabelValues,
      refetchLabelNames,
    };
  }, [labelSources, labelNameFromMatchers, refetchLabelValues, refetchLabelNames]);

  return <LabelContext.Provider value={value}>{children}</LabelContext.Provider>;
}

export function useLabels(): LabelContextValue {
  const context = useContext(LabelContext);
  if (context === null) {
    throw new Error('useLabels must be used within a LabelProvider');
  }
  return context;
}
