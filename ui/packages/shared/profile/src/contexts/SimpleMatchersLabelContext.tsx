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

interface LabelNameSection {
  type: string;
  values: SelectItem[];
}

interface LabelContextValue {
  labelNameOptions: LabelNameSection[];
  isLoading: boolean;
  error: Error | null;
  refetchLabelValues?: (labelName?: string) => Promise<void>;
  refetchLabelNames?: () => Promise<void>;
}

const LabelContext = createContext<LabelContextValue | null>(null);

export interface Labels {
  type: string;
  labelNames: string[];
  isLoading: boolean;
  error?: Error | null;
  fetchLabelValues?: (labelName: string) => Promise<string[]>;
}

interface LabelProviderProps {
  children: React.ReactNode;
  labels: Labels;
  labelNameFromMatchers: string[];
  refetchLabelValues?: (labelName?: string) => Promise<void>;
  refetchLabelNames?: () => Promise<void>;
}

export function LabelProvider({
  children,
  labels,
  labelNameFromMatchers,
  refetchLabelValues,
  refetchLabelNames,
}: LabelProviderProps): JSX.Element {
  const value = useMemo(() => {
    const isLoading = labels.isLoading;
    const error = labels.error ?? null;

    if (isLoading || error != null) {
      return {
        labelNameOptions: [],
        isLoading,
        error,
        fetchLabelValues: undefined,
        refetchLabelValues,
        refetchLabelNames,
      };
    }

    const allLabelNames = new Set(labels.labelNames);

    const nonMatchingLabels = labelNameFromMatchers.filter(label => !allLabelNames.has(label));

    const options: LabelNameSection[] = [];

    options.push({
      type: labels.type,
      values: transformLabelsForSelect(labels.labelNames),
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
      refetchLabelValues,
      refetchLabelNames,
    };
  }, [labels, labelNameFromMatchers, refetchLabelValues, refetchLabelNames]);

  return <LabelContext.Provider value={value}>{children}</LabelContext.Provider>;
}

export function useLabels(): LabelContextValue {
  const context = useContext(LabelContext);
  if (context === null) {
    throw new Error('useLabels must be used within a LabelProvider');
  }
  return context;
}
