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

import {useState} from 'react';

import {QueryServiceClient} from '@parca/client';
import {DateTimeRange} from '@parca/components';
import {ProfileType} from '@parca/parser';

import {useLabelNames} from './hooks/useLabels';

export const DEFAULT_EMPTY_SUM_BY: string[] = [];

const getDefaultSumBy = (
  profile: ProfileType | undefined,
  labels: string[] | undefined
): string[] | undefined => {
  if (profile === undefined || labels === undefined) {
    return undefined;
  }

  if (!profile.delta) {
    return DEFAULT_EMPTY_SUM_BY;
  }

  if (labels.includes('comm')) {
    return ['comm'];
  }

  if (labels.includes('namespace')) {
    return ['namespace'];
  }

  if (labels.includes('container')) {
    return ['container'];
  }

  return undefined;
};

export const useSumBySelection = (
  profileType: ProfileType | undefined,
  labelNamesLoading: boolean,
  labels: string[] | undefined,
  draftSumBy: string[] | undefined,
  {
    defaultValue,
  }: {
    defaultValue?: string[];
  } = {}
): [
  string[] | undefined,
  (labels: string[]) => void,
  {
    isLoading: boolean;
  }
] => {
  const [userSelectedSumBy, setUserSelectedSumBy] = useState<Record<string, string[] | undefined>>(
    profileType != null ? {[profileType.toString()]: defaultValue} : {}
  );

  // Update userSelectedSumBy when defaultValue changes (e.g., during navigation)
  const [prevProfileType, setPrevProfileType] = useState(profileType);
  const [prevDefaultValue, setPrevDefaultValue] = useState(defaultValue);

  if (prevProfileType !== profileType || prevDefaultValue !== defaultValue) {
    setPrevProfileType(profileType);
    setPrevDefaultValue(defaultValue);
    if (profileType != null && defaultValue !== undefined) {
      setUserSelectedSumBy(prev => ({
        ...prev,
        [profileType.toString()]: defaultValue,
      }));
    }
  }

  const setSumBy = (sumBy: string[]): void => {
    setUserSelectedSumBy(prev => {
      if (profileType == null) {
        return prev;
      }

      return {
        ...prev,
        [profileType.toString()]: sumBy,
      };
    });
  };

  const {defaultSumBy} = useDefaultSumBy(profileType, labelNamesLoading, labels);

  // For smoother UX, return draftSumBy first if available during loading
  // as this must be recently computed with the draft time range labels.
  let sumBy: string[] | undefined;
  if (labelNamesLoading && draftSumBy !== undefined) {
    sumBy = draftSumBy;
  } else {
    // Prefer non-empty URL default over auto-computed default to avoid a
    // one-render race where defaultSumBy overwrites the default value from upstream.
    const hasExplicitDefault = defaultValue != null && defaultValue.length > 0;
    sumBy =
      userSelectedSumBy[profileType?.toString() ?? ''] ??
      (hasExplicitDefault ? defaultValue : undefined) ??
      defaultSumBy ??
      DEFAULT_EMPTY_SUM_BY;

    if (profileType?.delta !== true) {
      sumBy = DEFAULT_EMPTY_SUM_BY;
    }
  }

  return [
    sumBy,
    setSumBy,
    {
      isLoading: labelNamesLoading,
    },
  ];
};

export const useDefaultSumBy = (
  profileType: ProfileType | undefined,
  labelNamesLoading: boolean,
  labels: string[] | undefined
): {defaultSumBy: string[] | undefined; isLoading: boolean} => {
  const defaultSumBy = getDefaultSumBy(profileType, labels);

  return {defaultSumBy, isLoading: labelNamesLoading};
};

const getSumByFromParam = (param: string | string[] | undefined): string[] | undefined => {
  if (param?.length === 0) {
    return undefined;
  }

  if (param === '__none__') {
    return DEFAULT_EMPTY_SUM_BY;
  }

  if (typeof param === 'string') {
    // Handle comma-separated strings (e.g., "comm,node" -> ["comm", "node"])
    if (param.includes(',')) {
      return param
        .split(',')
        .map(s => s.trim())
        .filter(s => s.length > 0);
    }
    return [param];
  }

  return param;
};

export const useSumByFromParams = (param: string | string[] | undefined): string[] | undefined => {
  return getSumByFromParam(param);
};

export const sumByToParam = (sumBy: string[] | undefined): string | null => {
  if (sumBy === undefined) {
    return null;
  }

  if (sumBy.length === 0) {
    return '__none__';
  }

  return sumBy.join(',');
};

// Combined hook that handles all sumBy logic: fetching labels, computing defaults, and managing selection
export const useSumBy = (
  queryClient: QueryServiceClient,
  profileType: ProfileType | undefined,
  timeRange: DateTimeRange,
  draftProfileType: ProfileType | undefined,
  draftTimeRange: DateTimeRange,
  defaultValue?: string[]
): {
  sumBy: string[] | undefined;
  setSumBy: (sumBy: string[]) => void;
  isLoading: boolean;
  draftSumBy: string[] | undefined;
  setDraftSumBy: (sumBy: string[] | undefined) => void;
  isDraftSumByLoading: boolean;
} => {
  const {loading: labelNamesLoading, result} = useLabelNames(
    queryClient,
    profileType?.toString() ?? '',
    timeRange.getFromMs(),
    timeRange.getToMs()
  );

  const {draftSumBy, setDraftSumBy, isDraftSumByLoading} = useDraftSumBy(
    queryClient,
    draftProfileType,
    draftTimeRange,
    defaultValue
  );

  const labels = result.response?.labelNames === undefined ? [] : result.response.labelNames;

  const [sumBySelection, setSumByInternal, {isLoading}] = useSumBySelection(
    profileType,
    labelNamesLoading,
    labels,
    draftSumBy,
    {defaultValue}
  );

  return {
    sumBy: sumBySelection,
    setSumBy: setSumByInternal,
    isLoading,
    draftSumBy,
    setDraftSumBy,
    isDraftSumByLoading,
  };
};

export const useDraftSumBy = (
  queryClient: QueryServiceClient,
  profileType: ProfileType | undefined,
  timeRange: DateTimeRange,
  defaultValue?: string[]
): {
  draftSumBy: string[] | undefined;
  setDraftSumBy: (sumBy: string[] | undefined) => void;
  isDraftSumByLoading: boolean;
} => {
  const [draftSumBy, setDraftSumBy] = useState<string[] | undefined>(defaultValue);
  const {loading: labelNamesLoading, result} = useLabelNames(
    queryClient,
    profileType?.toString() ?? '',
    timeRange.getFromMs(),
    timeRange.getToMs()
  );

  const labels = result.response?.labelNames === undefined ? [] : result.response.labelNames;

  const {defaultSumBy, isLoading} = useDefaultSumBy(profileType, labelNamesLoading, labels);

  return {
    draftSumBy:
      draftSumBy ??
      (defaultValue != null && defaultValue.length > 0 ? defaultValue : undefined) ??
      defaultSumBy ??
      DEFAULT_EMPTY_SUM_BY,
    setDraftSumBy,
    isDraftSumByLoading: isLoading,
  };
};
