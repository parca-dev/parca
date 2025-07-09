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

import {type FlamegraphArrow} from '@parca/client';

import ProfileFlameGraph from '../../ProfileFlameGraph';
import {type CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';
import {type ProfileSource} from '../../ProfileSource';

interface CalleesSectionProps {
  calleesRef: React.RefObject<HTMLDivElement>;
  calleesFlamegraphResponse?: {
    report: {
      oneofKind: string;
      flamegraphArrow?: FlamegraphArrow;
    };
    total?: string;
  };
  calleesFlamegraphLoading: boolean;
  calleesFlamegraphError: any;
  filtered: bigint;
  profileSource: ProfileSource;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  metadataMappingFiles?: string[];
}

export function CalleesSection({
  calleesRef,
  calleesFlamegraphResponse,
  calleesFlamegraphLoading,
  calleesFlamegraphError,
  filtered,
  profileSource,
  curPathArrow,
  setCurPathArrow,
  metadataMappingFiles,
}: CalleesSectionProps): JSX.Element {
  return (
    <div className="flex relative items-start flex-row" ref={calleesRef}>
      <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
        {'<-'} Callees
      </div>
      <ProfileFlameGraph
        arrow={
          calleesFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
            ? calleesFlamegraphResponse?.report?.flamegraphArrow
            : undefined
        }
        total={BigInt(calleesFlamegraphResponse?.total ?? '0')}
        filtered={filtered}
        profileType={profileSource?.ProfileType()}
        loading={calleesFlamegraphLoading}
        error={calleesFlamegraphError}
        isHalfScreen={true}
        width={
          calleesRef.current != null ? calleesRef.current.getBoundingClientRect().width - 25 : 0
        }
        metadataMappingFiles={metadataMappingFiles}
        metadataLoading={false}
        isSandwichFlameGraph={true}
        curPathArrow={curPathArrow}
        setNewCurPathArrow={setCurPathArrow}
        profileSource={profileSource}
        tooltipId="callees"
      />
    </div>
  );
}
