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

import cx from 'classnames';
import {scaleLinear} from 'd3-scale';
import {
  Mapping,
  Function as ParcaFunction,
  Location,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {isSearchMatch} from '@parca/functions';
import {FlamegraphNode, FlamegraphRootNode} from '@parca/client';

import {nodeLabel} from './utils';
import useNodeColor from './useNodeColor';
import {useKeyDown} from '@parca/components';

export const RowHeight = 26;

interface IcicleGraphNodesProps {
  data: FlamegraphNode[];
  strings: string[];
  mappings: Mapping[];
  locations: Location[];
  functions: ParcaFunction[];
  x: number;
  y: number;
  total: number;
  totalWidth: number;
  level: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (node: FlamegraphNode | FlamegraphRootNode | undefined) => void;
  path: string[];
  xScale: (value: number) => number;
  searchString?: string;
  compareMode: boolean;
}

export const IcicleGraphNodes = React.memo(function IcicleGraphNodes({
  data,
  strings,
  mappings,
  locations,
  functions,
  x,
  y,
  xScale,
  total,
  totalWidth,
  level,
  setHoveringNode,
  path,
  setCurPath,
  curPath,
  searchString,
  compareMode,
}: IcicleGraphNodesProps): JSX.Element {
  const nodes =
    curPath.length === 0
      ? data
      : data.filter(
          d => d != null && curPath[0] === nodeLabel(d, strings, mappings, locations, functions)
        );

  return (
    <g transform={`translate(${x}, ${y})`}>
      {nodes.map((d, i) => {
        const start = nodes.slice(0, i).reduce((sum, d) => sum + parseFloat(d.cumulative), 0);
        const xStart = xScale(start);

        return (
          <IcicleNode
            key={`node-${level}-${i}`}
            x={xStart}
            y={0}
            totalWidth={totalWidth}
            height={RowHeight}
            path={path}
            setCurPath={setCurPath}
            setHoveringNode={setHoveringNode}
            level={level}
            curPath={curPath}
            data={d}
            strings={strings}
            mappings={mappings}
            locations={locations}
            functions={functions}
            total={total}
            xScale={xScale}
            searchString={searchString}
            compareMode={compareMode}
          />
        );
      })}
    </g>
  );
});

interface IcicleNodeProps {
  x: number;
  y: number;
  height: number;
  totalWidth: number;
  curPath: string[];
  level: number;
  data: FlamegraphNode;
  strings: string[];
  mappings: Mapping[];
  locations: Location[];
  functions: ParcaFunction[];
  path: string[];
  total: number;
  setCurPath: (path: string[]) => void;
  setHoveringNode: (node: FlamegraphNode | FlamegraphRootNode | undefined) => void;
  xScale: (value: number) => number;
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

export const IcicleNode = React.memo(function IcicleNode({
  x,
  y,
  height,
  setCurPath,
  setHoveringNode,
  curPath,
  level,
  path,
  data,
  strings,
  mappings,
  locations,
  functions,
  total,
  totalWidth,
  xScale,
  isRoot = false,
  searchString,
  compareMode,
}: IcicleNodeProps): JSX.Element {
  const {isShiftDown} = useKeyDown();
  const colorResult = useNodeColor({data, compareMode});
  const name = useMemo(() => {
    return isRoot ? 'root' : nodeLabel(data, strings, mappings, locations, functions);
  }, [data, strings, mappings, locations, functions, isRoot]);
  const nextPath = path.concat([name]);
  const isFaded = curPath.length > 0 && name !== curPath[curPath.length - 1];
  const styles = isFaded ? fadedIcicleRectStyles : icicleRectStyles;
  const nextLevel = level + 1;
  const cumulative = parseFloat(data.cumulative);
  const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
  const newXScale =
    nextCurPath.length === 0 && curPath.length === 1
      ? scaleLinear().domain([0, cumulative]).range([0, totalWidth])
      : xScale;

  const width =
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

    setHoveringNode(data);
  };
  const onMouseLeave = (): void => {
    if (isShiftDown) return;

    setHoveringNode(undefined);
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
          width={width - 1}
          height={height - 1}
          style={{
            fill: colorResult,
          }}
          className={cx({
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
      {data.children !== undefined && data.children.length > 0 && (
        <IcicleGraphNodes
          data={data.children}
          strings={strings}
          mappings={mappings}
          locations={locations}
          functions={functions}
          x={x}
          y={RowHeight}
          xScale={newXScale}
          total={total}
          totalWidth={totalWidth}
          level={nextLevel}
          setHoveringNode={setHoveringNode}
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
