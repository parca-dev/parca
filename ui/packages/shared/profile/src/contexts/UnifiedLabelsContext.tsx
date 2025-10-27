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

import {createContext, useContext} from 'react';

import {QueryServiceClient} from '@parca/client';
import {Query} from '@parca/parser';

export interface ExternalLabelSource {
  type: string;
  labelNames: string[];
  isLoading: boolean;
  error?: Error | null;
  fetchLabelValues?: (labelName: string) => Promise<string[]>;
  refetchLabelNames?: () => Promise<void>;
  refetchLabelValues?: (labelName?: string) => Promise<void>;
}

interface UnifiedLabelsContextType {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  start?: number;
  end?: number;

  queryBrowserRef: React.RefObject<HTMLDivElement>;
  searchExecutedTimestamp?: number;

  externalLabelSource?: ExternalLabelSource;
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

  queryBrowserRef: React.RefObject<HTMLDivElement>;
  searchExecutedTimestamp?: number;
  externalLabelSource?: ExternalLabelSource;
}

export function UnifiedLabelsProvider({
  children,
  queryClient,
  setMatchersString,
  runQuery,
  currentQuery,
  profileType,
  queryBrowserRef,
  searchExecutedTimestamp,
  externalLabelSource,
  start,
  end,
}: UnifiedLabelsProviderProps): JSX.Element {
  const value = {
    externalLabelSource,
    queryClient,
    setMatchersString,
    runQuery,
    currentQuery,
    profileType,
    queryBrowserRef,
    searchExecutedTimestamp,
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
