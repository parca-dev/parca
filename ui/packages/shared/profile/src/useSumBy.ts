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

import {useCallback, useMemo} from 'react';

import {useParcaContext, useURLState} from '@parca/components';
import {ProfileType} from '@parca/parser';

export const DEFAULT_EMPTY_SUM_BY: string[] = [];

const getDefaultSumBy = (
  profile: ProfileType | undefined,
  labels: string[] | undefined
): string[] | undefined => {
  if (profile === undefined || labels === undefined) {
    return undefined;
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

export const useSumBy = (
  profileType: ProfileType | undefined,
  labelNamesLoading: boolean,
  labels: string[] | undefined,
  {
    urlParamKey = 'sum_by',
    withURLUpdate = true,
    defaultValue,
  }: {
    urlParamKey?: string;
    withURLUpdate?: boolean;
    defaultValue?: string[];
  } = {}
): [
  string[],
  (labels: string[]) => void,
  {userSelectedSumBy: string[] | undefined; isLoading: boolean},
] => {
  const {navigateTo} = useParcaContext();
  const [userSelectedSumByParam, setUserSelectedSumByParam] = useURLState({
    param: urlParamKey,
    navigateTo,
    withURLUpdate,
  });

  const userSelectedSumBy = useMemo<string[] | undefined>(() => {
    if (userSelectedSumByParam?.length === 0) {
      return undefined;
    }

    if (userSelectedSumByParam === '__none__') {
      return [];
    }

    if (userSelectedSumByParam === undefined && defaultValue !== undefined) {
      return defaultValue;
    }

    if (typeof userSelectedSumByParam === 'string') {
      return [userSelectedSumByParam];
    }

    return userSelectedSumByParam;

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [userSelectedSumByParam]);

  const setUserSelectedSumBy = useCallback(
    (sumBy: string[]) => {
      if (sumBy.length === 0) {
        setUserSelectedSumByParam('__none__');
        return;
      }

      if (sumBy.length === 1) {
        // Handle this separately to take care of the empty string scenario
        setUserSelectedSumByParam(sumBy[0]);
        return;
      }

      setUserSelectedSumByParam(sumBy);
    },
    [setUserSelectedSumByParam]
  );

  const defaultSumBy = useMemo(() => {
    return getDefaultSumBy(profileType, labels);
  }, [profileType, labels]);

  let sumBy = userSelectedSumBy ?? defaultSumBy ?? DEFAULT_EMPTY_SUM_BY;

  if (profileType?.delta !== true) {
    sumBy = DEFAULT_EMPTY_SUM_BY;
  }

  return [sumBy, setUserSelectedSumBy, {userSelectedSumBy, isLoading: labelNamesLoading}];
};
