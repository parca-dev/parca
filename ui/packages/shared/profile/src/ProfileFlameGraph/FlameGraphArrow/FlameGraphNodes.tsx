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

import React, {useMemo, useCallback} from 'react';

import {Table} from 'apache-arrow';
import cx from 'classnames';

import {selectBinaries, useAppSelector} from '@parca/store';

import 'react-contexify/dist/ReactContexify.css';

import {ProfileSource} from '../../ProfileSource';
import TextWithEllipsis from './TextWithEllipsis';
import {
  FIELD_CUMULATIVE,
  FIELD_DEPTH,
  FIELD_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_MAPPING_FILE,
  FIELD_TIMESTAMP,
  FIELD_VALUE_OFFSET,
} from './index';
import useNodeColor from './useNodeColor';
import {arrowToString, boundsFromProfileSource, nodeLabel} from './utils';

export const RowHeight = 26;

export interface colorByColors {
  [key: string]: string;
}

export interface FlameNodeProps {
  height: number;
  totalWidth: number;
  table: Table<any>;
  row: number;
  colors: colorByColors;
  colorBy: string;
  darkMode: boolean;
  compareMode: boolean;
  onContextMenu: (e: React.MouseEvent, row: number) => void;
  colorForSimilarNodes: string;
  selectedRow: number;
  onClick: () => void;
  isFlameChart: boolean;
  profileSource: ProfileSource;
  isRenderedAsFlamegraph?: boolean;
  isInSandwichView?: boolean;
  maxDepth?: number;
  effectiveDepth?: number;
  tooltipId?: string;

  // Hovering row must only ever be used for highlighting similar nodes, otherwise it will cause performance issues as it causes the full flamegraph to get rerendered every time the hovering row changes.
  hoveringRow?: number;
  setHoveringRow: (row: number | undefined) => void;
}

export const flameRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
};
export const fadedFlameRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
  opacity: '0.5',
};

export const FlameNode = React.memo(
  function FlameNodeNoMemo({
    table,
    row,
    colors,
    colorBy,
    height,
    totalWidth,
    darkMode,
    compareMode,
    colorForSimilarNodes,
    selectedRow,
    onClick,
    onContextMenu,
    hoveringRow,
    setHoveringRow,
    isFlameChart,
    profileSource,
    isRenderedAsFlamegraph = false,
    isInSandwichView = false,
    maxDepth = 0,
    effectiveDepth,
    tooltipId = 'default',
  }: FlameNodeProps): React.JSX.Element {
    // Memoize column references - only changes when table changes
    const columns = useMemo(() => ({
      mapping: table.getChild(FIELD_MAPPING_FILE),
      functionName: table.getChild(FIELD_FUNCTION_NAME),
      cumulative: table.getChild(FIELD_CUMULATIVE),
      depth: table.getChild(FIELD_DEPTH),
      diff: table.getChild(FIELD_DIFF),
      filename: table.getChild(FIELD_FUNCTION_FILE_NAME),
      valueOffset: table.getChild(FIELD_VALUE_OFFSET),
      ts: table.getChild(FIELD_TIMESTAMP),
    }), [table]);

    // get the actual values from the columns
    const binaries = useAppSelector(selectBinaries);

    // Memoize row data extraction - only changes when table or row changes
    const rowData = useMemo(() => {
      const mappingFile: string | null = arrowToString(columns.mapping?.get(row));
      const functionName: string | null = arrowToString(columns.functionName?.get(row));
      const cumulative = columns.cumulative?.get(row) != null ? BigInt(columns.cumulative?.get(row)) : 0n;
      const diff: bigint | null = columns.diff?.get(row) != null ? BigInt(columns.diff?.get(row)) : null;
      const filename: string | null = arrowToString(columns.filename?.get(row));
      const depth: number = columns.depth?.get(row) ?? 0;
      const valueOffset: bigint =
        columns.valueOffset?.get(row) !== null && columns.valueOffset?.get(row) !== undefined
          ? BigInt(columns.valueOffset?.get(row))
          : 0n;

      return { mappingFile, functionName, cumulative, diff, filename, depth, valueOffset };
    }, [columns, row]);

    const { mappingFile, functionName, cumulative, diff, filename, depth, valueOffset } = rowData;

    const colorAttribute =
      colorBy === 'filename' ? filename : colorBy === 'binary' ? mappingFile : null;

    // Memoize hovering name lookup
    const hoveringName = useMemo(() => {
      return hoveringRow !== undefined ? arrowToString(columns.functionName?.get(hoveringRow)) : '';
    }, [columns.functionName, hoveringRow]);

    const shouldBeHighlighted =
      functionName != null && hoveringName != null && functionName === hoveringName;

    const colorResult = useNodeColor({
      isDarkMode: darkMode,
      compareMode,
      cumulative,
      diff,
      colorsMap: colors,
      colorAttribute,
    });

    const name = useMemo(() => {
      return row === 0 ? 'root' : nodeLabel(table, row, binaries.length > 1);
    }, [table, row, binaries]);

    // Memoize selection data - only changes when selectedRow changes
    const selectionData = useMemo(() => {
      const selectionOffset =
        columns.valueOffset?.get(selectedRow) !== null &&
        columns.valueOffset?.get(selectedRow) !== undefined
          ? BigInt(columns.valueOffset?.get(selectedRow))
          : 0n;
      const selectionCumulative =
        columns.cumulative?.get(selectedRow) !== null ? BigInt(columns.cumulative?.get(selectedRow)) : 0n;
      const selectedDepth = columns.depth?.get(selectedRow);
      const total = columns.cumulative?.get(selectedRow);
      return { selectionOffset, selectionCumulative, selectedDepth, total };
    }, [columns, selectedRow]);

    const { selectionOffset, selectionCumulative, selectedDepth, total } = selectionData;

    // Memoize tsBounds - only changes when profileSource changes
    const tsBounds = useMemo(() => boundsFromProfileSource(profileSource), [profileSource]);

    // Memoize event handlers
    const onMouseEnter = useCallback((): void => {
      setHoveringRow(row);
      window.dispatchEvent(
        new CustomEvent(`flame-tooltip-update-${tooltipId}`, {
          detail: {row},
        })
      );
    }, [setHoveringRow, row, tooltipId]);

    const onMouseLeave = useCallback((): void => {
      setHoveringRow(undefined);
      window.dispatchEvent(
        new CustomEvent(`flame-tooltip-update-${tooltipId}`, {
          detail: {row: null},
        })
      );
    }, [setHoveringRow, tooltipId]);

    const handleContextMenu = useCallback((e: React.MouseEvent): void => {
      onContextMenu(e, row);
    }, [onContextMenu, row]);

    // Early returns - all hooks must be called before this point
    // Hide frames beyond effective depth limit
    if (effectiveDepth !== undefined && depth > effectiveDepth) {
      return <></>;
    }

    if (
      valueOffset + cumulative <= selectionOffset ||
      valueOffset >= selectionOffset + selectionCumulative
    ) {
      // If the end of the node is before the selection offset or the start of the node is after the selection offset + totalWidth, we don't render it.
      return <></>;
    }

    if (row === 0 && (isFlameChart || isInSandwichView)) {
      // The root node is not rendered in the flame chart or sandwich view, so we return null.
      return <></>;
    }

    // Cumulative can be larger than total when a selection is made. All parents of the selection are likely larger, but we want to only show them as 100% in the graph.
    const totalRatio = cumulative > total ? 1 : Number(cumulative) / Number(total);
    const width: number = isFlameChart
      ? (Number(cumulative) / (Number(tsBounds[1]) - Number(tsBounds[0]))) * totalWidth
      : totalRatio * totalWidth;

    if (width <= 1) {
      return <></>;
    }

    const styles =
      selectedDepth !== undefined && selectedDepth > depth ? fadedFlameRectStyles : flameRectStyles;

    const ts = columns.ts !== null ? Number(columns.ts.get(row)) : 0;
    const x =
      isFlameChart && columns.ts !== null
        ? ((ts - Number(tsBounds[0])) / (Number(tsBounds[1]) - Number(tsBounds[0]))) * totalWidth
        : selectedDepth > depth
        ? 0
        : ((Number(valueOffset) - Number(selectionOffset)) / Number(total)) * totalWidth;

    const calculateY = (
      isRenderedAsFlamegraph: boolean,
      isInSandwichView: boolean,
      isFlameChart: boolean,
      maxDepth: number,
      depth: number,
      height: number
    ): number => {
      if (isRenderedAsFlamegraph) {
        return (maxDepth - depth) * height; // Flamegraph is inverted
      }

      if (isFlameChart || isInSandwichView) {
        return (depth - 1) * height;
      }

      return depth * height;
    };

    const y = calculateY(
      isRenderedAsFlamegraph,
      isInSandwichView,
      isFlameChart,
      effectiveDepth ?? maxDepth,
      depth,
      height
    );

    return (
      <>
        <g
          id={row === 0 ? 'root-span' : undefined}
          transform={`translate(${x + 1}, ${y + 1})`}
          style={styles}
          onMouseEnter={onMouseEnter}
          onMouseLeave={onMouseLeave}
          onClick={onClick}
          onContextMenu={handleContextMenu}
        >
          <rect
            x={0}
            y={0}
            width={width}
            height={height}
            style={{
              fill: colorResult,
            }}
            className={cx(
              shouldBeHighlighted
                ? `${colorForSimilarNodes} stroke-[3] [stroke-dasharray:6,4] [stroke-linecap:round] [stroke-linejoin:round] h-6`
                : 'stroke-white dark:stroke-gray-700'
            )}
          />
          {width > 5 && (
            <svg width={width - 5} height={height}>
              <TextWithEllipsis
                text={name}
                x={5}
                y={15}
                width={width - 10} // Subtract padding from available width
              />
            </svg>
          )}
        </g>
      </>
    );
  },
  (prevProps, nextProps) => {
    // Only re-render if the relevant props have changed
    return (
      prevProps.row === nextProps.row &&
      prevProps.selectedRow === nextProps.selectedRow &&
      prevProps.hoveringRow === nextProps.hoveringRow &&
      prevProps.totalWidth === nextProps.totalWidth &&
      prevProps.height === nextProps.height &&
      prevProps.effectiveDepth === nextProps.effectiveDepth &&
      prevProps.colorBy === nextProps.colorBy &&
      prevProps.colors === nextProps.colors
    );
  }
);
