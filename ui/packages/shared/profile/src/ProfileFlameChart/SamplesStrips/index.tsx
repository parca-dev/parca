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

import {useMemo, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import * as d3 from 'd3';
import isEqual from 'fast-deep-equal';
import {useIntersectionObserver} from 'usehooks-ts';

import {LabelSet} from '@parca/client';
import {Button} from '@parca/components';

import {TimelineGuide} from '../../TimelineGuide';
import {NumberDuo} from '../../utils';
import {DataPoint, SamplesGraph} from './SamplesGraph';
import {createLabelSetComparator, labelSetToString} from './labelSetUtils';

export type {DataPoint} from './SamplesGraph';

interface DragState {
  stripIndex: number;
  startX: number;
  currentX: number;
}

interface Props {
  cpus: LabelSet[];
  data: DataPoint[][];
  selectedTimeframe?: {
    labels: LabelSet;
    bounds: NumberDuo;
  };
  onSelectedTimeframe: (labels: LabelSet, bounds: NumberDuo | undefined) => void;
  width?: number;
  bounds: NumberDuo;
  stepMs: number;
}

const STRIP_HEIGHT = 24;
const LABEL_ROW_HEIGHT = 16; // text-xs label row above each strip
const GAP = 4; // gap-1 between flex children
const MAX_VISIBLE_STRIPS = 20;

const getTimelineGuideHeight = (cpusCount: number, collapsedCount: number): number => {
  const expandedCount = cpusCount - collapsedCount;
  // Each expanded strip: label row + graph height
  // Each collapsed strip: min-h-5 (20px)
  // Gaps between strips (gap-1 = 4px)
  const expandedTotal = expandedCount * (LABEL_ROW_HEIGHT + STRIP_HEIGHT);
  const collapsedTotal = collapsedCount * 20; // min-h-5
  const gaps = cpusCount * GAP + 20; // timeline header
  return expandedTotal + collapsedTotal + gaps;
};

const stickyPx = 0;

const SamplesGraphContainer = ({
  isSelected,
  isCollapsed,
  label: labelStr,
  width,
  onToggleCollapse,
  data,
  selectionBounds,
  setSelectionBounds,
  color,
  stepMs,
  onDragStart,
  dragState,
  stripIndex,
  isAnyDragActive,
  timeBounds,
}: {
  isSelected: boolean;
  isCollapsed: boolean;
  label: string;
  width: number | undefined;
  onToggleCollapse: () => void;
  data: DataPoint[];
  selectionBounds: NumberDuo | undefined;
  setSelectionBounds: (bounds: NumberDuo | undefined) => void;
  color: (label: string) => string;
  stepMs: number;
  onDragStart: (stripIndex: number, startX: number) => void;
  dragState: DragState | undefined;
  stripIndex: number;
  isAnyDragActive: boolean;
  timeBounds: NumberDuo;
}): JSX.Element => {
  const {isIntersecting, ref} = useIntersectionObserver({
    rootMargin: `${stickyPx}px 0px 0px 0px`,
  });

  const isSticky = useMemo(() => {
    return isSelected && isIntersecting;
  }, [isSelected, isIntersecting]);

  return (
    <div
      className={cx('min-h-5', {
        relative: !isSelected,
        'sticky z-30 bg-white dark:bg-black bg-opacity-75': isSelected,
        '!bg-opacity-100': isSticky,
      })}
      style={{width: width ?? 1468, top: isSelected ? stickyPx : undefined}}
      key={labelStr}
      ref={ref}
    >
      <div
        className="text-xs flex gap-[2px] items-center px-1 cursor-pointer text-gray-600 dark:text-gray-400"
        onClick={onToggleCollapse}
      >
        <Icon icon={isCollapsed ? 'bxs:right-arrow' : 'bxs:down-arrow'} className="shrink-0" />
        {labelStr}
      </div>
      {!isCollapsed ? (
        <SamplesGraph
          data={data}
          height={STRIP_HEIGHT}
          width={width ?? 1468}
          fill={color(labelStr)}
          selectionBounds={selectionBounds}
          setSelectionBounds={setSelectionBounds}
          stepMs={stepMs}
          onDragStart={(startX: number) => onDragStart(stripIndex, startX)}
          dragState={dragState?.stripIndex === stripIndex ? dragState : undefined}
          isAnyDragActive={isAnyDragActive}
          timeBounds={timeBounds}
        />
      ) : null}
    </div>
  );
};

export const SamplesStrip = ({
  cpus,
  data,
  selectedTimeframe,
  onSelectedTimeframe,
  width,
  bounds,
  stepMs,
}: Props): JSX.Element => {
  const [collapsedLabels, setCollapsedLabels] = useState<Set<string>>(new Set());
  const [showAll, setShowAll] = useState(false);
  const [dragState, setDragState] = useState<DragState | undefined>(undefined);
  const containerRef = useRef<HTMLDivElement>(null);

  const isDragging = dragState !== undefined;

  const {compare, keyOrder} = useMemo(() => createLabelSetComparator(cpus), [cpus]);

  const sortedItems = useMemo(() => {
    const items = cpus.map((cpu, i) => ({
      cpu,
      data: data[i],
      label: labelSetToString(cpu, keyOrder),
    }));
    return items.sort((a, b) => compare(a.cpu, b.cpu));
  }, [cpus, data, compare, keyOrder]);

  const hasMore = useMemo(() => sortedItems.length > MAX_VISIBLE_STRIPS, [sortedItems]);
  const visibleItems = useMemo(
    () => (showAll || !hasMore ? sortedItems : sortedItems.slice(0, MAX_VISIBLE_STRIPS)),
    [sortedItems, showAll, hasMore]
  );

  // Deterministic color: hash the label string so the same label always gets the same color
  // regardless of render order.
  const color = useMemo(() => {
    const palette = d3.schemeObservable10;
    const hashStr = (s: string): number => {
      let h = 0;
      for (let i = 0; i < s.length; i++) {
        h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
      }
      return Math.abs(h);
    };
    return (label: string): string => palette[hashStr(label) % palette.length];
  }, []);

  const handleDragStart = (stripIndex: number, startX: number): void => {
    setDragState({stripIndex, startX, currentX: startX});
  };

  const handleMouseMove = (e: React.MouseEvent): void => {
    if (dragState === undefined || containerRef.current === null) return;

    const rect = containerRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    // Clamp to container bounds
    const clampedX = Math.max(0, Math.min(x, width ?? rect.width));
    setDragState({...dragState, currentX: clampedX});
  };

  const handleMouseUp = (e: React.MouseEvent): void => {
    if (dragState === undefined || containerRef.current === null) return;

    const rect = containerRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const clampedX = Math.max(0, Math.min(x, width ?? rect.width));

    const {stripIndex, startX} = dragState;
    if (startX !== clampedX) {
      const start = Math.min(startX, clampedX);
      const end = Math.max(startX, clampedX);
      // Convert pixel positions to timestamps
      const innerWidth = width ?? rect.width;
      const startTs = bounds[0] + (start / innerWidth) * (bounds[1] - bounds[0]);
      const endTs = bounds[0] + (end / innerWidth) * (bounds[1] - bounds[0]);
      onSelectedTimeframe(visibleItems[stripIndex].cpu, [startTs, endTs]);
    }

    setDragState(undefined);
  };

  const handleMouseLeave = (): void => {
    setDragState(undefined);
  };

  if (data.length === 0) {
    return (
      <span className="flex justify-center my-10">
        There is no data matching your filter criteria, please try changing the filter.
      </span>
    );
  }

  return (
    <div
      ref={containerRef}
      className={cx('flex flex-col gap-1 relative my-0', {'cursor-ew-resize': isDragging})}
      style={{width: width ?? '100%'}}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseLeave}
    >
      <TimelineGuide
        bounds={[BigInt(0), BigInt(bounds[1] - bounds[0])]}
        width={width ?? 1468}
        height={getTimelineGuideHeight(
          visibleItems.length,
          [...collapsedLabels].filter(l => visibleItems.some(item => item.label === l)).length
        )}
        margin={1}
      />
      {visibleItems.map((item, i) => {
        const isCollapsed = collapsedLabels.has(item.label);
        const isSelected = isEqual(item.cpu, selectedTimeframe?.labels);

        return (
          <SamplesGraphContainer
            isSelected={isSelected}
            isCollapsed={isCollapsed}
            label={item.label}
            width={width}
            data={item.data}
            onToggleCollapse={() => {
              const newCollapsedLabels = new Set(collapsedLabels);
              if (collapsedLabels.has(item.label)) {
                newCollapsedLabels.delete(item.label);
              } else {
                newCollapsedLabels.add(item.label);
              }
              setCollapsedLabels(newCollapsedLabels);
            }}
            selectionBounds={isSelected ? selectedTimeframe?.bounds : undefined}
            setSelectionBounds={newBounds => {
              onSelectedTimeframe(item.cpu, newBounds);
            }}
            color={color}
            stepMs={stepMs}
            onDragStart={handleDragStart}
            dragState={dragState}
            stripIndex={i}
            isAnyDragActive={isDragging}
            timeBounds={bounds}
            key={item.label}
          />
        );
      })}
      {hasMore && !showAll && (
        <Button variant="secondary" onClick={() => setShowAll(true)} className="w-fit mx-auto mt-2">
          Show all {sortedItems.length} rows
        </Button>
      )}
    </div>
  );
};
