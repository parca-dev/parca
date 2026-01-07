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

import React from 'react';

import {TEST_IDS, testId} from '@parca/test-utils';

import ProfileFlameGraph from '../../ProfileFlameGraph';
import {type CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';
import {type ProfileSource} from '../../ProfileSource';
import {FlamegraphData} from '../../ProfileView/types/visualization';

interface CalleesSectionProps {
  calleesRef: React.RefObject<HTMLDivElement>;
  calleesFlamegraphData: FlamegraphData;
  profileSource: ProfileSource;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  metadataMappingFiles?: string[];
}

export function CalleesSection({
  calleesRef,
  calleesFlamegraphData,
  profileSource,
  curPathArrow,
  setCurPathArrow,
}: CalleesSectionProps): JSX.Element {
  return (
    <div
      className="flex relative items-start flex-row"
      ref={calleesRef}
      {...testId(TEST_IDS.SANDWICH_CALLEES_SECTION)}
    >
      <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
        {'<-'} Callees
      </div>
      <ProfileFlameGraph
        arrow={calleesFlamegraphData?.arrow}
        total={calleesFlamegraphData.total ?? BigInt(0)}
        filtered={calleesFlamegraphData.filtered ?? BigInt(0)}
        profileType={profileSource?.ProfileType()}
        loading={calleesFlamegraphData.loading}
        error={calleesFlamegraphData.error}
        isHalfScreen={true}
        width={
          calleesRef.current != null ? calleesRef.current.getBoundingClientRect().width - 25 : 0
        }
        metadataMappingFiles={calleesFlamegraphData.metadataMappingFiles}
        metadataLoading={calleesFlamegraphData.metadataLoading}
        isInSandwichView={true}
        curPathArrow={curPathArrow}
        setNewCurPathArrow={setCurPathArrow}
        profileSource={profileSource}
        tooltipId="callees"
      />
    </div>
  );
}
