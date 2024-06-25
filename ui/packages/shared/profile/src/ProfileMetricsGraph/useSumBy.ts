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

import {useEffect, useState} from 'react';

import {ProfileType} from '@parca/parser';

const DEFAULT_EMPTY_SUM_BY = [];

const getDefaultSumBy = (
  profile: ProfileType | undefined,
  labels: string[] | undefined
): string[] | undefined => {
  if (profile === undefined || labels === undefined) {
    return undefined;
  }

  if (profile.delta) {
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
  const [userSelectedSumBy, setUserSelectedSumBy] = useState<string[] | undefined>(undefined);
  const [defaultSumBy, setDefaultSumBy] = useState<string[] | undefined>(
    getDefaultSumBy(profileType, labels)
  );

  useEffect(() => {
    setDefaultSumBy(getDefaultSumBy(profileType, labels));
  }, [profileType, labels]);

  useEffect(() => {
    if (profileType === undefined) {
      return;
    }

    // Reset user selected sumBy if profile type changes
    setUserSelectedSumBy(undefined);
  }, [profileType]);

  return [userSelectedSumBy ?? defaultSumBy ?? DEFAULT_EMPTY_SUM_BY, setUserSelectedSumBy];
};
