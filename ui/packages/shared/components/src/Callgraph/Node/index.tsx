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

const Node = ({node, hoveredNode, setHoveredNode, nodeRadius: defaultRadius}: Props) => {
  const {
    data: {id},
    x,
    y,
    color,
  } = node;

  const hoverRadius = (defaultRadius as number) + 3;
  const isHovered = Boolean(hoveredNode) && hoveredNode?.data.id === id;

  return (
    <Circle
      x={+x}
      y={+y}
      draggable
      radius={Boolean(isHovered) ? hoverRadius : defaultRadius}
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
