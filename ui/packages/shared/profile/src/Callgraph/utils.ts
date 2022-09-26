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

import {CallgraphNode, CallgraphEdge} from '@parca/client';

export const pixelsToInches = (pixels: number): number => pixels / 96;

export const parseEdgePos = ({
  pos,
  xScale = n => n,
  yScale = n => n,
  source = [],
  target = [],
  nodeRadius,
  isSelfLoop = false,
}: {
  pos: string;
  xScale?: (pos: number) => void;
  yScale?: (pos: number) => void;
  source?: number[];
  target?: number[];
  nodeRadius: number;
  isSelfLoop?: boolean;
}): number[] => {
  const parts = pos.split(' ');
  const arrow = parts.shift() ?? '';
  const partsAsArrays = parts.map(part => part.split(','));
  const scalePosArray = (posArr: string[]): number[] => [+xScale(+posArr[0]), +yScale(+posArr[1])];
  const [_start, cp1, cp2, _end] = partsAsArrays.map(posArr => scalePosArray(posArr));
  const arrowEnd: number[] = scalePosArray(arrow.replace('e,', '').split(','));

  const getTargetWithOffset = (target: number[], lastEdgePoint: number[]): number[] => {
    const diffX = target[0] - lastEdgePoint[0];
    const diffY = target[1] - lastEdgePoint[1];
    const diffZ = Math.hypot(diffX, diffY);

    const offsetX = (diffX * nodeRadius) / diffZ;
    const offsetY = (diffY * nodeRadius) / diffZ;

    return [target[0] - offsetX, target[1] - offsetY];
  };

  if (isSelfLoop) {
    const [sourceX, sourceY] = source;
    const [targetX, targetY] = target;
    return [
      sourceX,
      sourceY + nodeRadius,
      sourceX,
      sourceY + 3 * nodeRadius,
      targetX + 5 * nodeRadius,
      targetY,
      targetX + nodeRadius,
      targetY,
    ];
  }
  return [...source, ...cp1, ...cp2, ...getTargetWithOffset(target, arrowEnd)];
};

export const jsonToDot = ({
  graph,
  width,
  nodeRadius,
}: {
  graph: {nodes: CallgraphNode[]; edges: CallgraphEdge[]};
  width: number;
  nodeRadius: number;
}): string => {
  const {nodes, edges} = graph;

  const objectAsDotAttributes = (obj: {[key: string]: string}): string =>
    Object.entries(obj)
      .map(entry => `${entry[0]}="${entry[1]}"`)
      .join(' ');

  const nodesAsStrings = nodes.map((node: CallgraphNode) => {
    const dataAttributes = {
      address: node.meta?.location?.address ?? '',
      functionName: node.meta?.function?.name ?? '',
      cumulative: node.cumulative ?? '',
      root: (node.id === 'root').toString(),
    };

    return `"${node.id}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const edgesAsStrings = edges.map((edge: CallgraphEdge) => {
    const dataAttributes = {
      cumulative: edge.cumulative,
    };
    return `"${edge.source}" -> "${edge.target}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const graphAsDot = `digraph "callgraph" {
      rankdir="TB"
      ratio="1,3"
      size="${pixelsToInches(width)}, ${pixelsToInches(width)}!"
      margin=10
      edge [margin=0]
      node [margin=0 width=${nodeRadius}]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};
