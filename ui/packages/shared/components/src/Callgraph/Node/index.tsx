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

import {Circle} from 'react-konva';

export interface INode {
  x: number;
  y: number;
  data: {id: string};
  color: string;
  mouseX?: number;
  mouseY?: number;
}

interface Props {
  node: INode;
  hoveredNode: INode | null;
  setHoveredNode: (node: INode | null) => void;
  nodeRadius: number;
}

const Node = ({
  node,
  hoveredNode,
  setHoveredNode,
  nodeRadius: defaultRadius,
}: Props): JSX.Element => {
  const {
    data: {id},
    x,
    y,
    color,
  } = node;

  const hoverRadius = defaultRadius + 3;
  const isHovered = Boolean(hoveredNode) && hoveredNode?.data.id === id;

  return (
    <Circle
      x={+x}
      y={+y}
      draggable
      radius={isHovered ? hoverRadius : defaultRadius}
      fill={color}
      onMouseOver={() => {
        setHoveredNode({...node, mouseX: x, mouseY: y});
      }}
      onMouseOut={() => {
        setHoveredNode(null);
      }}
    />
  );
};

export default Node;
