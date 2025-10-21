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

interface LabelNameMapping {
  displayName: string;
  fullName: string;
}

interface LabelsContextType {
  labelNames: string[];
  labelValues: string[];
  labelNameMappings: LabelNameMapping[];
  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;
  currentLabelName: string | null;
  setCurrentLabelName: (name: string | null) => void;
  refetchLabelValues: () => void;
  refetchLabelNames: () => void;
}

const LabelsContext = createContext<LabelsContextType | null>(null);

interface LabelsProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  start?: number;
  end?: number;
}

export function LabelsProvider({
  children,
  queryClient,
  profileType,
  start,
  end,
}: LabelsProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = React.useState<string | null>(null);

  const {
    result: labelNamesResponse,
    loading: isLabelNamesLoading,
    refetch: refetchLabelNames,
  } = useLabelNames(queryClient, profileType, start, end);

  const labelNamesFromAPI = useMemo(() => {
    return (labelNamesResponse.error === undefined || labelNamesResponse.error == null) &&
      labelNamesResponse !== undefined &&
      labelNamesResponse != null
      ? labelNamesResponse.response?.labelNames.filter(e => e !== '__name__') ?? []
      : [];
  }, [labelNamesResponse]);

  const {
    result: labelValuesOriginal,
    loading: isLabelValuesLoading,
    refetch: refetchLabelValues,
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType, start, end);

  const labelNameMappings = useMemo(() => {
    return labelNamesFromAPI.map(name => ({
      displayName: name.replace(/^(attributes\.|attributes_resource\.)/, ''),
      fullName: name,
    }));
  }, [labelNamesFromAPI]);

  const labelNames = useMemo(() => {
    return labelNameMappings.map(m => m.displayName);
  }, [labelNameMappings]);

  const labelValues = useMemo(() => {
    return labelValuesOriginal.response;
  }, [labelValuesOriginal]);

  const value = useMemo(
    () => ({
      labelNames,
      labelValues,
      labelNameMappings,
      isLabelNamesLoading,
      isLabelValuesLoading,
      currentLabelName,
      setCurrentLabelName,
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
