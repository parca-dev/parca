import React, {MouseEvent, useEffect, useRef, useState} from 'react';
import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';
import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {FlamegraphTooltip} from '@parca/components';
import {getLastItem, diffColor, isSearchMatch} from '@parca/functions';
import {useAppSelector, selectDarkMode, selectSearchNodeString} from '@parca/store';

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
}: IcicleRectProps) {
  const currentSearchString = useAppSelector(selectSearchNodeString);
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
            Boolean(currentSearchString) && !isSearchMatch(currentSearchString, name) ? 0.5 : 1,
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

export function nodeLabel(node: FlamegraphNode): string {
  if (node.meta === undefined) return '<unknown>';
  const mapping = `${
    node.meta?.mapping?.file !== undefined && node.meta?.mapping?.file !== ''
      ? '[' + getLastItem(node.meta.mapping.file) + '] '
      : ''
  }`;
  if (node.meta.function?.name !== undefined && node.meta.function?.name !== '')
    return mapping + node.meta.function.name;

  const address = hexifyAddress(node.meta.location?.address);
  const fallback = `${mapping}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
}

export function IcicleGraphNodes({
  data,
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
}: IcicleGraphNodesProps) {
  const isDarkMode = useAppSelector(selectDarkMode);

  const nodes =
    curPath.length === 0 ? data : data.filter(d => d != null && curPath[0] === nodeLabel(d));

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
          return <></>;
        }

        const name = nodeLabel(d);
        const key = `${level}-${i}`;
        const nextPath = path.concat([name]);

        const color = diffColor(diff, cumulative, isDarkMode);

        const onClick = () => {
          setCurPath(nextPath);
        };

        const xStart = xScale(start);
        const newXScale =
          nextCurPath.length === 0 && curPath.length === 1
            ? scaleLinear().domain([0, cumulative]).range([0, totalWidth])
            : xScale;

        const onMouseEnter = () => setHoveringNode(d);
        const onMouseLeave = () => setHoveringNode(undefined);

        return (
          <React.Fragment>
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
          </React.Fragment>
        );
      })}
    </g>
  );
}

const MemoizedIcicleGraphNodes = React.memo(IcicleGraphNodes);

export function IcicleGraphRootNode({
  node,
  xScale,
  total,
  totalWidth,
  setHoveringNode,
  setCurPath,
  curPath,
}: IcicleGraphRootNodeProps) {
  const isDarkMode = useAppSelector(selectDarkMode);

  const cumulative = parseFloat(node.cumulative);
  const diff = parseFloat(node.diff);
  const color = diffColor(diff, cumulative, isDarkMode);

  const onClick = () => setCurPath([]);
  const onMouseEnter = () => setHoveringNode(node);
  const onMouseLeave = () => setHoveringNode(undefined);
  const path = [];

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

const MemoizedIcicleGraphRootNode = React.memo(IcicleGraphRootNode);

export default function IcicleGraph({
  graph,
  width,
  setCurPath,
  curPath,
  sampleUnit,
}: IcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<
    FlamegraphNode | FlamegraphRootNode | undefined
  >();
  const [pos, setPos] = useState([0, 0]);
  const [height, setHeight] = useState(0);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width]);

  if (graph.root === undefined || width === undefined) return <></>;

  const throttledSetPos = throttle(setPos, 20);
  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    throttledSetPos([rel[0], rel[1]]);
  };

  const total = parseFloat(graph.total);
  const xScale = scaleLinear().domain([0, total]).range([0, width]);

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)}>
      <FlamegraphTooltip
        unit={sampleUnit}
        total={total}
        x={pos[0]}
        y={pos[1]}
        hoveringNode={hoveringNode}
        contextElement={svg.current}
      />
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
