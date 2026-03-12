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
import type {NavigateFunction} from '@parca/utilities';

import {ProfileViewWithData} from '..';
import ProfileSelector from '../ProfileSelector';
import {useQueryState} from '../hooks/useQueryState';

interface ProfileExplorerSingleProps {
  queryClient: QueryServiceClient;
  navigateTo: NavigateFunction;
}

const ProfileExplorerSingle = ({
  queryClient,
  navigateTo,
}: ProfileExplorerSingleProps): JSX.Element => {
  const [showMetricsGraph, setShowMetricsGraph] = useState(true);
  const {profileSource} = useQueryState({suffix: '_a'});

  return (
    <>
      <div className="relative">
        <ProfileSelector
          queryClient={queryClient}
          closeProfile={() => {}} // eslint-disable-line @typescript-eslint/no-empty-function
          comparing={false}
          enforcedProfileName={''}
          navigateTo={navigateTo}
          suffix="_a"
          showMetricsGraph={showMetricsGraph}
          setDisplayHideMetricsGraphButton={setShowMetricsGraph}
        />
      </div>

      {profileSource != null && (
        <ProfileViewWithData queryClient={queryClient} profileSource={profileSource} />
      )}
    </>
  );
};

export default ProfileExplorerSingle;
