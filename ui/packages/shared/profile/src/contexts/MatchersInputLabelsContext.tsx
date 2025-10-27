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

import {createContext, useContext, useMemo, useState} from 'react';

import {QueryServiceClient} from '@parca/client';

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
}

const LabelsContext = createContext<LabelsContextType | null>(null);

interface LabelsProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  start?: number;
  end?: number;
  shouldHandlePrefixes?: boolean;
}

export function LabelsProvider({
  children,
  queryClient,
  profileType,
  start,
  end,
  shouldHandlePrefixes = true,
}: LabelsProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);

  const {
    result: labelNamesResponse,
    loading: isLabelNamesLoading,
    refetch: labelNamesRefetch,
  } = useLabelNames(queryClient, profileType, start, end);

  const labelNamesFromAPI = useExtractedLabelNames(
    labelNamesResponse.response,
    labelNamesResponse.error
  );

  const {
    result: labelValuesOriginal,
    loading: isLabelValuesLoading,
    refetch: labelValuesRefetch,
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType, start, end);

  const labelNameMappings = useLabelNameMappings(labelNamesFromAPI);

  const labelNames = useMemo(() => {
    return labelNameMappings.map(m => m.displayName);
  }, [labelNameMappings]);

  const labelValues = labelValuesOriginal.response;
  const refetchLabelNames = labelNamesRefetch;
  const refetchLabelValues = labelValuesRefetch;

  const value = useMemo(
    () => ({
      labelNames,
      labelValues,
      labelNameMappings,
      isLabelNamesLoading,
      isLabelValuesLoading,
      currentLabelName,
      setCurrentLabelName,
      shouldHandlePrefixes,
      refetchLabelValues,
      refetchLabelNames,
    }),
    [
      labelNames,
      labelValues,
      labelNameMappings,
      isLabelNamesLoading,
      isLabelValuesLoading,
      currentLabelName,
      shouldHandlePrefixes,
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
