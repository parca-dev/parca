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
import {parseParams} from '../index';
import {
  useAppDispatch,
  useAppSelector,
  setProfileStateValue,
  selectProfileStateValue,
} from '@parca/store';

interface Props {
  param: string;
  navigateTo?: (...args: any[]) => void;
  withURLUpdate?: boolean;
}

export const useURLState = ({param, navigateTo, withURLUpdate = true}) => {
  const dispatch = useAppDispatch();
  const router = parseParams(window.location.search);

  // 1. set initial value to the store value or URL value
  const value = useAppSelector(selectProfileStateValue(param)) ?? router[param];
  const setValue = value => dispatch(setProfileStateValue({key: param, value}));

  // whenever the store value changes, (optionally) update the URL
  useEffect(() => {
    const isEmpty = val => val === undefined || val === null || val === '';

    if (withURLUpdate && navigateTo) {
      if (router[param] !== value) {
        const searchParams = router;
        searchParams[param] = value;
        Object.keys(searchParams).forEach(
          key => isEmpty(searchParams[key]) && delete searchParams[key]
        );

        navigateTo('/', {...searchParams}, {replace: true});
      }
    }
  }, [value]);

  return [value, setValue] as const;
};
