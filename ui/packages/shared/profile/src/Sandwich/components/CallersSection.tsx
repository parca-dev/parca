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

import {Column, tableFromIPC} from '@uwdata/flechette';
import {Tooltip} from 'react-tooltip';

import {Button} from '@parca/components';
import {TEST_IDS, testId} from '@parca/test-utils';

import ProfileFlameGraph from '../../ProfileFlameGraph';
import {type CurrentPathFrame} from '../../ProfileFlameGraph/FlameGraphArrow/utils';
import {type ProfileSource} from '../../ProfileSource';
import {FlamegraphData} from '../../ProfileView/types/visualization';

const FIELD_DEPTH = 'depth';

function getMaxDepth(depthColumn: Column<number> | null): number {
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
  callersFlamegraphData: FlamegraphData;
  profileSource: ProfileSource;
  curPathArrow: CurrentPathFrame[];
  setCurPathArrow: (path: CurrentPathFrame[]) => void;
  isExpanded: boolean;
  setIsExpanded: (isExpanded: boolean) => void;
  defaultMaxFrames: number;
}

export function CallersSection({
  callersRef,
  callersFlamegraphData,
  profileSource,
  curPathArrow,
  setCurPathArrow,
  isExpanded,
  setIsExpanded,
  defaultMaxFrames,
}: CallersSectionProps): JSX.Element {
  const maxDepth = useMemo(() => {
    if (callersFlamegraphData?.arrow != null) {
      // Copy to aligned buffer only if byteOffset is not 8-byte aligned (required for BigUint64Array)
      const record = callersFlamegraphData.arrow.record;
      const aligned = record.byteOffset % 8 === 0 ? record : new Uint8Array(record);
      const table = tableFromIPC(aligned, {useBigInt: true});
      const depthColumn = table.getChild(FIELD_DEPTH);
      return getMaxDepth(depthColumn);
    }
    return 0;
  }, [callersFlamegraphData]);

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
      <div
        className="flex relative flex-row overflow-hidden"
        ref={callersRef}
        {...testId(TEST_IDS.SANDWICH_CALLERS_SECTION)}
      >
        <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left flex-shrink-0">
          Callers {'->'}
        </div>
        <div className="flex-1 overflow-hidden relative">
          <ProfileFlameGraph
            arrow={callersFlamegraphData?.arrow}
            total={callersFlamegraphData.total ?? BigInt(0)}
            filtered={callersFlamegraphData.filtered ?? BigInt(0)}
            profileType={profileSource?.ProfileType()}
            loading={callersFlamegraphData.loading}
            error={callersFlamegraphData.error}
            isHalfScreen={true}
            width={
              callersRef.current != null ? callersRef.current.getBoundingClientRect().width - 25 : 0
            }
            metadataMappingFiles={callersFlamegraphData.metadataMappingFiles}
            metadataLoading={callersFlamegraphData.metadataLoading}
            isInSandwichView={true}
            curPathArrow={curPathArrow}
            setNewCurPathArrow={setCurPathArrow}
            isRenderedAsFlamegraph={true}
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
