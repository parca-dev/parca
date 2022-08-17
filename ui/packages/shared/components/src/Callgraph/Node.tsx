import {Circle} from 'react-konva';

const Node = ({node, hoveredNode, setHoveredNode, nodeRadius: defaultRadius}) => {
  const {
    data: {id},
    x,
    y,
    color,
  } = node;

  const hoverRadius = defaultRadius + 3;
  const isHovered = hoveredNode && hoveredNode.data.id === id;

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
