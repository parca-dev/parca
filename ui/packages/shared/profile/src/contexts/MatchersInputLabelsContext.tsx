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

import React, {createContext, useContext, useMemo} from 'react';

import {QueryServiceClient} from '@parca/client';

import {useLabelNames, useLabelValues} from '../MatchersInput';
import {useExtractedLabelNames, useLabelNameMappings, type LabelNameMapping} from './utils';

interface LabelsContextType {
  labelNames: string[];
  labelValues: string[];
  labelNameMappings: LabelNameMapping[];
  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;
  currentLabelName: string | null;
  setCurrentLabelName: (name: string | null) => void;
  shouldHandlePrefixes: boolean;
  fetchLabelValues?: (labelName: string) => Promise<string[]>;
  refetchLabelValues: () => Promise<void>;
  refetchLabelNames: () => Promise<void>;
}

const LabelsContext = createContext<LabelsContextType | null>(null);

interface LabelsProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  start?: number;
  end?: number;
  shouldHandlePrefixes?: boolean;
  externalLabelNames?: string[];
  externalLabelNamesLoading?: boolean;
  externalFetchLabelValues?: (labelName: string) => Promise<string[]>;
  externalRefetchLabelNames?: () => Promise<void>;
  externalRefetchLabelValues?: (labelName?: string) => Promise<void>;
}

export function LabelsProvider({
  children,
  queryClient,
  profileType,
  start,
  end,
  shouldHandlePrefixes = true,
  externalLabelNames,
  externalLabelNamesLoading = false,
  externalFetchLabelValues,
  externalRefetchLabelNames,
  externalRefetchLabelValues,
}: LabelsProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = React.useState<string | null>(null);

  const {
    result: labelNamesResponse,
    loading: isLabelNamesLoading,
    refetch: refetchLabelNamesInternal,
  } = useLabelNames(queryClient, profileType, start, end);

  const internalLabelNames = useExtractedLabelNames(
    labelNamesResponse.response,
    labelNamesResponse.error
  );

  const labelNamesFromAPI = useMemo(() => {
    const combined = [...internalLabelNames];
    if (externalLabelNames != null) {
      combined.push(...externalLabelNames);
    }
    return Array.from(new Set(combined)); // dedupe
  }, [internalLabelNames, externalLabelNames]);

  const mergedLoading = isLabelNamesLoading || externalLabelNamesLoading;

  const {
    result: labelValuesOriginal,
    loading: isLabelValuesLoading,
    refetch: refetchLabelValuesInternal,
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType, start, end);

  const labelNameMappings = useLabelNameMappings(labelNamesFromAPI);

  const labelNames = useMemo(() => {
    return labelNameMappings.map(m => m.displayName);
  }, [labelNameMappings]);

  const labelValues = useMemo(() => {
    return labelValuesOriginal.response;
  }, [labelValuesOriginal]);

  const refetchLabelNames = React.useCallback(async () => {
    await Promise.all([
      refetchLabelNamesInternal(),
      externalRefetchLabelNames?.() ?? Promise.resolve(),
    ]);
  }, [refetchLabelNamesInternal, externalRefetchLabelNames]);

  const refetchLabelValues = React.useCallback(
    async (labelName?: string) => {
      await Promise.all([
        refetchLabelValuesInternal(),
        externalRefetchLabelValues?.(labelName) ?? Promise.resolve(),
      ]);
    },
    [refetchLabelValuesInternal, externalRefetchLabelValues]
  );

  const value = useMemo(
    () => ({
      labelNames,
      labelValues,
      labelNameMappings,
      isLabelNamesLoading: mergedLoading,
      isLabelValuesLoading,
      currentLabelName,
      setCurrentLabelName,
      shouldHandlePrefixes,
      fetchLabelValues: externalFetchLabelValues,
      refetchLabelValues,
      refetchLabelNames,
    }),
    [
      labelNames,
      labelValues,
      labelNameMappings,
      mergedLoading,
      isLabelValuesLoading,
      currentLabelName,
      shouldHandlePrefixes,
      externalFetchLabelValues,
      refetchLabelValues,
      refetchLabelNames,
    ]
  );

  return <LabelsContext.Provider value={value}>{children}</LabelsContext.Provider>;
}

export function useLabels(): LabelsContextType {
  const context = useContext(LabelsContext);
  if (context === null) {
    throw new Error('useLabels must be used within a LabelsProvider');
  }
  return context;
}
