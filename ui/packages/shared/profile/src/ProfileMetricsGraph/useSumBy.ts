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

import {useCallback, useEffect, useMemo, useState} from 'react';

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

  if (!profile.delta) {
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
  labels: string[] | undefined
): [string[], (labels: string[]) => void] => {
  const {navigateTo} = useParcaContext();
  const [userSelectedSumByParam, setUserSelectedSumByParam] = useURLState({
    param: 'sum_by',
    navigateTo,
  });

  console.log('labels', labels);

  console.log('userSelectedSumByParam', userSelectedSumByParam);

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

  console.log('userSelectedSumBy', userSelectedSumBy);

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
    setDefaultSumBy(getDefaultSumBy(profileType, labels));
  }, [profileType, labels]);

  useEffect(() => {
    if (profileType === undefined || labels === undefined) {
      return;
    }

    if (userSelectedSumBy !== undefined && userSelectedSumBy.length === 0) {
      // User has explicitly selected no sumBy, so don't reset it
      return;
    }

    if (userSelectedSumBy !== undefined && userSelectedSumBy.length > 0) {
      // If any of the user selected sumBy is present in the labels, then don't reset it
      if (userSelectedSumBy.some(sumBy => labels?.includes(sumBy))) {
        return;
      }
    }

    // Reset user selected sumBy if profile type changes
    setUserSelectedSumBy(['']);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileType, labels]);

  return [userSelectedSumBy ?? defaultSumBy ?? DEFAULT_EMPTY_SUM_BY, setUserSelectedSumBy];
};
