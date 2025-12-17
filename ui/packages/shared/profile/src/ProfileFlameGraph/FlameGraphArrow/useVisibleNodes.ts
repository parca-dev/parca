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

import {useMemo, useRef} from 'react';

import {Table} from 'apache-arrow';

import {RowHeight} from './FlameGraphNodes';
import {FIELD_CUMULATIVE, FIELD_DEPTH, FIELD_VALUE_OFFSET} from './index';
import {ViewportState} from './useScrollViewport';
import {getMaxDepth} from './utils';

/**
 * This function groups rows by their depth level.
 * Instead of scanning all rows to find depth matches, we pre-compute
 * buckets so viewport rendering only examines depth ranges that are relevant.
 */
const useDepthBuckets = <TRow extends Record<string, any>>(
  table: Table<TRow> | undefined
): number[][] => {
  return useMemo(() => {
    if (table === undefined) return [];

    const depthColumn = table.getChild(FIELD_DEPTH);
    if (depthColumn === null) return [];

    // Find max depth
    const maxDepth = getMaxDepth(depthColumn);

    // Create buckets for each depth level
    const buckets: number[][] = Array.from({length: maxDepth + 1}, () => []);

    // Populate buckets with row indices
    for (let row = 0; row < table.numRows; row++) {
      const depth = depthColumn.get(row) ?? 0;
      buckets[depth].push(row);
    }

    return buckets;
  }, [table]);
};

export interface UseVisibleNodesParams {
  table: Table<any>;
  viewport: ViewportState;
  total: bigint;
  width: number;
  selectedRow: number;
  effectiveDepth: number;
}

/**
 * useVisibleNodes returns row indices visible in the current viewport through multi-stage culling.
 * Combines depth buckets, horizontal bounds checking, and size filtering to
 * minimize rendered nodes from potentially 100K+ rows to ~hundreds.
 *
 * We use depth buckets to only iterate through the rows that are visible in the viewport vertically.
 * After that we use horizontal bounds checking to only iterate through the rows that are visible in the viewport horizontally.
 * Finally we use size filtering to only iterate through the rows that are visible in the viewport by size.
 *
 * Critical for maintaining 60fps performance on large flamegraphs where
 * rendering all nodes would freeze the browser.
 */
export const useVisibleNodes = ({
  table,
  viewport,
  total,
  width,
  selectedRow,
  effectiveDepth,
}: UseVisibleNodesParams): number[] => {
  const depthBuckets = useDepthBuckets(table);
  const lastResultRef = useRef<{
    key: string;
    result: number[];
  }>({key: '', result: []});

  const renderedRangeRef = useRef<{minDepth: number; maxDepth: number; table: Table<any> | null}>({
    minDepth: Infinity,
    maxDepth: -Infinity,
    table: null,
  });

  return useMemo(() => {
    // This happens when the continer is scrolled off screen, in this case we return all previously rendered nodes
    // to avoid trimming the rendered nodes to zero which would cause jank when scrolling back into view
    if (viewport.containerHeight === 0 && lastResultRef.current.result.length > 0) {
      return lastResultRef.current.result;
    }

    // Create a stable key for memoization to prevent unnecessary recalculations
    const memoKey = `${viewport.scrollTop}-${
      viewport.containerHeight
    }-${selectedRow}-${effectiveDepth}-${width}-${Number(total)}-${table.numRows}`;

    // Return cached result if viewport hasn't meaningfully changed
    if (lastResultRef.current.key === memoKey) {
      return lastResultRef.current.result;
    }

    if (table === null) return [];

    const visibleRows: number[] = [];
    const {scrollTop, containerHeight} = viewport;

    // Viewport Culling Algorithm:
    // 1. Calculate visible depth range based on scroll position and container height
    // 2. Add 5-row buffer above/below for smooth scrolling experience
    // Note: We never shrink the rendered range to avoid back and forth node removals (and in turn additions when scrolled down again) to the dom.

    const BUFFER = 15; // Buffer for smoother scrolling

    const visibleStartDepth = Math.max(0, Math.floor(scrollTop / RowHeight) - BUFFER);
    const visibleDepths = Math.ceil(containerHeight / RowHeight);
    const visibleEndDepth = Math.min(effectiveDepth, visibleStartDepth + visibleDepths + BUFFER);

    // Reset range if table changed (new data loaded) as this is new data
    if (renderedRangeRef.current.table !== table) {
      renderedRangeRef.current = {
        minDepth: Infinity,
        maxDepth: -Infinity,
        table: table,
      };
    }

    // Expand the rendered range (never shrink when scrolling up/down)
    renderedRangeRef.current.minDepth = Math.min(
      renderedRangeRef.current.minDepth,
      visibleStartDepth
    );
    renderedRangeRef.current.maxDepth = Math.max(
      renderedRangeRef.current.maxDepth,
      visibleEndDepth
    );

    const startDepth = renderedRangeRef.current.minDepth;
    const endDepth = renderedRangeRef.current.maxDepth;

    const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
    const valueOffsetColumn = table.getChild(FIELD_VALUE_OFFSET);

    const selectionOffset =
      valueOffsetColumn?.get(selectedRow) !== null &&
      valueOffsetColumn?.get(selectedRow) !== undefined
        ? BigInt(valueOffsetColumn?.get(selectedRow))
        : 0n;
    const selectionCumulative =
      cumulativeColumn?.get(selectedRow) != null ? BigInt(cumulativeColumn?.get(selectedRow)) : 0n;

    const totalNumber = Number(total);
    const selectionOffsetNumber = Number(selectionOffset);
    const selectionCumulativeNumber = Number(selectionCumulative);

    // Iterate only through visible depth range instead of all rows
    for (let depth = startDepth; depth <= endDepth && depth < depthBuckets.length; depth++) {
      // Skip if depth is beyond effective depth limit
      if (effectiveDepth !== undefined && depth > effectiveDepth) {
        continue;
      }

      const rowsAtDepth = depthBuckets[depth];

      for (const row of rowsAtDepth) {
        const cumulative =
          cumulativeColumn?.get(row) != null ? Number(cumulativeColumn?.get(row)) : 0;

        const valueOffset =
          valueOffsetColumn?.get(row) !== null && valueOffsetColumn?.get(row) !== undefined
            ? Number(valueOffsetColumn?.get(row))
            : 0;

        // Horizontal culling: Skip nodes outside selection bounds
        if (
          valueOffset + cumulative <= selectionOffsetNumber ||
          valueOffset >= selectionOffsetNumber + selectionCumulativeNumber
        ) {
          continue;
        }

        // Size culling: Skip nodes too small to be visible (< 1px width)
        const computedWidth = (cumulative / totalNumber) * width;
        if (computedWidth <= 1) {
          continue;
        }

        visibleRows.push(row);
      }
    }

    // Cache the result with the current key
    lastResultRef.current = {key: memoKey, result: visibleRows};

    return visibleRows;
  }, [depthBuckets, viewport, total, width, selectedRow, effectiveDepth, table]);
};
