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

import {useEffect} from 'react';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks/';
import {
  selectProfileStateValue,
  setProfileStateValue,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {parseParams} from '@parca/utilities';

interface Props {
  param: string;
  navigateTo?: (...args: any[]) => void;
  withURLUpdate?: boolean;
}

export const useURLState = ({
  param,
  navigateTo,
  withURLUpdate = true,
}: Props): [string | string[], (val: string | string[]) => void] => {
  const dispatch = useAppDispatch();
  const router = parseParams(window?.location.search);
  const [highlightAfterFilteringEnabled] = useUserPreference<boolean>(
    USER_PREFERENCES.HIGHTLIGHT_AFTER_FILTERING.key
  );

  // 1. set initial value to the store value or URL value
  const value = useAppSelector(selectProfileStateValue(param)) ?? router[param];
  const setValue = (
    value: string | string[]
  ): {payload: {key: string; value?: string | string[]}; type: string} =>
    dispatch(setProfileStateValue({key: param, value}));

  // whenever the store value changes, (optionally) update the URL
  useEffect(() => {
    const isEmpty = (val: string | string[] | undefined): boolean =>
      val === undefined || val == null || val === '';

    if (withURLUpdate && navigateTo !== undefined) {
      if (router[param] !== value) {
        const searchParams = router;
        searchParams[param] = value;

        if (param === 'filter_by_function') {
          searchParams.search_string = highlightAfterFilteringEnabled ? value : '';
        }

        if (param === 'dashboard_items') {
          searchParams[param] = value;
        }

        Object.keys(searchParams).forEach(
          // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
          key => isEmpty(searchParams[key]) && delete searchParams[key]
        );

        navigateTo('/', {...searchParams}, {replace: true});
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value, highlightAfterFilteringEnabled, param, withURLUpdate]);

  if (param === 'dashboard_items') {
    let dashboardItems: string[] = [];
    if (typeof value === 'string') {
      dashboardItems = [value] ?? [];
    } else {
      dashboardItems = value as string[];
    }
    return [dashboardItems, setValue];
  }

  return [value as string[], setValue];
};
