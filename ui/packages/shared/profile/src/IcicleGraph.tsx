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

import {MouseEvent, useEffect, useMemo, useRef, useState, memo, Fragment} from 'react';

import cx from 'classnames';
import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';

import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {
  Mapping,
  Function as ParcaFunction,
  Location,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import type {HoveringNode} from './GraphTooltip';
import GraphTooltip from './GraphTooltip';
import {
  diffColor,
  getLastItem,
  isSearchMatch,
  selectQueryParam,
  useURLState,
} from '@parca/functions';
import {selectDarkMode, useAppSelector} from '@parca/store';
import useIsShiftDown from '@parca/components/src/hooks/useIsShiftDown';
import {Button} from '@parca/components';
import {hexifyAddress} from './utils';

interface IcicleGraphProps {
  graph: Flamegraph;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
}

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
}

interface IcicleGraphRootNodeProps {
  node: FlamegraphRootNode;
  strings: string[];
  mappings: Mapping[];
  locations: Location[];
  functions: ParcaFunction[];
  xScale: (value: number) => number;
  total: number;
  totalWidth: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (node: FlamegraphNode | FlamegraphRootNode | undefined) => void;
}

interface IcicleRectProps {
  x: number;
  y: number;
  width: number;
  height: number;
  color: string;
  name: string;
  onMouseEnter: (e: MouseEvent) => void;
  onMouseLeave: (e: MouseEvent) => void;
  onClick: (e: MouseEvent) => void;
  curPath: string[];
}

const RowHeight = 26;

const icicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
};
const fadedIcicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
  opacity: '0.5',
};

function IcicleRect({
  x,
  y,
  width,
  height,
  color,
  name,
  onMouseEnter,
  onMouseLeave,
  onClick,
  curPath,
}: IcicleRectProps): JSX.Element {
  const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  const isFaded = curPath.length > 0 && name !== curPath[curPath.length - 1];
  const styles = isFaded ? fadedIcicleRectStyles : icicleRectStyles;

  return (
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
        width={width - 1}
        height={height - 1}
        style={{
          opacity:
            currentSearchString !== undefined &&
            currentSearchString !== '' &&
            !isSearchMatch(currentSearchString, name)
              ? 0.5
              : 1,
          fill: color,
        }}
      />
      {width > 5 && (
        <svg width={width - 5} height={height}>
          <text x={5} y={15} style={{fontSize: '12px'}}>
            {name}
          </text>
        </svg>
      )}
    </g>
  );
}

export function nodeLabel(
  node: FlamegraphNode,
  strings: string[],
  mappings: Mapping[],
  locations: Location[],
  functions: ParcaFunction[]
): string {
  if (node.meta?.locationIndex === undefined) return '<unknown>';
  if (node.meta?.locationIndex === 0) return '<unknown>';

  const location = locations[node.meta.locationIndex - 1];
  const mapping =
    location.mappingIndex !== undefined || location.mappingIndex !== 0
      ? mappings[location.mappingIndex - 1]
      : undefined;

  const mappingFile =
    mapping?.fileStringIndex !== undefined ? strings[mapping.fileStringIndex] : '';

  const mappingString = `${
    mappingFile !== '' ? '[' + (getLastItem(mappingFile) ?? '') + '] ' : ''
  }`;

  if (location.lines.length > 0) {
    const funcName =
      strings[functions[location.lines[node.meta.lineIndex].functionIndex - 1].nameStringIndex];
    return `${mappingString} ${funcName}`;
  }

  const address = hexifyAddress(location.address);
  const fallback = `${mappingString}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
}

export function IcicleGraphNodes({
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
}: IcicleGraphNodesProps): JSX.Element {
  const isDarkMode = useAppSelector(selectDarkMode);
  const isShiftDown = useIsShiftDown();

  const nodes =
    curPath.length === 0
      ? data
      : data.filter(
          d => d != null && curPath[0] === nodeLabel(d, strings, mappings, locations, functions)
        );

  const nextLevel = level + 1;

  return (
    <g transform={`translate(${x}, ${y})`}>
      {nodes.map((d, i) => {
        const cumulative = parseFloat(d.cumulative);
        const diff = parseFloat(d.diff);
        const start = nodes.slice(0, i).reduce((sum, d) => sum + parseFloat(d.cumulative), 0);

        const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
        const width =
          nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
            ? totalWidth
            : xScale(cumulative);

        if (width <= 1) {
          return null;
        }

        const name = nodeLabel(d, strings, mappings, locations, functions);
        const key = `${level}-${i}`;
        const nextPath = path.concat([name]);

        const color = diffColor(diff, cumulative, isDarkMode);

        const onClick = (): void => {
          setCurPath(nextPath);
        };

        const xStart = xScale(start);
        const newXScale =
          nextCurPath.length === 0 && curPath.length === 1
            ? scaleLinear().domain([0, cumulative]).range([0, totalWidth])
            : xScale;

        const onMouseEnter = (): void => {
          if (isShiftDown) return;

          setHoveringNode(d);
        };
        const onMouseLeave = (): void => {
          if (isShiftDown) return;

          setHoveringNode(undefined);
        };

        return (
          <Fragment key={`node-${key}`}>
            <IcicleRect
              key={`rect-${key}`}
              x={xStart}
              y={0}
              width={width}
              height={RowHeight}
              name={name}
              color={color}
              onClick={onClick}
              onMouseEnter={onMouseEnter}
              onMouseLeave={onMouseLeave}
              curPath={curPath}
            />
            {data !== undefined && data.length > 0 && (
              <IcicleGraphNodes
                key={`node-${key}`}
                data={d.children}
                strings={strings}
                mappings={mappings}
                locations={locations}
                functions={functions}
                x={xStart}
                y={RowHeight}
                xScale={newXScale}
                total={total}
                totalWidth={totalWidth}
                level={nextLevel}
                setHoveringNode={setHoveringNode}
                path={nextPath}
                curPath={nextCurPath}
                setCurPath={setCurPath}
              />
            )}
          </Fragment>
        );
      })}
    </g>
  );
}

const MemoizedIcicleGraphNodes = memo(IcicleGraphNodes);

export function IcicleGraphRootNode({
  node,
  strings,
  mappings,
  locations,
  functions,
  xScale,
  total,
  totalWidth,
  setHoveringNode,
  setCurPath,
  curPath,
}: IcicleGraphRootNodeProps): JSX.Element {
  const isDarkMode = useAppSelector(selectDarkMode);
  const isShiftDown = useIsShiftDown();

  const cumulative = parseFloat(node.cumulative);
  const diff = parseFloat(node.diff);
  const color = diffColor(diff, cumulative, isDarkMode);

  const onClick = (): void => setCurPath([]);
  const onMouseEnter = (): void => {
    if (isShiftDown) return;

    setHoveringNode(node);
  };
  const onMouseLeave = (): void => {
    if (isShiftDown) return;

    setHoveringNode(undefined);
  };

  const path: string[] = [];

  return (
    <g transform={'translate(0, 0)'}>
      <IcicleRect
        x={0}
        y={0}
        width={totalWidth}
        height={RowHeight}
        name={'root'}
        color={color}
        onClick={onClick}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        curPath={curPath}
      />
      <MemoizedIcicleGraphNodes
        data={node.children}
        strings={strings}
        mappings={mappings}
        locations={locations}
        functions={functions}
        x={0}
        y={RowHeight}
        xScale={xScale}
        total={total}
        totalWidth={totalWidth}
        level={0}
        setHoveringNode={setHoveringNode}
        path={path}
        curPath={curPath}
        setCurPath={setCurPath}
      />
    </g>
  );
}

const MemoizedIcicleGraphRootNode = memo(IcicleGraphRootNode);

export default function IcicleGraph({
  graph,
  width,
  setCurPath,
  curPath,
  sampleUnit,
}: IcicleGraphProps): JSX.Element {
  const [hoveringNode, setHoveringNode] = useState<
    FlamegraphNode | FlamegraphRootNode | undefined
  >();
  const [pos, setPos] = useState([0, 0]);
  const [height, setHeight] = useState(0);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);
  const [rawDashboardItems] = useURLState({
    param: 'dashboard_items',
  });

  const dashboardItems = rawDashboardItems as string[];

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width, graph]);

  const total = useMemo(() => parseFloat(graph.total), [graph.total]);
  const xScale = useMemo(() => {
    if (width === undefined) {
      return () => 0;
    }
    return scaleLinear().domain([0, total]).range([0, width]);
  }, [total, width]);

  if (graph.root === undefined || width === undefined) {
    return <></>;
  }

  const throttledSetPos = throttle(setPos, 20);
  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    throttledSetPos([rel[0], rel[1]]);
  };

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)} className="relative">
      <GraphTooltip
        unit={sampleUnit}
        total={total}
        x={pos[0]}
        y={pos[1]}
        hoveringNode={hoveringNode as HoveringNode}
        contextElement={svg.current}
        strings={graph.stringTable}
        mappings={graph.mapping}
        locations={graph.locations}
        functions={graph.function}
      />
      <div
        className={cx(
          dashboardItems.length > 1 ? 'left-[25px]' : 'top-[-45px]',
          'flex justify-start absolute top-[-45px]'
        )}
      >
        <Button
          color="neutral"
          onClick={() => setCurPath([])}
          disabled={curPath.length === 0}
          className="w-auto"
          variant="neutral"
        >
          Reset View
        </Button>
      </div>
      <svg
        className="font-robotoMono"
        width={width}
        height={height}
        onMouseMove={onMouseMove}
        preserveAspectRatio="xMinYMid"
        ref={svg}
      >
        <g ref={ref}>
          <MemoizedIcicleGraphRootNode
            node={graph.root}
            strings={graph.stringTable}
            mappings={graph.mapping}
            locations={graph.locations}
            functions={graph.function}
            setHoveringNode={setHoveringNode}
            curPath={curPath}
            setCurPath={setCurPath}
            xScale={xScale}
            total={total}
            totalWidth={width}
          />
        </g>
      </svg>
    </div>
  );
}
