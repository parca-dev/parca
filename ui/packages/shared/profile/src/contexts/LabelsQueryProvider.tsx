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

import {createContext, useContext, useState} from 'react';

import {QueryServiceClient} from '@parca/client';
import {Query} from '@parca/parser';

import {useLabelNames, useLabelValues} from '../hooks/useLabels';
import {useExtractedLabelNames} from './utils';

interface LabelsQueryProviderContextType {
  isLabelNamesLoading: boolean;
  isLabelValuesLoading: boolean;
  currentLabelName: string | null;
  setCurrentLabelName: (name: string | null) => void;
  refetchLabelValues: () => Promise<void>;
  refetchLabelNames: () => Promise<void>;

  labelNames: string[];
  labelValues: string[];

  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  start?: number;
  end?: number;
}

const LabelsQueryProviderContext = createContext<LabelsQueryProviderContextType | null>(null);

interface LabelsQueryProviderProps {
  children: React.ReactNode;

  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  start?: number;
  end?: number;
}

export function LabelsQueryProvider({
  children,
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
  profileType,
  start,
  end,
}: LabelsQueryProviderProps): JSX.Element {
  const [currentLabelName, setCurrentLabelName] = useState<string | null>(null);

  const {
    result: labelNamesResponse,
    loading: isLabelNamesLoading,
    refetch: labelNamesRefetch,
  } = useLabelNames(queryClient, profileType, start, end);

  const labelNames = useExtractedLabelNames(labelNamesResponse.response, labelNamesResponse.error);

  const {
    result: labelValuesOriginal,
    loading: isLabelValuesLoading,
    refetch: labelValuesRefetch,
  } = useLabelValues(queryClient, currentLabelName ?? '', profileType, start, end);

  const labelValues = labelValuesOriginal.response;

  const value = {
    labelNames,
    labelValues,
    isLabelNamesLoading,
    isLabelValuesLoading,
    refetchLabelValues: labelValuesRefetch,
    refetchLabelNames: labelNamesRefetch,
    queryClient,
    setMatchersString,
    runQuery,
    currentQuery,
    profileType,
    start,
    end,
    setCurrentLabelName,
    currentLabelName,
  };

  return (
    <LabelsQueryProviderContext.Provider value={value}>
      {children}
    </LabelsQueryProviderContext.Provider>
  );
}

export function useLabelsQueryProvider(): LabelsQueryProviderContextType {
  const context = useContext(LabelsQueryProviderContext);
  if (context === null) {
    throw new Error('useLabelsQueryProvider must be used within a LabelsQueryProvider');
  }
  return context;
}
