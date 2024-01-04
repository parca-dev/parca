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

import {ProfileTypesResponse} from '@parca/client';
import {selectAutoQuery, setAutoQuery, useAppDispatch, useAppSelector} from '@parca/store';

import {constructProfileName} from '../ProfileTypeSelector';

interface Props {
  selectedProfileName: string;
  profileTypesData: ProfileTypesResponse | undefined;
  setProfileName: (name: string) => void;
  setQueryExpression: () => void;
}

export const useAutoQuerySelector = ({
  selectedProfileName,
  profileTypesData,
  setProfileName,
  setQueryExpression,
}: Props): void => {
  const autoQuery = useAppSelector(selectAutoQuery);
  const dispatch = useAppDispatch();

  // Effect to load some initial data on load when is no selection
  useEffect(() => {
    void (async () => {
      if (selectedProfileName.length > 0) {
        return;
      }
      if (profileTypesData?.types == null || profileTypesData.types.length < 1) {
        return;
      }
      if (autoQuery === 'true') {
        // Autoquery already enabled.
        return;
      }
      dispatch(setAutoQuery('true'));
      let profileType = profileTypesData.types.find(type => type.name === 'parca_agent_cpu');
      if (profileType == null) {
        profileType = profileTypesData.types[0];
      }
      setProfileName(constructProfileName(profileType));
    })();
  }, [
    profileTypesData,
    selectedProfileName,
    autoQuery,
    dispatch,
    setQueryExpression,
    setProfileName,
  ]);

  useEffect(() => {
    void (async () => {
      if (
        autoQuery !== 'true' ||
        profileTypesData?.types == null ||
        profileTypesData.types.length < 1 ||
        selectedProfileName.length === 0
      ) {
        return;
      }
      setQueryExpression();
      dispatch(setAutoQuery('false'));
    })();
  }, [
    profileTypesData,
    setQueryExpression,
    autoQuery,
    setProfileName,
    dispatch,
    selectedProfileName,
  ]);
};
