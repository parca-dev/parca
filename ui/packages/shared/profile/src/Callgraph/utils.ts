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

import * as d3 from 'd3';
import {CallgraphNode, CallgraphEdge} from '@parca/client';

export const pixelsToInches = (pixels: number): number => pixels / 96;
export const inchesToPixels = (inches: number): number => inches * 96;

export const getCurvePoints = ({
  pos,
  xScale = n => n,
  yScale = n => n,
  source = [],
  target = [],
  offsetRadius = 0,
  isSelfLoop = false,
}: {
  pos: string;
  xScale?: (pos: number) => number;
  yScale?: (pos: number) => number;
  source?: number[];
  target?: number[];
  isSelfLoop?: boolean;
  offsetRadius?: number;
}): number[] => {
  if (isSelfLoop) {
    const [sourceX, sourceY] = source;
    const [targetX, targetY] = target;

    return [
      sourceX,
      sourceY + offsetRadius,
      sourceX,
      sourceY + 3 * offsetRadius,
      targetX + 5 * offsetRadius,
      targetY,
      targetX + offsetRadius,
      targetY,
    ];
  }

  // graphviz pos format is in format 'endpoint,startpoint,triple(cp1,cp2,end),...triple...'
  const scalePoint = (point: number[]): number[] => [xScale(point[0]), yScale(point[1])];
  const strAsNumArray = string =>
    string
      .replace('e,', '')
      .split(',')
      .map(str => Number(str));
  const getLastPointWithOffset = (target: number[], last: number[], offsetRadius): number[] => {
    const [targetX, targetY] = target;
    const [lastX, lastY] = last;
    const diffX = targetX - lastX;
    const diffY = targetY - lastY;
    const diffZ = Math.hypot(diffX, diffY);

    const offsetX = (diffX * offsetRadius) / diffZ;
    const offsetY = (diffY * offsetRadius) / diffZ;

    return [targetX - offsetX, targetY - offsetY];
  };
  const points: number[][] = pos.split(' ').map(str => strAsNumArray(str));
  const scaledPoints: number[][] = points.map(point => scalePoint(point));

  const lastPointIndex = scaledPoints.length - 1;
  const lastPointWithOffset = getLastPointWithOffset(
    target,
    scaledPoints[lastPointIndex],
    offsetRadius
  );

  return [source, ...scaledPoints.slice(2, points.length - 1), lastPointWithOffset].flat();
};

const objectAsDotAttributes = (obj: {[key: string]: string | number}): string =>
  Object.entries(obj)
    .map(entry => `${entry[0]}="${entry[1]}"`)
    .join(' ');

export const jsonToDot = ({
  graph,
  width,
  colorRange,
}: {
  graph: {nodes: CallgraphNode[]; edges: CallgraphEdge[]};
  width: number;
  colorRange: [string, string];
}): string => {
  const {nodes, edges} = graph;
  const defaultNodeRadius = 12;
  const cumulatives = nodes.map((node: CallgraphNode) => node.cumulative);
  const cumulativesRange = d3.extent(cumulatives).map(value => Number(value));
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain(cumulativesRange)
    .range(colorRange);
  const colorOpacityScale = d3.scaleSequentialLog().domain(cumulativesRange).range([0.2, 1]);
  const nodeRadiusScale = d3
    .scaleLog()
    .domain(cumulativesRange)
    .range([defaultNodeRadius - 2, defaultNodeRadius + 3]);

  const nodesAsStrings = nodes.map((node: CallgraphNode) => {
    const dataAttributes = {
      label: node.meta?.function?.name ? node.meta?.function?.name.substring(0, 15) : '',
      address: node.meta?.location?.address ?? '',
      functionName: node.meta?.function?.name ?? '',
      cumulative: node.cumulative ?? '',
      root: (node.id === 'root').toString(),
      // width: nodeRadiusScale(Number(node.cumulative)) * 2,
      color: colorScale(Number(node.cumulative)),
    };

    return `"${node.id}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const edgesAsStrings = edges.map((edge: CallgraphEdge) => {
    const dataAttributes = {
      cumulative: edge.cumulative,
      color: colorRange[1],
      opacity: colorOpacityScale(Number(edge.cumulative)),
      // nodeRadius: nodeRadiusScale(Number(edge.cumulative)),
    };

    return `"${edge.source}" -> "${edge.target}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  // can provide a node label that will size the nodes appropriately (and change the layout as well to account for diff widths)
  // then needs to set fixedsize=shape
  // ratio="1,3"
  // size="${pixelsToInches(width)}, ${pixelsToInches(width) * 10}"
  const graphAsDot = `digraph "callgraph" {
      rankdir="BT"
      overlap="prism"
      ratio="1,3"
      margin=15
      edge [margin=0]
      node [shape=record style=rounded fixedsize=shape height=0.3]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  console.log(graphAsDot);

  return graphAsDot;
};
