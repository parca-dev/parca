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

import React from 'react';

import {Binary, StructRow} from 'apache-arrow';
import cx from 'classnames';
import twColors from 'tailwindcss/colors';

import {scaleLinear} from '@parca/utilities';

import {FIELD_CHILDREN, FIELD_CUMULATIVE, FIELD_GROUPBY_METADATA} from '.';
import {ProfileSource} from '../../ProfileSource';
import {IcicleNode, IcicleNodeProps, RowHeight} from './IcicleGraphNodes';
import {arrowToString, boundsFromProfileSource} from './utils';

interface IcicleChartRootNodeSpecificProps {
  profileSource?: ProfileSource;
}

export const IcicleChartRootNode = React.memo(function IcicleChartRootNodeNonMemo({
  table,
  row,
  colors,
  colorBy,
  y,
  height,
  setCurPath,
  curPath,
  level,
  path,
  total,
  totalWidth,
  xScale,
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
  profileSource,
}: IcicleNodeProps & IcicleChartRootNodeSpecificProps): React.JSX.Element {
  // get the columns to read from
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const groupByMetadata = table.getChild(FIELD_GROUPBY_METADATA);
  const cumulative = cumulativeColumn?.get(row) !== null ? BigInt(cumulativeColumn?.get(row)) : 0n;
  const childRows: number[] = Array.from<number>(
    table.getChild(FIELD_CHILDREN)?.get(row) ?? []
  ).sort((a, b) => a - b);

  const tsBounds = boundsFromProfileSource(profileSource);

  const nextLevel = level + 1;
  const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
  const tsXScale = scaleLinear([tsBounds[0], tsBounds[1]], [0, totalWidth]);

  const width: number =
    nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
      ? totalWidth
      : xScale(BigInt(cumulative));

  if (width <= 1) {
    return <>{null}</>;
  }

  return (
    <>
      {childRows.map(row => {
        const groupByFields = (
          groupByMetadata?.get(row) as StructRow<Record<string, Binary>>
        ).toJSON();

        const tsStr = arrowToString(groupByFields.time_nanos) as string;

        const tsNanos = BigInt(parseInt(tsStr, 10));
        const durationStr = arrowToString(groupByFields.duration) as string;
        const duration = parseInt(durationStr, 10);

        const x = tsXScale(tsNanos);
        const width = tsXScale(tsNanos + BigInt(Math.round(duration))) - x;

        const cumulative =
          cumulativeColumn?.get(row) !== null ? BigInt(cumulativeColumn?.get(row)) : 0n;
        const newXScale = scaleLinear([0n, BigInt(cumulative)], [0, width]);

        return (
          <>
            <g transform={`translate(${x + 1}, ${y + 1})`}>
              <rect
                x={0}
                y={0}
                width={width}
                height={RowHeight}
                style={{
                  fill: darkMode ? twColors.gray[500] : twColors.gray[200],
                }}
                className={cx(`stroke-white dark:stroke-gray-700 fill-gray-600 dark:fill-gray-100`)}
              />
              <svg width={width - 5} height={height}>
                <text x={5} y={15} style={{fontSize: '12px'}}>
                  root
                </text>
              </svg>
            </g>
            <IcicleNode
              table={table}
              row={row}
              colors={colors}
              colorBy={colorBy}
              x={x}
              y={RowHeight}
              totalWidth={width ?? 1}
              height={RowHeight}
              setCurPath={setCurPath}
              curPath={curPath}
              total={total}
              xScale={newXScale}
              path={path}
              level={nextLevel}
              searchString={(searchString as string) ?? ''}
              setHoveringRow={setHoveringRow}
              setHoveringLevel={setHoveringLevel}
              sortBy={sortBy}
              darkMode={darkMode}
              compareMode={compareMode}
              profileType={profileType}
              isContextMenuOpen={isContextMenuOpen}
              hoveringName={hoveringName}
              setHoveringName={setHoveringName}
              hoveringRow={hoveringRow}
              colorForSimilarNodes={colorForSimilarNodes}
              highlightSimilarStacksPreference={highlightSimilarStacksPreference}
              key={row}
            />
          </>
        );
      })}
    </>
  );
});
