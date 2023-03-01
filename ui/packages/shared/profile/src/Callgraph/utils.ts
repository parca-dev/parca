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
import {withAlphaHex} from 'with-alpha-hex';

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
  const cumulatives = edges.map((edge: CallgraphEdge) => Number(edge.cumulative));
  const cumulativesRange = d3.extent(cumulatives) as [number, number];
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain(cumulativesRange)
    .range(colorRange);
  const colorOpacityScale = d3.scaleLinear().domain(cumulativesRange).range([0.5, 1]);
  // const boxWidthScale = d3
  //   .scaleLog()
  //   .domain(cumulativesRange)
  //   .range([DEFAULT_NODE_HEIGHT, DEFAULT_NODE_HEIGHT + 40]);

  const nodesAsStrings = nodes.map((node: CallgraphNode) => {
    const rgbColor = colorScale(Number(node.cumulative));
    const hexColor = d3.color(rgbColor)?.formatHex() ?? 'red';
    const dataAttributes = {
      label: node.meta?.function?.name.substring(0, 12) ?? '',
      root: (node.id === 'root').toString(),
      fillcolor: hexColor,
      className: 'node',
      id: node.id,
    };

    return `"${node.id}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const edgesAsStrings = edges.map((edge: CallgraphEdge) => {
    const dataAttributes = {
      cumulative: edge.cumulative,
      color: withAlphaHex(colorRange[1], colorOpacityScale(Number(edge.cumulative))),
      className: 'edge',
      // boxHeight: DEFAULT_NODE_HEIGHT,
    };

    return `"${edge.source}" -> "${edge.target}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const graphAsDot = `digraph "callgraph" {
      rankdir="BT"
      overlap="prism"
      ratio="1,3"
      margin=15
      edge [margin=0]
      node [shape=box style="rounded,filled"]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};
