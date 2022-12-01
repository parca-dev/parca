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

import {useState, useEffect, Dispatch} from 'react';
import {useNavigate} from 'react-router-dom';
import {parseParams, convertToQueryParams} from '../index';
import {useAppDispatch, useAppSelector, setGlobalStateValue} from '@parca/store';

// every time the param value changes, the URL is updated
export const useURLState = (paramName: string, withURLUpdate: boolean = true) => {
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const router = parseParams(window.location.search);
  const storeValue = useAppSelector(state => state.profile[paramName]);

  // 1. set the initial state to the store value or the URL value
  const [value, setValue] = useState<string | string[] | undefined>(
    storeValue ?? router[paramName]
  );

  // 2. whenever the value changes
  useEffect(() => {
    const isEmpty = val => val === undefined || val === null || val === '';
    // 2a. update the local store value
    if (storeValue !== value) {
      dispatch(setGlobalStateValue({key: paramName, value}));
    }

    // 2b. and (optionally) update the URL if the value is not the same as the URL value
    if (withURLUpdate) {
      if (router[paramName] !== value) {
        const searchParams = router;
        searchParams[paramName] = value;
        Object.keys(searchParams).forEach(
          key => isEmpty(searchParams[key]) && delete searchParams[key]
        );

        navigate(
          {
            pathname: '/',
            search: `?${convertToQueryParams(searchParams)}`,
          },
          {replace: true}
        );
      }
    }
  }, [value]);

  return [value, setValue] as const;
};
