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

import cx from 'classnames';

import {QueryServiceClient} from '@parca/client';
import type {NavigateFunction} from '@parca/utilities';

import {ProfileSelection, ProfileViewWithData} from '..';
import ProfileSelector, {QuerySelection} from '../ProfileSelector';

interface ProfileExplorerSingleProps {
  queryClient: QueryServiceClient;
  query: QuerySelection;
  selectQuery: (query: QuerySelection) => void;
  selectProfile: (source: ProfileSelection) => void;
  profile: ProfileSelection | null;
  navigateTo: NavigateFunction;
}

const ProfileExplorerSingle = ({
  queryClient,
  query,
  selectQuery,
  selectProfile,
  profile,
  navigateTo,
}: ProfileExplorerSingleProps): JSX.Element => {
  const [showMetricsGraph, setShowMetricsGraph] = useState(true);
  const [showButton, setShowButton] = useState(false);

  return (
    <>
      <div
        className="relative"
        onMouseEnter={() => {
          if (!showMetricsGraph) return;
          setShowButton(true);
        }}
        onMouseLeave={() => {
          if (!showMetricsGraph) return;
          setShowButton(false);
        }}
      >
        <button
          onClick={() => setShowMetricsGraph(!showMetricsGraph)}
          className={cx(
            'hidden right-0 bottom-3 z-10 px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:bg-gray-900',
            showButton && showMetricsGraph && 'absolute !flex',
            !showMetricsGraph && 'relative !flex mt-3 ml-auto'
          )}
        >
          {showMetricsGraph ? 'Hide' : 'Show'} Metrics Graph
        </button>

        {showMetricsGraph ? (
          <ProfileSelector
            queryClient={queryClient}
            querySelection={query}
            selectQuery={selectQuery}
            selectProfile={selectProfile}
            closeProfile={() => {}} // eslint-disable-line @typescript-eslint/no-empty-function
            profileSelection={profile}
            comparing={false}
            enforcedProfileName={''} // TODO
            navigateTo={navigateTo}
            suffix="_a"
          />
        ) : (
          <></>
        )}
      </div>

      {profile != null ? (
        <ProfileViewWithData queryClient={queryClient} profileSource={profile.ProfileSource()} />
      ) : (
        <></>
      )}
    </>
  );
};

export default ProfileExplorerSingle;
