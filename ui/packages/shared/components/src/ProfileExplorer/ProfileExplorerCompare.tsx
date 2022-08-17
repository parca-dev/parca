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

import {ProfileDiffSource, ProfileSelection, ProfileViewWithData} from '@parca/profile';
import {Query} from '@parca/parser';
import {QueryServiceClient} from '@parca/client';

import {NavigateFunction} from '.';
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
  const closeProfileA = () => {
    closeProfile('A');
  };

  const closeProfileB = () => {
    closeProfile('B');
  };

  return (
    <>
      <div className="grid grid-cols-2">
        <div className="pr-2">
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryA}
            profileSelection={profileA}
            selectProfile={selectProfileA}
            selectQuery={selectQueryA}
            closeProfile={closeProfileA}
            enforcedProfileName={''}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </div>
        <div className="pl-2">
          <ProfileSelector
            queryClient={queryClient}
            querySelection={queryB}
            profileSelection={profileB}
            selectProfile={selectProfileB}
            selectQuery={selectQueryB}
            closeProfile={closeProfileB}
            enforcedProfileName={Query.parse(queryA.expression).profileName()}
            comparing={true}
            onCompareProfile={() => {}}
          />
        </div>
      </div>
      <div className="grid grid-cols-1">
        {profileA != null && profileB != null ? (
          <ProfileViewWithData
            navigateTo={navigateTo}
            queryClient={queryClient}
            profileSource={
              new ProfileDiffSource(profileA.ProfileSource(), profileB.ProfileSource())
            }
          />
        ) : (
          <div>
            <div className="my-20 text-center">
              <p>Select a profile on both sides.</p>
            </div>
          </div>
        )}
      </div>
    </>
  );
};

export default ProfileExplorerCompare;
