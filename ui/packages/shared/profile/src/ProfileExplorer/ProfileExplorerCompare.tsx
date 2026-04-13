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

import {QueryServiceClient} from '@parca/client';
import {useURLStateBatch} from '@parca/components';
import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import type {NavigateFunction} from '@parca/utilities';

import {ProfileDiffSource, ProfileViewWithData} from '..';
import ProfileSelector from '../ProfileSelector';
import {useCompareModeMeta} from '../hooks/useCompareModeMeta';
import {useQueryState} from '../hooks/useQueryState';

interface ProfileExplorerCompareProps {
  queryClient: QueryServiceClient;
  navigateTo: NavigateFunction;
}

const ProfileExplorerCompare = ({
  queryClient,
  navigateTo,
}: ProfileExplorerCompareProps): JSX.Element => {
  const [showMetricsGraph, setShowMetricsGraph] = useState(true);
  const batchUpdates = useURLStateBatch();
  const {closeCompareMode, isCompareMode, isCompareAbsolute} = useCompareModeMeta();

  // Read ProfileSource states from URL for both sides
  const {profileSource: profileSourceA, querySelection: querySelectionA} = useQueryState({
    suffix: '_a',
    comparing: true,
  });
  const {
    profileSource: profileSourceB,
    querySelection: querySelectionB,
    commitDraft: commitDraftB,
    setDraftExpression: setDraftExpressionB,
    setDraftTimeRange: setDraftTimeRangeB,
  } = useQueryState({suffix: '_b', comparing: true});

  // Derive enforced profile name from side A's expression
  const enforcedProfileNameA = useMemo(() => {
    return querySelectionA.expression !== ''
      ? Query.parse(querySelectionA.expression).profileName()
      : '';
  }, [querySelectionA.expression]);

  // Initialize side B with side A's values if side B is empty
  useEffect(() => {
    // If not in compare mode, don't initialize
    if (!isCompareMode) {
      return;
    }

    if (querySelectionB.expression === '' && querySelectionA.expression !== '') {
      batchUpdates(() => {
        setDraftExpressionB(querySelectionA.expression);
        setDraftTimeRangeB(querySelectionA.from, querySelectionA.to, querySelectionA.timeSelection);
        // Commit to update the URL and trigger metrics graph load
        commitDraftB();
      });
    }
  }, [
    isCompareMode,
    querySelectionA.expression,
    querySelectionA.from,
    querySelectionA.to,
    querySelectionA.timeSelection,
    querySelectionB.expression,
    setDraftExpressionB,
    setDraftTimeRangeB,
    commitDraftB,
    batchUpdates,
  ]);

  const closeProfileA = useCallback((): void => {
    closeCompareMode('A');
  }, [closeCompareMode]);

  const closeProfileB = useCallback((): void => {
    closeCompareMode('B');
  }, [closeCompareMode]);

  return (
    <div {...testId(TEST_IDS.COMPARE_CONTAINER)}>
      <div className="flex justify-between gap-2 relative mb-2">
        <div
          className="flex-column flex-1 p-2 shadow-md rounded-md"
          {...testId(TEST_IDS.COMPARE_SIDE_A)}
        >
          <ProfileSelector
            queryClient={queryClient}
            closeProfile={closeProfileA}
            enforcedProfileName={''}
            comparing={true}
            navigateTo={navigateTo}
            suffix="_a"
            showMetricsGraph={showMetricsGraph}
            setDisplayHideMetricsGraphButton={setShowMetricsGraph}
          />
        </div>
        <div
          className="flex-column flex-1 p-2 shadow-md rounded-md"
          {...testId(TEST_IDS.COMPARE_SIDE_B)}
        >
          <ProfileSelector
            queryClient={queryClient}
            closeProfile={closeProfileB}
            enforcedProfileName={enforcedProfileNameA}
            comparing={true}
            navigateTo={navigateTo}
            suffix="_b"
            showMetricsGraph={showMetricsGraph}
            setDisplayHideMetricsGraphButton={setShowMetricsGraph}
          />
        </div>
      </div>
      <div className="grid grid-cols-1">
        {profileSourceA != null && profileSourceB != null ? (
          <div {...testId(TEST_IDS.COMPARE_PROFILE_VIEW)}>
            <ProfileViewWithData
              queryClient={queryClient}
              profileSource={
                new ProfileDiffSource(profileSourceA, profileSourceB, isCompareAbsolute)
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
