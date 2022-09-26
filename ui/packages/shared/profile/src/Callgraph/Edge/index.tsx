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

import {Arrow} from 'react-konva';
import {parseEdgePos} from '../utils';

export interface IEdge {
  source: number;
  target: number;
  color: string;
  points: string;
}
interface Props {
  edge: IEdge;
  sourceNode: {x: number; y: number};
  targetNode: {x: number; y: number};
  xScale: (x: number) => number;
  yScale: (y: number) => number;
  nodeRadius: number;
}

const Edge = ({edge, sourceNode, targetNode, xScale, yScale, nodeRadius}: Props): JSX.Element => {
  const {points, color, source, target} = edge;

  const scaledPoints = parseEdgePos({
    pos: points,
    xScale,
    yScale,
    source: [sourceNode.x, sourceNode.y],
    target: [targetNode.x, targetNode.y],
    nodeRadius,
    isSelfLoop: source === target,
  });

  return (
    <Arrow
      points={scaledPoints}
      bezier={true}
      stroke={color}
      strokeWidth={3}
      pointerLength={10}
      pointerWidth={10}
      fill={color}
      opacity={0.8}
    />
  );
};

export default Edge;
