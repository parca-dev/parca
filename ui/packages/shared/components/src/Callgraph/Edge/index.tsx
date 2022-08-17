import {Arrow} from 'react-konva';
import {parseEdgePos} from '../utils';

interface Props {
  edge: {points: string; color: string};
  sourceNode: {x: number; y: number};
  targetNode: {x: number; y: number};
  xScale: (x: number) => number;
  yScale: (y: number) => number;
  nodeRadius: number;
}

const Edge = ({edge, sourceNode, targetNode, xScale, yScale, nodeRadius}: Props) => {
  const {points, color} = edge;

  const scaledPoints = parseEdgePos({
    pos: points,
    xScale,
    yScale,
    source: [sourceNode.x, sourceNode.y],
    target: [targetNode.x, targetNode.y],
    nodeRadius,
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
