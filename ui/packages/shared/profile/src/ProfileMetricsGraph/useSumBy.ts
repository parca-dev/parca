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
  labels: string[] | undefined
): [string[], (labels: string[]) => void] => {
  const {navigateTo} = useParcaContext();
  const [userSelectedSumByParam, setUserSelectedSumByParam] = useURLState({
    param: 'sum_by',
    navigateTo,
  });

  const userSelectedSumBy = useMemo<string[] | undefined>(() => {
    if (userSelectedSumByParam?.length === 0) {
      return undefined;
    }

    if (userSelectedSumByParam === '__none__') {
      return [];
    }

    if (typeof userSelectedSumByParam === 'string') {
      return [userSelectedSumByParam];
    }

    return userSelectedSumByParam;
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

  const [defaultSumBy, setDefaultSumBy] = useState<string[] | undefined>(
    getDefaultSumBy(profileType, labels)
  );

  useEffect(() => {
    if (labelNamesLoading) {
      return;
    }
    setDefaultSumBy(getDefaultSumBy(profileType, labels));
  }, [profileType, labels, labelNamesLoading]);

  let sumBy = userSelectedSumBy ?? defaultSumBy ?? DEFAULT_EMPTY_SUM_BY;

  if (profileType?.delta !== true) {
    sumBy = [];
  }

  return [sumBy, setUserSelectedSumBy];
};
