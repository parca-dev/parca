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

import TextWithEllipsis from './TextWithEllipsis';
import {
  FIELD_CHILDREN,
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_MAPPING_FILE,
} from './index';
import useNodeColor from './useNodeColor';
import {arrowToString, nodeLabel} from './utils';

export const RowHeight = 26;

interface IcicleGraphNodesProps {
  table: Table<any>;
  row: number;
  colors: colorByColors;
  colorBy: string;
  childRows: number[];
  x: number;
  y: number;
  total: bigint;
  totalWidth: number;
  level: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringRow: (row: number | null) => void;
  setHoveringLevel: (level: number | null) => void;
  path: string[];
  xScale: (value: bigint) => number;
  searchString?: string;
  sortBy: string;
  darkMode: boolean;
  compareMode: boolean;
  profileType?: ProfileType;
  isContextMenuOpen: boolean;
  hoveringName: string | null;
  setHoveringName: (name: string | null) => void;
  hoveringRow: number | null;
  colorForSimilarNodes: string;
  highlightSimilarStacksPreference: boolean;
}

export const IcicleGraphNodes = React.memo(function IcicleGraphNodesNoMemo({
  table,
  childRows,
  colors,
  colorBy,
  x,
  y,
  xScale,
  total,
  totalWidth,
  level,
  path,
  setCurPath,
  setHoveringRow,
  setHoveringLevel,
  curPath,
  sortBy,
  searchString,
  darkMode,
  compareMode,
  profileType,
  isContextMenuOpen,
  hoveringName,
  setHoveringName,
  hoveringRow,
  colorForSimilarNodes,
  highlightSimilarStacksPreference,
}: IcicleGraphNodesProps): React.JSX.Element {
  const cumulatives = table.getChild(FIELD_CUMULATIVE);

  if (childRows === undefined || childRows.length === 0) {
    return <></>;
  }

  childRows =
    curPath.length === 0
      ? childRows
      : childRows.filter(c => nodeLabel(table, c, level, false) === curPath[0]);

  let childrenCumulative = BigInt(0);
  const childrenElements: ReactNode[] = [];
  childRows.forEach((child, i) => {
    const xStart = Math.floor(xScale(BigInt(childrenCumulative)));
    const c = BigInt(cumulatives?.get(child));
    childrenCumulative += c;

    childrenElements.push(
      <IcicleNode
        key={`node-${level}-${i}`}
        table={table}
        row={child}
        colors={colors}
        colorBy={colorBy}
        x={xStart}
        y={0}
        totalWidth={totalWidth}
        height={RowHeight}
        path={path}
        setCurPath={setCurPath}
        setHoveringRow={setHoveringRow}
        setHoveringLevel={setHoveringLevel}
        level={level}
        curPath={curPath}
        total={total}
        xScale={xScale}
        sortBy={sortBy}
        searchString={searchString}
        darkMode={darkMode}
        compareMode={compareMode}
        profileType={profileType}
        isContextMenuOpen={isContextMenuOpen}
        hoveringName={hoveringName}
        setHoveringName={setHoveringName}
        hoveringRow={hoveringRow}
        colorForSimilarNodes={colorForSimilarNodes}
        highlightSimilarStacksPreference={highlightSimilarStacksPreference}
      />
    );
  });

  return <g transform={`translate(${x}, ${y})`}>{childrenElements}</g>;
});

export interface colorByColors {
  [key: string]: string;
}

export interface IcicleNodeProps {
  x: number;
  y: number;
  height: number;
  totalWidth: number;
  curPath: string[];
  level: number;
  table: Table<any>;
  row: number;
  colors: colorByColors;
  colorBy: string;
  path: string[];
  total: bigint;
  setCurPath: (path: string[]) => void;
  setHoveringRow: (row: number | null) => void;
  setHoveringLevel: (level: number | null) => void;
  xScale: (value: bigint) => number;
  isRoot?: boolean;
  searchString?: string;
  sortBy: string;
  darkMode: boolean;
  compareMode: boolean;
  profileType?: ProfileType;
  isContextMenuOpen: boolean;
  hoveringName: string | null;
  setHoveringName: (name: string | null) => void;
  hoveringRow: number | null;
  colorForSimilarNodes: string;
  highlightSimilarStacksPreference: boolean;
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
  x,
  y,
  height,
  setCurPath,
  curPath,
  level,
  path,
  total,
  totalWidth,
  xScale,
  isRoot = false,
  searchString,
  setHoveringRow,
  setHoveringLevel,
  sortBy,
  darkMode,
  compareMode,
  profileType,
  isContextMenuOpen,
  hoveringName,
  setHoveringName,
  hoveringRow,
  colorForSimilarNodes,
  highlightSimilarStacksPreference,
}: IcicleNodeProps): React.JSX.Element {
  // get the columns to read from
  const mappingColumn = table.getChild(FIELD_MAPPING_FILE);
  const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const diffColumn = table.getChild(FIELD_DIFF);
  const filenameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
  // get the actual values from the columns
  const mappingFile: string | null = arrowToString(mappingColumn?.get(row));
  const functionName: string | null = arrowToString(functionNameColumn?.get(row));
  const cumulative = cumulativeColumn?.get(row) !== null ? BigInt(cumulativeColumn?.get(row)) : 0n;
  const diff: bigint | null = diffColumn?.get(row) !== null ? BigInt(diffColumn?.get(row)) : null;
  const childRows: number[] = Array.from(table.getChild(FIELD_CHILDREN)?.get(row) ?? []);
  const filename: string | null = arrowToString(filenameColumn?.get(row));

  const colorAttribute: string | null = useMemo(() => {
    let attr: string | null | undefined;

    if (colorBy === 'filename') {
      attr = filename;
    } else if (colorBy === 'binary') {
      attr = mappingFile;
    }

    return attr ?? null; // Provide a default value of null if attr is undefined
  }, [colorBy, filename, mappingFile]);

  const colorsMap = colors;

  const highlightedNodes = useMemo(() => {
    if (!highlightSimilarStacksPreference) {
      return null;
    }

    if (functionName != null && functionName === hoveringName) {
      return {functionName, row: hoveringRow};
    }
    return null; // Nothing to highlight
  }, [functionName, hoveringName, hoveringRow, highlightSimilarStacksPreference]);

  const shouldBeHighlightedIfSimilarStacks = useMemo(() => {
    return highlightedNodes !== null && row !== highlightedNodes.row;
  }, [row, highlightedNodes]);

  // TODO: Maybe it's better to pass down the sorter function as prop instead of figuring this out here.
  switch (sortBy) {
    case FIELD_FUNCTION_NAME:
      childRows.sort((a, b) => {
        // TODO: Support fallthrough to comparing addresses or something
        const afn: string | null = arrowToString(functionNameColumn?.get(a));
        const bfn: string | null = arrowToString(functionNameColumn?.get(b));
        if (afn !== null && bfn !== null) {
          return afn.localeCompare(bfn);
        }
        if (afn === null && bfn !== null) {
          return -1;
        }
        if (afn !== null && bfn === null) {
          return 1;
        }
        // both are null
        return 0;
      });
      break;
    case FIELD_CUMULATIVE:
      childRows.sort((a, b) => {
        const aCumulative: bigint = cumulativeColumn?.get(a);
        const bCumulative: bigint = cumulativeColumn?.get(b);
        return Number(bCumulative - aCumulative);
      });
      break;
    case FIELD_DIFF:
      childRows.sort((a, b) => {
        let aRatio: number | null = null;
        const aDiff: bigint | null =
          diffColumn?.get(a) !== null ? BigInt(diffColumn?.get(a)) : null;
        if (aDiff !== null) {
          const cumulative: bigint =
            cumulativeColumn?.get(a) !== null ? BigInt(cumulativeColumn?.get(a)) : 0n;
          const prev: bigint = cumulative - aDiff;
          aRatio = Number(aDiff) / Number(prev);
        }
        let bRatio: number | null = null;
        const bDiff: bigint | null =
          diffColumn?.get(b) !== null ? BigInt(diffColumn?.get(b)) : null;
        if (bDiff !== null) {
          const cumulative: bigint =
            cumulativeColumn?.get(b) !== null ? BigInt(cumulativeColumn?.get(b)) : 0n;
          const prev: bigint = cumulative - bDiff;
          bRatio = Number(bDiff) / Number(prev);
        }

        if (aRatio !== null && bRatio !== null) {
          return bRatio - aRatio;
        }
        if (aRatio === null && bRatio !== null) {
          return -1;
        }
        if (aRatio !== null && bRatio === null) {
          return 1;
        }
        // both are null
        return 0;
      });
      break;
  }

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
    return isRoot ? 'root' : nodeLabel(table, row, level, binaries.length > 1);
  }, [table, row, level, isRoot, binaries]);
  const nextPath = path.concat([name]);
  const isFaded = curPath.length > 0 && name !== curPath[curPath.length - 1];
  const styles = isFaded ? fadedIcicleRectStyles : icicleRectStyles;
  const nextLevel = level + 1;
  const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
  const newXScale =
    nextCurPath.length === 0 && curPath.length === 1
      ? scaleLinear([0n, BigInt(cumulative)], [0, totalWidth])
      : xScale;

  const width: number =
    nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
      ? totalWidth
      : xScale(BigInt(cumulative));

  const {isHighlightEnabled = false, isHighlighted = false} = useMemo(() => {
    if (searchString === undefined || searchString === '') {
      return {isHighlightEnabled: false};
    }
    return {isHighlightEnabled: true, isHighlighted: isSearchMatch(searchString, name)};
  }, [searchString, name]);

  if (width <= 1) {
    return <>{null}</>;
  }

  const onMouseEnter = (): void => {
    if (isContextMenuOpen) return;
    setHoveringRow(row);
    setHoveringLevel(level);
    setHoveringName(name);
  };

  const onMouseLeave = (): void => {
    if (isContextMenuOpen) return;
    setHoveringRow(null);
    setHoveringLevel(null);
    setHoveringName(null);
  };

  return (
    <>
      <g
        transform={`translate(${x + 1}, ${y + 1})`}
        style={styles}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        onClick={() => {
          setCurPath(nextPath);
        }}
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
            shouldBeHighlightedIfSimilarStacks
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
      {childRows.length > 0 && (
        <IcicleGraphNodes
          table={table}
          row={row}
          colors={colors}
          colorBy={colorBy}
          childRows={childRows}
          x={x}
          y={RowHeight}
          xScale={newXScale}
          total={total}
          totalWidth={totalWidth}
          level={nextLevel}
          path={nextPath}
          curPath={nextCurPath}
          setCurPath={setCurPath}
          setHoveringRow={setHoveringRow}
          setHoveringLevel={setHoveringLevel}
          searchString={searchString}
          sortBy={sortBy}
          darkMode={darkMode}
          profileType={profileType}
          compareMode={compareMode}
          isContextMenuOpen={isContextMenuOpen}
          hoveringName={hoveringName}
          setHoveringName={setHoveringName}
          hoveringRow={hoveringRow}
          colorForSimilarNodes={colorForSimilarNodes}
          highlightSimilarStacksPreference={highlightSimilarStacksPreference}
        />
      )}
    </>
  );
});
