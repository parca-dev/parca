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

import {ReactNode, createContext, useContext} from 'react';

export interface UtilizationMetrics {
  metrics: Array<{
    name: string;
    humanReadableName: string;
    data: UtilizationSeries[];
    loading?: boolean;
    renderAs?: 'standard' | 'throughput';
    groupWith?: string[];
    yAxisUnit?: string;
  }>;
  loading: boolean;
}

export interface UtilizationSeries {
  isSelected: boolean;
  labelset: {
    labels: Array<{
      name: string;
      value: string;
    }>;
  };
  samples: Array<{
    timestamp: number;
    value: number;
  }>;
}

export interface UtilizationLabels {
  labelNames?: string[];
  fetchLabelValues?: (key: string) => Promise<string[]>;
  labelValues?: string[];
  labelNamesLoading?: boolean;
}

export interface UtilizationProps {
  utilizationMetrics?: UtilizationMetrics;
  utilizationLabels?: UtilizationLabels;
  onUtilizationSeriesSelect?: (seriesIndex: number) => void;
}

interface UtilizationProviderProps {
  children: ReactNode;
  value: UtilizationProps | undefined;
}

// The UtilizationContext is used to store the utilization label names and values, metrics and loading state. It also
// contains the function utilizationFetchLabelValues to fetch the utilization label values.
// This context was created so as to avoid props drilling.
const UtilizationContext = createContext<UtilizationProps | undefined>(undefined);

export function UtilizationProvider({children, value}: UtilizationProviderProps): JSX.Element {
  return <UtilizationContext.Provider value={value}>{children}</UtilizationContext.Provider>;
}

export function useUtilization(): UtilizationProps | undefined {
  const context = useContext(UtilizationContext);
  return context;
}
