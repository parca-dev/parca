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

import {useKeyDown} from '@parca/components';
import {
  EVERYTHING_ELSE,
  selectBinaries,
  setHoveringRow,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {getLastItem, isSearchMatch, scaleLinear} from '@parca/utilities';

import {
  FIELD_CHILDREN,
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_NAME,
  FIELD_MAPPING_FILE,
} from './index';
import {nodeLabel} from './utils';

export const RowHeight = 26;

interface IcicleGraphNodesProps {
  table: Table<any>;
  row: number;
  mappingColors: mappingColors;
  children: number[];
  x: number;
  y: number;
  total: bigint;
  totalWidth: number;
  level: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  path: string[];
  xScale: (value: bigint) => number;
  searchString?: string;
  sortBy: string;
  compareMode: boolean;
}

export const IcicleGraphNodes = React.memo(function IcicleGraphNodesNoMemo({
  table,
  children,
  mappingColors,
  x,
  y,
  xScale,
  total,
  totalWidth,
  level,
  path,
  setCurPath,
  curPath,
  sortBy,
  searchString,
  compareMode,
}: IcicleGraphNodesProps): React.JSX.Element {
  const cumulatives = table.getChild(FIELD_CUMULATIVE);

  if (children === undefined || children.length === 0) {
    return <></>;
  }

  children =
    curPath.length === 0
      ? children
      : children.filter(c => nodeLabel(table, c, false) === curPath[0]);

  let childrenCumulative = BigInt(0);
  const childrenElements: ReactNode[] = [];
  children.forEach((child, i) => {
    const xStart = Math.floor(xScale(childrenCumulative));
    const c: bigint = cumulatives?.get(child);
    childrenCumulative += c;

    childrenElements.push(
      <IcicleNode
        key={`node-${level}-${i}`}
        table={table}
        row={child}
        mappingColors={mappingColors}
        x={xStart}
        y={0}
        totalWidth={totalWidth}
        height={RowHeight}
        path={path}
        setCurPath={setCurPath}
        level={level}
        curPath={curPath}
        total={total}
        xScale={xScale}
        sortBy={sortBy}
        searchString={searchString}
        compareMode={compareMode}
      />
    );
  });

  return <g transform={`translate(${x}, ${y})`}>{childrenElements}</g>;
});

interface mappingColors {
  [key: string]: string;
}

interface IcicleNodeProps {
  x: number;
  y: number;
  height: number;
  totalWidth: number;
  curPath: string[];
  level: number;
  table: Table<any>;
  row: number;
  mappingColors: mappingColors;
  path: string[];
  total: bigint;
  setCurPath: (path: string[]) => void;
  xScale: (value: bigint) => number;
  isRoot?: boolean;
  searchString?: string;
  sortBy: string;
  compareMode: boolean;
}

const icicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
};
const fadedIcicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
  opacity: '0.5',
};

export const IcicleNode = React.memo(function IcicleNodeNoMemo({
  table,
  row,
  mappingColors,
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
  sortBy,
  compareMode,
}: IcicleNodeProps): React.JSX.Element {
  const {isShiftDown} = useKeyDown();
  const dispatch = useAppDispatch();

  // get the columns to read from
  const mappingColumn = table.getChild(FIELD_MAPPING_FILE);
  const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const diffColumn = table.getChild(FIELD_DIFF);
  // get the actual values from the columns
  const mappingFile: string | null = mappingColumn?.get(row);
  const functionName: string | null = functionNameColumn?.get(row);
  const cumulative: bigint = cumulativeColumn?.get(row);
  const children: number[] = Array.from(table.getChild(FIELD_CHILDREN)?.get(row) ?? []);

  // TODO: Maybe it's better to pass down the sorter function as prop instead of figuring this out here.
  switch (sortBy) {
    case FIELD_FUNCTION_NAME:
      children.sort((a, b) => {
        // TODO: Support fallthrough to comparing addresses or something
        const afn: string = functionNameColumn?.get(a);
        const bfn: string = functionNameColumn?.get(b);
        return afn.localeCompare(bfn);
      });
      break;
    case FIELD_CUMULATIVE:
      children.sort((a, b) => {
        const aCumulative: bigint = cumulativeColumn?.get(a);
        const bCumulative: bigint = cumulativeColumn?.get(b);
        return Number(bCumulative - aCumulative);
      });
      break;
    case FIELD_DIFF:
      children.sort((a, b) => {
        const aDiff: bigint | null = diffColumn?.get(a);
        const bDiff: bigint | null = diffColumn?.get(b);
        // TODO: Double check this sorting actually makes sense, when coloring is back
        if (aDiff !== null && bDiff !== null) {
          return Number(bDiff - aDiff);
        }
        if (aDiff === null && bDiff !== null) {
          return 1;
        }
        if (aDiff !== null && bDiff === null) {
          return -1;
        }
        // both are null
        return 0;
      });
      break;
  }

  const binaries = useAppSelector(selectBinaries);
  // const {isShiftDown} = useKeyDown();
  // const colorResult = useNodeColor({table, row, compareMode});
  const name = useMemo(() => {
    return isRoot ? 'root' : nodeLabel(table, row, binaries.length > 1);
  }, [table, row, isRoot, binaries]);
  const nextPath = path.concat([name]);
  const isFaded = curPath.length > 0 && name !== curPath[curPath.length - 1];
  const styles = isFaded ? fadedIcicleRectStyles : icicleRectStyles;
  const nextLevel = level + 1;
  const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
  const newXScale =
    nextCurPath.length === 0 && curPath.length === 1
      ? scaleLinear([0n, cumulative], [0, totalWidth])
      : xScale;

  const width: number =
    nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
      ? totalWidth
      : xScale(cumulative);

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
    if (isShiftDown) return;
    dispatch(setHoveringRow({row}));
  };

  const onMouseLeave = (): void => {
    if (isShiftDown) return;
    dispatch(setHoveringRow(undefined));
  };

  // To get the color we first check if the function name starts with 'runtime'.
  // If it does, we color it as runtime. Otherwise, we check the mapping file.
  // If there is no mapping file, we color it as 'everything else'.
  const color =
    functionName?.startsWith('runtime') === true
      ? mappingColors.runtime
      : mappingColors[getLastItem(mappingFile ?? '') ?? EVERYTHING_ELSE];

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
            fill: color,
          }}
          className={cx('stroke-white dark:stroke-gray-700', {
            'opacity-50': isHighlightEnabled && !isHighlighted,
          })}
        />
        {width > 5 && (
          <svg width={width - 5} height={height}>
            <text x={5} y={15} style={{fontSize: '12px'}}>
              {name}
            </text>
          </svg>
        )}
      </g>
      {children.length > 0 && (
        <IcicleGraphNodes
          table={table}
          row={row}
          mappingColors={mappingColors}
          children={children}
          x={x}
          y={RowHeight}
          xScale={newXScale}
          total={total}
          totalWidth={totalWidth}
          level={nextLevel}
          path={nextPath}
          curPath={nextCurPath}
          setCurPath={setCurPath}
          searchString={searchString}
          sortBy={sortBy}
          compareMode={compareMode}
        />
      )}
    </>
  );
});
