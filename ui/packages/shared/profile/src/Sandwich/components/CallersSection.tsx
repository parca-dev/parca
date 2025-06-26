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

interface CallersSectionProps {
  callersRef: React.RefObject<HTMLDivElement>;
  isHalfScreen: boolean;
  callersFlamegraphResponse?: {
    report: {
      oneofKind: string;
      flamegraphArrow?: FlamegraphArrow;
    };
    total?: string;
  };
  callersFlamegraphLoading: boolean;
  callersFlamegraphError: any;
  filtered: bigint;
  profileSource: ProfileSource;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  metadataMappingFiles?: string[];
}

export function CallersSection({
  callersRef,
  isHalfScreen,
  callersFlamegraphResponse,
  callersFlamegraphLoading,
  callersFlamegraphError,
  filtered,
  profileSource,
  curPathArrow,
  setCurPathArrow,
  metadataMappingFiles,
}: CallersSectionProps): JSX.Element {
  return (
    <div className="flex relative flex-row" ref={callersRef}>
      <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
        Callers {'->'}
      </div>
      <ProfileFlameGraph
        arrow={
          callersFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
            ? callersFlamegraphResponse?.report?.flamegraphArrow
            : undefined
        }
        total={BigInt(callersFlamegraphResponse?.total ?? '0')}
        filtered={filtered}
        profileType={profileSource?.ProfileType()}
        loading={callersFlamegraphLoading}
        error={callersFlamegraphError}
        isHalfScreen={true}
        width={
          callersRef.current != null
            ? isHalfScreen
              ? (callersRef.current.getBoundingClientRect().width - 54) / 2
              : callersRef.current.getBoundingClientRect().width - 16
            : 0
        }
        metadataMappingFiles={metadataMappingFiles}
        metadataLoading={false}
        isSandwichFlameGraph={true}
        curPathArrow={curPathArrow}
        setNewCurPathArrow={setCurPathArrow}
        isFlamegraph={true}
        profileSource={profileSource}
        tooltipId="callers"
      />
    </div>
  );
}
