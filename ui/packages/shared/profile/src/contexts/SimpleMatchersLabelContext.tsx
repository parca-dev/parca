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

import {QueryServiceClient} from '@parca/client';

import {useLabelNames} from '../MatchersInput';
import {transformLabelsForSelect} from '../SimpleMatchers';
import type {SelectItem} from '../SimpleMatchers/Select';
import {useUtilizationLabels} from './UtilizationLabelsContext';

interface LabelContextValue {
  labelNameOptions: SelectItem[];
  isLoading: boolean;
  error: Error | null;
}

const LabelContext = createContext<LabelContextValue | null>(null);

interface LabelProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  labelNameFromMatchers: string[];
}

export function LabelProvider({
  children,
  queryClient,
  profileType,
  labelNameFromMatchers,
}: LabelProviderProps): JSX.Element {
  const utilizationLabels = useUtilizationLabels();
  const {loading, result} = useLabelNames(queryClient, profileType);

  const value = useMemo(() => {
    const baseLabels =
      result.error != null ? [] : result.response?.labelNames.filter(e => e !== '__name__') ?? [];

    const labelsToUse =
      utilizationLabels?.utilizationLabelNames?.length !== undefined
        ? utilizationLabels?.utilizationLabelNames ?? []
        : baseLabels;

    const uniqueLabels = Array.from(new Set([...labelsToUse, ...labelNameFromMatchers]));
    const shouldTrimPrefix = Boolean(utilizationLabels?.utilizationLabelNames);

    return {
      labelNameOptions: transformLabelsForSelect(uniqueLabels, shouldTrimPrefix),
      isLoading: loading,
      error: result.error ?? null,
    };
  }, [result, loading, utilizationLabels, labelNameFromMatchers]);

  return <LabelContext.Provider value={value}>{children}</LabelContext.Provider>;
}

export function useLabels(): LabelContextValue {
  const context = useContext(LabelContext);
  if (context === null) {
    throw new Error('useLabels must be used within a LabelProvider');
  }
  return context;
}
