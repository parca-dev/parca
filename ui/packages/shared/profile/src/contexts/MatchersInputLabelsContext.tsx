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

import {useFetchUtilizationLabelValues, useLabelNames, useLabelValues} from '../MatchersInput';
import {useUtilization} from './UtilizationContext';

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
  shouldHandlePrefixes: boolean;
}

const LabelsContext = createContext<LabelsContextType | null>(null);

interface LabelsProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  start?: number;
  end?: number;
}

// With there being the possibility of having utilization labels, we need to be able to determine whether the labels to be used are utilization labels or profiling data labels.
// This context is used to determine this.
export function LabelsProvider({
  children,
  queryClient,
  profileType,
  start,
  end,
}: LabelsProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = React.useState<string | null>(null);
  const utilizationContext = useUtilization();

  const {result: labelNamesResponse, loading: isLabelNamesLoading} = useLabelNames(
    queryClient,
    profileType,
    start,
    end
  );

  const labelNamesFromAPI = useMemo(() => {
    return (labelNamesResponse.error === undefined || labelNamesResponse.error == null) &&
      labelNamesResponse !== undefined &&
      labelNamesResponse != null
      ? labelNamesResponse.response?.labelNames.filter(e => e !== '__name__') ?? []
      : [];
  }, [labelNamesResponse]);

  const {result: labelValuesOriginal, loading: isLabelValuesLoading} = useLabelValues(
    queryClient,
    currentLabelName ?? '',
    profileType,
    start,
    end
  );

  const utilizationLabelValues = useFetchUtilizationLabelValues(
    currentLabelName ?? '',
    utilizationContext
  );

  const shouldHandlePrefixes = utilizationContext?.utilizationLabels?.labelNames !== undefined;

  const labelNameMappings = useMemo(() => {
    const names = utilizationContext?.utilizationLabels?.labelNames ?? labelNamesFromAPI;
    return names.map(name => ({
      displayName: name.replace(/^(attributes\.|attributes_resource\.)/, ''),
      fullName: name,
    }));
  }, [labelNamesFromAPI, utilizationContext?.utilizationLabels?.labelNames]);

  const labelNames = useMemo(() => {
    return shouldHandlePrefixes ? labelNameMappings.map(m => m.displayName) : labelNamesFromAPI;
  }, [labelNameMappings, labelNamesFromAPI, shouldHandlePrefixes]);

  const labelValues = useMemo(() => {
    return utilizationContext?.utilizationLabels?.fetchLabelValues !== undefined
      ? utilizationLabelValues
      : labelValuesOriginal.response;
  }, [
    labelValuesOriginal,
    utilizationLabelValues,
    utilizationContext?.utilizationLabels?.fetchLabelValues,
  ]);

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
    }),
    [
      labelNames,
      labelValues,
      labelNameMappings,
      isLabelNamesLoading,
      isLabelValuesLoading,
      currentLabelName,
      shouldHandlePrefixes,
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
