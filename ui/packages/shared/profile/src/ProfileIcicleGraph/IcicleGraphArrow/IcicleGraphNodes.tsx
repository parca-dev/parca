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
import {selectBinaries, setHoveringRow, useAppDispatch, useAppSelector} from '@parca/store';
import {isSearchMatch, scaleLinear} from '@parca/utilities';

import {FIELD_CHILDREN, FIELD_CUMULATIVE} from './index';
import {nodeLabel} from './utils';

export const RowHeight = 26;

interface IcicleGraphNodesProps {
  table: Table<any>;
  row: number;
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
  compareMode: boolean;
}

export const IcicleGraphNodes = React.memo(function IcicleGraphNodesNoMemo({
  table,
  children,
  x,
  y,
  xScale,
  total,
  totalWidth,
  level,
  path,
  setCurPath,
  curPath,
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
        searchString={searchString}
        compareMode={compareMode}
      />
    );
  });

  return <g transform={`translate(${x}, ${y})`}>{childrenElements}</g>;
});

interface IcicleNodeProps {
  x: number;
  y: number;
  height: number;
  totalWidth: number;
  curPath: string[];
  level: number;
  table: Table<any>;
  row: number;
  path: string[];
  total: bigint;
  setCurPath: (path: string[]) => void;
  xScale: (value: bigint) => number;
  isRoot?: boolean;
  searchString?: string;
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
  compareMode,
}: IcicleNodeProps): React.JSX.Element {
  const {isShiftDown} = useKeyDown();
  const dispatch = useAppDispatch();

  const cumulative: bigint = table.getChild(FIELD_CUMULATIVE)?.get(row);
  const children: number[] = Array.from(table.getChild(FIELD_CHILDREN)?.get(row) ?? []);

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
            fill: '#929FEB', // TODO: Introduce color coding for binaries again
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
          compareMode={compareMode}
        />
      )}
    </>
  );
});
