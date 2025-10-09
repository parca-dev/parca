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

import {createContext, useCallback, useContext, useMemo} from 'react';

import {useQueryClient} from '@tanstack/react-query';

import {QueryServiceClient} from '@parca/client';

import {useLabelNames} from '../MatchersInput';
import {transformLabelsForSelect} from '../SimpleMatchers';
import type {SelectItem} from '../SimpleMatchers/Select';
import {useUtilizationLabels} from './UtilizationLabelsContext';

interface LabelNameSection {
  type: string;
  values: SelectItem[];
}

interface LabelContextValue {
  labelNameOptions: LabelNameSection[];
  isLoading: boolean;
  error: Error | null;
  refetchLabelValues: () => void;
  refetchLabelNames: () => void;
}

const LabelContext = createContext<LabelContextValue | null>(null);

interface LabelProviderProps {
  children: React.ReactNode;
  queryClient: QueryServiceClient;
  profileType: string;
  labelNameFromMatchers: string[];
  start?: number;
  end?: number;
}

// With there being the possibility of having utilization labels, we need to be able to determine whether the labels to be used are utilization labels or profiling data labels.
// This context is used to determine this.

export function LabelProvider({
  children,
  queryClient,
  profileType,
  labelNameFromMatchers,
  start,
  end,
}: LabelProviderProps): JSX.Element {
  const reactQueryClient = useQueryClient();
  const utilizationLabelResponse = useUtilizationLabels();
  const {
    loading,
    result,
    refetch: refetchLabelNamesQuery,
  } = useLabelNames(queryClient, profileType, start, end);

  const profileValues = useMemo(() => {
    const profileLabelNames =
      result.error != null ? [] : result.response?.labelNames.filter(e => e !== '__name__') ?? [];
    const uniqueProfileLabelNames = Array.from(new Set(profileLabelNames));

    return {
      labelNameOptions: uniqueProfileLabelNames,
      isLoading: loading,
      error: result.error ?? null,
    };
  }, [result, loading]);

  const utilizationValues = useMemo(() => {
    if (utilizationLabelResponse?.utilizationLabelNamesLoading === true) {
      return {labelNameOptions: [] as string[], isLoading: true};
    }
    if (
      utilizationLabelResponse == null ||
      utilizationLabelResponse.utilizationLabelNames == null
    ) {
      return {labelNameOptions: [] as string[], isLoading: false};
    }

    const uniqueUtilizationLabelNames = Array.from(
      new Set(utilizationLabelResponse.utilizationLabelNames)
    );
    return {
      labelNameOptions: uniqueUtilizationLabelNames,
      isLoading: utilizationLabelResponse.utilizationLabelNamesLoading,
    };
  }, [utilizationLabelResponse]);

  const value = useMemo(() => {
    if (
      profileValues.error != null ||
      profileValues.isLoading ||
      utilizationValues.isLoading === true
    ) {
      return {
        labelNameOptions: [],
        isLoading: (profileValues.isLoading || utilizationValues.isLoading) ?? false,
        error: profileValues.error,
      };
    }

    let nonMatchingLabels = labelNameFromMatchers.filter(
      label => !utilizationValues.labelNameOptions.includes(label)
    );
    nonMatchingLabels = nonMatchingLabels.filter(
      label => !profileValues.labelNameOptions.includes(label)
    );

    const nonMatchingLabelsSet = Array.from(new Set(nonMatchingLabels));
    const options = [
      {
        type: 'cpu',
        values: transformLabelsForSelect(profileValues.labelNameOptions),
      },
      {
        type: 'gpu',
        values: transformLabelsForSelect(utilizationValues.labelNameOptions),
      },
      {
        type: '',
        values: transformLabelsForSelect(nonMatchingLabelsSet),
      },
    ];

    return {
      labelNameOptions: options.filter(e => e.values.length > 0),
      isLoading: false,
      error: null,
    };
  }, [profileValues, utilizationValues, labelNameFromMatchers]);

  const refetchLabelValues = useCallback(() => {
    void reactQueryClient.refetchQueries({
      predicate: query => {
        const key = query.queryKey;
        return (
          Array.isArray(key) &&
          key.length === 4 &&
          typeof key[0] === 'string' &&
          key[1] === profileType
        );
      },
    });
  }, [reactQueryClient, profileType]);

  const refetchLabelNames = useCallback(() => {
    refetchLabelNamesQuery();
  }, [refetchLabelNamesQuery]);

  const contextValue = useMemo(
    () => ({
      ...value,
      refetchLabelValues,
      refetchLabelNames,
    }),
    [value, refetchLabelValues, refetchLabelNames]
  );

  return <LabelContext.Provider value={contextValue}>{children}</LabelContext.Provider>;
}

export function useLabels(): LabelContextValue {
  const context = useContext(LabelContext);
  if (context === null) {
    throw new Error('useLabels must be used within a LabelProvider');
  }
  return context;
}
