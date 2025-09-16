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
import {useURLState} from '@parca/components';
import {Query} from '@parca/parser';
import {testId, TEST_IDS} from '@parca/test-utils';
import type {NavigateFunction} from '@parca/utilities';

import {ProfileDiffSource, ProfileSelection, ProfileViewWithData} from '..';
import ProfileSelector, {QuerySelection} from '../ProfileSelector';

interface ProfileExplorerCompareProps {
  queryClient: QueryServiceClient;

  queryA: QuerySelection;
  queryB: QuerySelection;
  profileA: ProfileSelection | null;
  profileB: ProfileSelection | null;
  selectQueryA: (query: QuerySelection) => void;
  selectQueryB: (query: QuerySelection) => void;
  selectProfileA: (source: ProfileSelection) => void;
  selectProfileB: (source: ProfileSelection) => void;
  closeProfile: (card: string) => void;

  navigateTo: NavigateFunction;
}

const ProfileExplorerCompare = ({
  queryClient,
  queryA,
  queryB,
  profileA,
  profileB,
  selectQueryA,
  selectQueryB,
  selectProfileA,
  selectProfileB,
  closeProfile,
  navigateTo,
}: ProfileExplorerCompareProps): JSX.Element => {
  const [showMetricsGraph, setShowMetricsGraph] = useState(true);

  const closeProfileA = (): void => {
    closeProfile('A');
  };

  const closeProfileB = (): void => {
    closeProfile('B');
  };

  const [compareAbsolute] = useURLState('compare_absolute');

  return (
    <div {...testId(TEST_IDS.COMPARE_CONTAINER)}>
      <div className="flex justify-between gap-2 relative mb-2">
        <div className="flex-column flex-1 p-2 shadow-md rounded-md" {...testId(TEST_IDS.COMPARE_SIDE_A)}>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryA}
            profileSelection={profileA}
            selectProfile={selectProfileA}
            selectQuery={selectQueryA}
            closeProfile={closeProfileA}
            enforcedProfileName={''}
            comparing={true}
            navigateTo={navigateTo}
            suffix="_a"
            showMetricsGraph={showMetricsGraph}
            setDisplayHideMetricsGraphButton={setShowMetricsGraph}
          />
        </div>
        <div className="flex-column flex-1 p-2 shadow-md rounded-md" {...testId(TEST_IDS.COMPARE_SIDE_B)}>
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryB}
            profileSelection={profileB}
            selectProfile={selectProfileB}
            selectQuery={selectQueryB}
            closeProfile={closeProfileB}
            enforcedProfileName={Query.parse(queryA.expression).profileName()}
            comparing={true}
            navigateTo={navigateTo}
            suffix="_b"
            showMetricsGraph={showMetricsGraph}
            setDisplayHideMetricsGraphButton={setShowMetricsGraph}
          />
        </div>
      </div>
      <div className="grid grid-cols-1">
        {profileA != null && profileB != null ? (
          <div {...testId(TEST_IDS.COMPARE_PROFILE_VIEW)}>
            <ProfileViewWithData
              queryClient={queryClient}
              profileSource={
                new ProfileDiffSource(
                  profileA.ProfileSource(),
                  profileB.ProfileSource(),
                  compareAbsolute === 'true'
                )
              }
            />
          </div>
        ) : (
          <div>
            <div className="my-20 text-center">
              <p>Select a profile on both sides.</p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ProfileExplorerCompare;
