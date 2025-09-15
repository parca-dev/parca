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
    // get the columns to read from
    const mappingColumn = table.getChild(FIELD_MAPPING_FILE);
    const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
    const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
    const depthColumn = table.getChild(FIELD_DEPTH);
    const diffColumn = table.getChild(FIELD_DIFF);
    const filenameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
    const valueOffsetColumn = table.getChild(FIELD_VALUE_OFFSET);
    const tsColumn = table.getChild(FIELD_TIMESTAMP);

    // get the actual values from the columns
    const binaries = useAppSelector(selectBinaries);

    const mappingFile: string | null = arrowToString(mappingColumn?.get(row));
    const functionName: string | null = arrowToString(functionNameColumn?.get(row));
    const cumulative =
      cumulativeColumn?.get(row) !== null ? BigInt(cumulativeColumn?.get(row)) : 0n;
    const diff: bigint | null = diffColumn?.get(row) !== null ? BigInt(diffColumn?.get(row)) : null;
    const filename: string | null = arrowToString(filenameColumn?.get(row));
    const depth: number = depthColumn?.get(row) ?? 0;

    const valueOffset: bigint =
      valueOffsetColumn?.get(row) !== null && valueOffsetColumn?.get(row) !== undefined
        ? BigInt(valueOffsetColumn?.get(row))
        : 0n;

    const colorAttribute =
      colorBy === 'filename' ? filename : colorBy === 'binary' ? mappingFile : null;

    const hoveringName =
      hoveringRow !== undefined ? arrowToString(functionNameColumn?.get(hoveringRow)) : '';
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

    // Hide frames beyond effective depth limit
    if (effectiveDepth !== undefined && depth > effectiveDepth) {
      return <></>;
    }

    const selectionOffset =
      valueOffsetColumn?.get(selectedRow) !== null &&
      valueOffsetColumn?.get(selectedRow) !== undefined
        ? BigInt(valueOffsetColumn?.get(selectedRow))
        : 0n;
    const selectionCumulative =
      cumulativeColumn?.get(selectedRow) !== null ? BigInt(cumulativeColumn?.get(selectedRow)) : 0n;
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
    const tsBounds = boundsFromProfileSource(profileSource);
    const total = cumulativeColumn?.get(selectedRow);
    const totalRatio = cumulative > total ? 1 : Number(cumulative) / Number(total);
    const width: number = isFlameChart
      ? (Number(cumulative) / (Number(tsBounds[1]) - Number(tsBounds[0]))) * totalWidth
      : totalRatio * totalWidth;

    if (width <= 1) {
      return <></>;
    }

    const selectedDepth = depthColumn?.get(selectedRow);
    const styles =
      selectedDepth !== undefined && selectedDepth > depth ? fadedFlameRectStyles : flameRectStyles;

    const onMouseEnter = (): void => {
      setHoveringRow(row);
      window.dispatchEvent(
        new CustomEvent(`flame-tooltip-update-${tooltipId}`, {
          detail: {row},
        })
      );
    };

    const onMouseLeave = (): void => {
      setHoveringRow(undefined);
      window.dispatchEvent(
        new CustomEvent(`flame-tooltip-update-${tooltipId}`, {
          detail: {row: null},
        })
      );
    };

    const handleContextMenu = (e: React.MouseEvent): void => {
      onContextMenu(e, row);
    };

    const ts = tsColumn !== null ? Number(tsColumn.get(row)) : 0;
    const x =
      isFlameChart && tsColumn !== null
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
