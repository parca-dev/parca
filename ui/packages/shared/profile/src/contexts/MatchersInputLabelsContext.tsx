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

import {createContext, useCallback, useContext, useMemo, useState} from 'react';

import {QueryServiceClient} from '@parca/client';

import {ExternalLabelSource} from '../contexts/UnifiedLabelsContext';
import {useExternalLabelValues} from '../hooks/useExternalLabels';
import {useLabelNames, useLabelValues} from '../hooks/useLabels';
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
  refetchLabelValues: () => Promise<void>;
  refetchLabelNames: () => Promise<void>;

  externalLabelSource?: ExternalLabelSource;
}

const LabelsContext = createContext<LabelsContextType | null>(null);

interface LabelsProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  start?: number;
  end?: number;
  shouldHandlePrefixes?: boolean;

  externalLabelSource?: ExternalLabelSource;
}

export function LabelsProvider({
  children,
  queryClient,
  profileType,
  start,
  end,
  shouldHandlePrefixes = true,
  externalLabelSource,
}: LabelsProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);

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
    if (externalLabelSource?.labelNames != null) {
      combined.push(...externalLabelSource.labelNames);
    }
    return Array.from(new Set(combined));
  }, [internalLabelNames, externalLabelSource?.labelNames]);

  const mergedLoading = isLabelNamesLoading || externalLabelSource?.isLoading === true;

  const {
    result: labelValuesOriginal,
    loading: isLabelValuesLoading,
    refetch: refetchLabelValuesInternal,
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType, start, end);

  const {
    data: externalLabelValues,
    loading: isExternalLabelValuesLoading,
    refetch: refetchExternalLabelValues,
  } = useExternalLabelValues(currentLabelName ?? '', externalLabelSource);

  const mergedLabelValuesLoading =
    isLabelValuesLoading || (externalLabelSource != null && isExternalLabelValuesLoading);

  const mergedLabelValues = useMemo(() => {
    if (externalLabelValues != null) {
      return [...labelValuesOriginal.response, ...externalLabelValues];
    }
    return labelValuesOriginal.response;
  }, [labelValuesOriginal, externalLabelValues]);

  const labelNameMappings = useLabelNameMappings(labelNamesFromAPI);

  const labelNames = useMemo(() => {
    return labelNameMappings.map(m => m.displayName);
  }, [labelNameMappings]);

  const refetchLabelNames = useCallback(async () => {
    await Promise.all([
      refetchLabelNamesInternal(),
      externalLabelSource?.refetchLabelNames ?? Promise.resolve(),
    ]);
  }, [refetchLabelNamesInternal, externalLabelSource?.refetchLabelNames]);

  const refetchLabelValues = useCallback(async () => {
    await Promise.all([
      refetchLabelValuesInternal(),
      refetchExternalLabelValues ?? Promise.resolve(),
    ]);
  }, [refetchLabelValuesInternal, refetchExternalLabelValues]);

  const value = useMemo(
    () => ({
      labelNames,
      labelValues: mergedLabelValues,
      labelNameMappings,
      isLabelNamesLoading: mergedLoading,
      isLabelValuesLoading: mergedLabelValuesLoading,
      currentLabelName,
      setCurrentLabelName,
      shouldHandlePrefixes,
      refetchLabelValues,
      refetchLabelNames,
    }),
    [
      labelNames,
      mergedLabelValues,
      labelNameMappings,
      mergedLoading,
      currentLabelName,
      shouldHandlePrefixes,
      refetchLabelValues,
      refetchLabelNames,
      mergedLabelValuesLoading,
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
