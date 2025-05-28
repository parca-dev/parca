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

import React, {ReactNode, useMemo} from 'react';

import {Table} from 'apache-arrow';
import cx from 'classnames';

import {selectBinaries, useAppSelector} from '@parca/store';
import {isSearchMatch, scaleLinear} from '@parca/utilities';

import 'react-contexify/dist/ReactContexify.css';

import {ProfileType} from '@parca/parser';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

import TextWithEllipsis from './TextWithEllipsis';
import {
  FIELD_CHILDREN,
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_MAPPING_FILE,
  FIELD_DEPTH,
  FIELD_PARENT,
  FIELD_VALUE_OFFSET,
} from './index';
import useNodeColor from './useNodeColor';
import {
  CurrentPathFrame,
  arrowToString,
  getCurrentPathFrameData,
  isCurrentPathFrameMatch,
  nodeLabel,
} from './utils';

export const RowHeight = 26;

export interface colorByColors {
  [key: string]: string;
}

export interface IcicleNodeProps {
  height: number;
  totalWidth: number;
  table: Table<any>;
  row: number;
  colors: colorByColors;
  colorBy: string;
  searchString?: string;
  darkMode: boolean;
  compareMode: boolean;
  profileType?: ProfileType;
  isContextMenuOpen: boolean;
  colorForSimilarNodes: string;
  selectedRow: number;
  onClick: () => void;
}

export const icicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
};
export const fadedIcicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
  opacity: '0.5',
};

export const IcicleNode = React.memo(function IcicleNodeNoMemo({
  table,
  row,
  colors,
  colorBy,
  height,
  totalWidth,
  searchString,
  darkMode,
  compareMode,
  profileType,
  isContextMenuOpen,
  colorForSimilarNodes,
  selectedRow,
  onClick,
}: IcicleNodeProps): React.JSX.Element {
  const [highlightSimilarStacksPreference] = useUserPreference<boolean>(
    USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS.key
  );

  // get the columns to read from
  const mappingColumn = table.getChild(FIELD_MAPPING_FILE);
  const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const depthColumn = table.getChild(FIELD_DEPTH);
  const diffColumn = table.getChild(FIELD_DIFF);
  const filenameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
  const valueOffsetColumn = table.getChild(FIELD_VALUE_OFFSET);

  // get the actual values from the columns
  const mappingFile: string | null = arrowToString(mappingColumn?.get(row));
  const functionName: string | null = arrowToString(functionNameColumn?.get(row));
  const cumulative = cumulativeColumn?.get(row) !== null ? BigInt(cumulativeColumn?.get(row)) : 0n;
  const diff: bigint | null = diffColumn?.get(row) !== null ? BigInt(diffColumn?.get(row)) : null;
  const childRows: number[] = Array.from(table.getChild(FIELD_CHILDREN)?.get(row) ?? []);
  const filename: string | null = arrowToString(filenameColumn?.get(row));
  const depth: number = depthColumn?.get(row) ?? 0;
  const valueOffset: bigint = valueOffsetColumn?.get(row) ?? 0n;

  const colorAttribute = colorBy === 'filename' ? filename  : colorBy === 'binary' ? mappingFile : null;

  const colorsMap = colors;

  const hoveringName = functionNameColumn?.get(0); // TODO

  const shouldBeHighlighted = (functionName != null && hoveringName != null && functionName === hoveringName);

  const binaries = useAppSelector(selectBinaries);
  const colorResult = useNodeColor({
    isDarkMode: darkMode,
    compareMode,
    cumulative,
    diff,
    colorsMap,
    colorAttribute,
  });
  const name = useMemo(() => {
    return row === 0 ? 'root' : nodeLabel(table, row, binaries.length > 1);
  }, [table, row, binaries]);
  const selectedDepth = depthColumn?.get(selectedRow);
  const styles = selectedDepth !== undefined && selectedDepth > depth ? fadedIcicleRectStyles : icicleRectStyles;

  // Cumulative can be larger than total when a selection is made. All parents of the selection are likely larger, but we want to only show them as 100% in the graph.
  const total = cumulativeColumn?.get(selectedRow) !== null ? cumulativeColumn?.get(0) : 0;
  const totalRatio = cumulative > total ? 1 : Number(cumulative) / Number(total);
  const width: number =
    (Number(cumulative) / Number(total)) * totalWidth;

  if (width <= 1) {
    return <></>;
  }

  const {isHighlightEnabled = false, isHighlighted = false} = useMemo(() => {
    if (searchString === undefined || searchString === '') {
      return {isHighlightEnabled: false};
    }
    return {isHighlightEnabled: true, isHighlighted: isSearchMatch(searchString, name)};
  }, [searchString, name]);

  if (width <= 1) {
    return <>{null}</>;
  }


  const onMouseEnter = (e: React.MouseEvent): void => {
    if (isContextMenuOpen) return;
    window.dispatchEvent(new CustomEvent('icicle-tooltip-update', {
      detail: { row }
    }));
  };

  const onMouseLeave = (): void => {
    if (isContextMenuOpen) return;
    window.dispatchEvent(new CustomEvent('icicle-tooltip-update', {
      detail: { row: null }
    }));
  };

  const selectionOffset = valueOffsetColumn?.get(selectedRow) ?? 0n;
  //const x = selectedDepth > depth ? 0 : (Number(valueOffset) - Number(selectionOffset)) * totalRatio;
  const x = (Number(valueOffset) / Number(total)) * totalWidth;
  const y = depth * (height);

  return (
    <>
      <g
        transform={`translate(${x + 1}, ${y + 1})`}
        style={styles}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onClick={onClick}
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
              : 'stroke-white dark:stroke-gray-700',
            {
              'opacity-50': isHighlightEnabled && !isHighlighted,
            }
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
});
