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

import {CallgraphEdge, CallgraphNode} from '@parca/client';

import {DEFAULT_NODE_HEIGHT} from './constants';

export const pixelsToInches = (pixels: number): number => pixels / 96;

export const getCurvePoints = ({
  pos,
  xScale = n => n,
  yScale = n => n,
  source = [],
  target = [],
  offset = 0,
  isSelfLoop = false,
}: {
  pos: string;
  xScale?: (pos: number) => number;
  yScale?: (pos: number) => number;
  source?: number[];
  target?: number[];
  isSelfLoop?: boolean;
  offset?: number;
}): number[] => {
  if (isSelfLoop) {
    const [sourceX, sourceY] = source;
    const [targetX, targetY] = target;

    return [
      sourceX,
      sourceY + offset,
      sourceX,
      sourceY + 3 * offset,
      targetX + 5 * offset,
      targetY,
      targetX + offset,
      targetY,
    ];
  }

  // graphviz pos format is in format 'endpoint,startpoint,triple(cp1,cp2,end),...triple...'
  const scalePoint = (point: number[]): number[] => [xScale(point[0]), yScale(point[1])];
  const strAsNumArray = (string: string): number[] =>
    string
      .replace('e,', '')
      .split(',')
      .map(str => Number(str));
  const getLastPointWithOffset = (target: number[], last: number[], offset: number): number[] => {
    const [targetX, targetY] = target;
    const [lastX, lastY] = last;
    const diffX = targetX - lastX;
    const diffY = targetY - lastY;
    const diffZ = Math.hypot(diffX, diffY);

    const offsetX = (diffX * offset) / diffZ;
    const offsetY = (diffY * offset) / diffZ;

    return [targetX - offsetX, targetY - offsetY];
  };
  const points: number[][] = pos.split(' ').map(str => strAsNumArray(str));
  const scaledPoints: number[][] = points.map(point => scalePoint(point));

  const lastPointIndex = scaledPoints.length - 1;
  const lastPointWithOffset = getLastPointWithOffset(target, scaledPoints[lastPointIndex], offset);

  return [source, ...scaledPoints.slice(2, points.length - 1), lastPointWithOffset].flat();
};

const objectAsDotAttributes = (obj: {[key: string]: string | number}): string =>
  Object.entries(obj)
    .map(entry => `${entry[0]}="${entry[1]}"`)
    .join(' ');

export const jsonToDot = ({
  graph,
  colorRange,
}: {
  graph: {nodes: CallgraphNode[]; edges: CallgraphEdge[]};
  width: number;
  colorRange: [string, string];
}): string => {
  const {nodes, edges} = graph;
  const cumulatives = nodes.map((node: CallgraphNode) => node.cumulative);
  const cumulativesRange = d3.extent(cumulatives).map(value => Number(value));

  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain(cumulativesRange)
    .range(colorRange);
  const colorOpacityScale = d3.scaleSequentialLog().domain(cumulativesRange).range([0.2, 1]);
  const boxWidthScale = d3
    .scaleLog()
    .domain(cumulativesRange)
    .range([DEFAULT_NODE_HEIGHT, DEFAULT_NODE_HEIGHT + 40]);

  const nodesAsStrings = nodes.map((node: CallgraphNode) => {
    const dataAttributes = {
      address: node.meta?.location?.address ?? '',
      functionName: node.meta?.function?.name ?? '',
      cumulative: node.cumulative ?? '',
      root: (node.id === 'root').toString(),
      // TODO: set box width scale to be based on flat value once we have that value available
      width: boxWidthScale(Number(node.cumulative)),
      color: colorScale(Number(node.cumulative)),
    };

    return `"${node.id}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const edgesAsStrings = edges.map((edge: CallgraphEdge) => {
    const dataAttributes = {
      cumulative: edge.cumulative,
      color: colorRange[1],
      opacity: colorOpacityScale(Number(edge.cumulative)),
      boxHeight: DEFAULT_NODE_HEIGHT,
    };

    return `"${edge.source}" -> "${edge.target}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const graphAsDot = `digraph "callgraph" {
      rankdir="BT"
      overlap="prism"
      ratio="1,3"
      margin=15
      edge [margin=0]
      node [shape=box style=rounded height=${DEFAULT_NODE_HEIGHT}]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};
