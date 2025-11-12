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

import {useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {ProfileType} from '@parca/parser';

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
  useEffect(() => {
    if (profileType != null && defaultValue !== undefined) {
      setUserSelectedSumBy(prev => ({
        ...prev,
        [profileType.toString()]: defaultValue,
      }));
    }
  }, [profileType, defaultValue]);

  const setSumBy = useCallback(
    (sumBy: string[]) => {
      setUserSelectedSumBy(prev => {
        if (profileType == null) {
          return prev;
        }

        return {
          ...prev,
          [profileType.toString()]: sumBy,
        };
      });
    },
    [setUserSelectedSumBy, profileType]
  );

  const {defaultSumBy} = useDefaultSumBy(profileType, labelNamesLoading, labels);

  // Store the last valid sumBy value to return during loading
  const lastValidSumByRef = useRef<string[]>(DEFAULT_EMPTY_SUM_BY);

  const sumBy = useMemo(() => {
    // If loading, return the last valid value to prevent input from blanking
    if (labelNamesLoading && lastValidSumByRef.current !== DEFAULT_EMPTY_SUM_BY) {
      return lastValidSumByRef.current;
    }

    let result =
      userSelectedSumBy[profileType?.toString() ?? ''] ?? defaultSumBy ?? DEFAULT_EMPTY_SUM_BY;

    if (profileType?.delta !== true) {
      result = DEFAULT_EMPTY_SUM_BY;
    }

    // Store the computed value for next loading state
    lastValidSumByRef.current = result;

    return result;
  }, [userSelectedSumBy, profileType, defaultSumBy, labelNamesLoading]);

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
  const defaultSumBy = useMemo(() => {
    return getDefaultSumBy(profileType, labels);
  }, [profileType, labels]);

  return {defaultSumBy, isLoading: labelNamesLoading};
};

const getSumByFromParam = (param: string | string[] | undefined): string[] | undefined => {
  if (param?.length === 0) {
    return undefined;
  }

  if (param === '__none__') {
    return [];
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
  const sumBy = useMemo(() => {
    return getSumByFromParam(param);
  }, [param]);

  return sumBy;
};

export const sumByToParam = (sumBy: string[] | undefined): string | string[] | undefined => {
  if (sumBy === undefined) {
    return undefined;
  }

  if (sumBy.length === 0) {
    return '__none__';
  }

  return sumBy;
};
