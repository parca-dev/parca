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
import {type ProfileType} from '@parca/parser';

import ProfileIcicleGraph from '../../ProfileIcicleGraph';
import {type CurrentPathFrame} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';
import {type ProfileSource} from '../../ProfileSource';

interface CalleesSectionProps {
  calleesRef: React.RefObject<HTMLDivElement>;
  isHalfScreen: boolean;
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
  profileSource?: ProfileSource;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  metadataMappingFiles?: string[];
  metadataLoading?: boolean;
}

export function CalleesSection({
  calleesRef,
  isHalfScreen,
  calleesFlamegraphResponse,
  calleesFlamegraphLoading,
  calleesFlamegraphError,
  filtered,
  profileSource,
  curPath,
  setCurPath,
  curPathArrow,
  setCurPathArrow,
  metadataMappingFiles,
  metadataLoading,
}: CalleesSectionProps) {
  return (
    <div className="flex relative items-start flex-row" ref={calleesRef}>
      <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
        {'<-'} Callees
      </div>
      <ProfileIcicleGraph
        curPath={curPath}
        setNewCurPath={setCurPath}
        arrow={
          calleesFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
            ? calleesFlamegraphResponse?.report?.flamegraphArrow
            : undefined
        }
        graph={undefined}
        total={BigInt(calleesFlamegraphResponse?.total ?? '0')}
        filtered={filtered}
        profileType={profileSource?.ProfileType()}
        loading={calleesFlamegraphLoading}
        error={calleesFlamegraphError}
        isHalfScreen={true}
        width={
          calleesRef.current != null
            ? isHalfScreen
              ? (calleesRef.current.getBoundingClientRect().width - 54) / 2
              : calleesRef.current.getBoundingClientRect().width - 16
            : 0
        }
        metadataMappingFiles={metadataMappingFiles}
        metadataLoading={metadataLoading}
        isSandwichIcicleGraph={true}
        curPathArrow={curPathArrow}
        setNewCurPathArrow={setCurPathArrow}
      />
    </div>
  );
}
