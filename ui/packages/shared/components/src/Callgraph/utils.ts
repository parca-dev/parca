import {CallgraphNode, CallgraphEdge} from '@parca/client';

export const pixelsToInches = pixels => pixels / 96;

export const parseEdgePos = ({
  pos,
  xScale = n => n,
  yScale = n => n,
  source = [],
  target = [],
  nodeRadius,
}: {
  pos: string;
  xScale?: (number) => void;
  yScale?: (number) => void;
  source?: number[];
  target?: number[];
  nodeRadius: number;
}): number[] => {
  const parts = pos.split(' ');
  const arrow = parts.shift() ?? '';
  const partsAsArrays = parts.map(part => part.split(','));
  const scalePosArray = (posArr): number[] => [+xScale(+posArr[0]), +yScale(+posArr[1])];
  const [start, cp1, cp2, end] = partsAsArrays.map(posArr => scalePosArray(posArr));
  const arrowEnd: number[] = scalePosArray(arrow.replace('e,', '').split(','));

  const getTargetWithOffset = (target, lastEdgePoint) => {
    const diffX = target[0] - lastEdgePoint[0];
    const diffY = target[1] - lastEdgePoint[1];
    const diffZ = Math.hypot(diffX, diffY);

    const offsetX = (diffX * nodeRadius) / diffZ;
    const offsetY = (diffY * nodeRadius) / diffZ;

    return [target[0] - offsetX, target[1] - offsetY];
  };

  return [...source, ...cp1, ...cp2, ...getTargetWithOffset(target, arrowEnd)];
};

export const jsonToDot = ({graph, width, nodeRadius}) => {
  const {nodes, edges} = graph;

  const objectAsDotAttributes = obj =>
    Object.entries(obj)
      .map(entry => `${entry[0]}="${entry[1]}"`)
      .join(' ');

  const nodesAsStrings = nodes.map((node: CallgraphNode) => {
    const dataAttributes = {
      address: node.meta?.location?.address,
      functionName: node.meta?.function?.name,
      cumulative: node.cumulative,
      root: node.id === 'root',
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
      node [margin=0 shape=circle style=filled width=${nodeRadius}]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};
