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

import React, {useMemo} from 'react';

import {Vector, tableFromIPC} from 'apache-arrow';
import {Tooltip} from 'react-tooltip';

import {type FlamegraphArrow} from '@parca/client';
import {Button} from '@parca/components';

import ProfileIcicleGraph from '../../ProfileIcicleGraph';
import {type CurrentPathFrame} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';
import {type ProfileSource} from '../../ProfileSource';

const FIELD_DEPTH = 'depth';

function getMaxDepth(depthColumn: Vector<any> | null): number {
  if (depthColumn === null) return 0;

  let max = 0;
  for (const val of depthColumn) {
    const numVal = Number(val);
    if (numVal > max) max = numVal;
  }
  return max;
}

interface CallersSectionProps {
  callersRef: React.RefObject<HTMLDivElement>;
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
  isExpanded: boolean;
  setIsExpanded: (isExpanded: boolean) => void;
  defaultMaxFrames: number;
}

export function CallersSection({
  callersRef,
  callersFlamegraphResponse,
  callersFlamegraphLoading,
  callersFlamegraphError,
  filtered,
  profileSource,
  curPathArrow,
  setCurPathArrow,
  metadataMappingFiles,
  isExpanded,
  setIsExpanded,
  defaultMaxFrames,
}: CallersSectionProps): JSX.Element {
  const maxDepth = useMemo(() => {
    if (
      callersFlamegraphResponse?.report.oneofKind === 'flamegraphArrow' &&
      callersFlamegraphResponse?.report?.flamegraphArrow != null
    ) {
      const table = tableFromIPC(callersFlamegraphResponse.report.flamegraphArrow.record);
      const depthColumn = table.getChild(FIELD_DEPTH);
      return getMaxDepth(depthColumn);
    }
    return 0;
  }, [callersFlamegraphResponse]);

  const shouldShowButton = maxDepth > defaultMaxFrames;

  return (
    <>
      {shouldShowButton && (
        <Button
          variant="neutral"
          onClick={() => setIsExpanded(!isExpanded)}
          className="absolute right-8 top-[-46px] z-10"
          type="button"
        >
          <span
            data-tooltip-content={
              !isExpanded
                ? `This profile has ${maxDepth} frames, showing only the top ${defaultMaxFrames} frames. Click to show more frames.`
                : `This profile has ${maxDepth} frames, showing all frames. Click to hide frames.`
            }
            data-tooltip-id="show-more-frames"
          >
            {isExpanded ? 'Hide frames' : 'Show more frames'}
          </span>
          <Tooltip id="show-more-frames" />
        </Button>
      )}
      <div className="flex relative flex-row overflow-hidden" ref={callersRef}>
        <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left flex-shrink-0">
          Callers {'->'}
        </div>
        <div className="flex-1 overflow-hidden relative">
          <ProfileIcicleGraph
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
              callersRef.current != null ? callersRef.current.getBoundingClientRect().width - 25 : 0
            }
            metadataMappingFiles={metadataMappingFiles}
            metadataLoading={false}
            isSandwichIcicleGraph={true}
            curPathArrow={curPathArrow}
            setNewCurPathArrow={setCurPathArrow}
            isFlamegraph={true}
            profileSource={profileSource}
            tooltipId="callers"
            maxFrameCount={defaultMaxFrames}
            isExpanded={isExpanded}
          />
        </div>
      </div>
    </>
  );
}
