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

import React, {memo, useEffect, useState} from 'react';

import GraphTooltipArrow from '../../GraphTooltipArrow';
import GraphTooltipArrowContent from '../../GraphTooltipArrow/Content';
import {DockedGraphTooltip} from '../../GraphTooltipArrow/DockedGraphTooltip';
import {useTooltipContext} from './TooltipContext';

interface MemoizedTooltipProps {
  contextElement: Element | null;
  dockedMetainfo: boolean;
}

export const MemoizedTooltip = memo(function MemoizedTooltip({
  contextElement,
  dockedMetainfo,
}: MemoizedTooltipProps): React.JSX.Element | null {
  const [tooltipRow, setTooltipRow] = useState<number | null>(null);
  const {table, total, totalUnfiltered, profileType, unit, compareAbsolute, tooltipId} =
    useTooltipContext();

  // This component subscribes to tooltip updates through a callback
  // passed to the TooltipProvider, avoiding the need to lift state
  // Fix for tooltip timing issue: Ensure proper cleanup and re-setup when table changes
  useEffect(() => {
    const handleTooltipUpdate = (event: CustomEvent<{row: number | null}>): void => {
      setTooltipRow(event.detail.row);
    };

    const eventName = `icicle-tooltip-update-${tooltipId}`;

    // Delay to ensure all DOM updates and React renders are complete
    // This fixes the race condition in sandwich view
    const timeoutId = setTimeout(() => {
      window.addEventListener(eventName as any, handleTooltipUpdate as any);
    }, 200);

    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener(eventName as any, handleTooltipUpdate as any);
    };
  }, [tooltipId, table]);

  // Re-render when contextElement becomes available (fixes sandwich view timing issue)
  useEffect(() => {
    // Force re-render when contextElement transitions from null to valid element
    // This ensures tooltips work immediately in sandwich view
  }, [contextElement, tooltipId]);

  if (dockedMetainfo) {
    return (
      <DockedGraphTooltip
        table={table}
        row={tooltipRow}
        total={total}
        totalUnfiltered={totalUnfiltered}
        profileType={profileType}
        unit={unit}
        compareAbsolute={compareAbsolute}
      />
    );
  }

  if (tooltipRow === null) {
    return null;
  }

  // Fix for sandwich view tooltip issue: Don't render tooltip if contextElement is null
  // This happens when the SVG ref isn't ready yet during initial sandwich view render
  if (contextElement === null) {
    return null;
  }
  return (
    <GraphTooltipArrow contextElement={contextElement}>
      <GraphTooltipArrowContent
        table={table}
        row={tooltipRow}
        isFixed={false}
        total={total}
        totalUnfiltered={totalUnfiltered}
        profileType={profileType}
        unit={unit}
        compareAbsolute={compareAbsolute}
      />
    </GraphTooltipArrow>
  );
});
