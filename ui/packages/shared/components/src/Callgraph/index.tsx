import {useState, useEffect, useRef} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer, Circle, Arrow} from 'react-konva';
import {Button, GraphTooltip as Tooltip} from '@parca/components';
import {Callgraph as CallgraphType, CallgraphNode, CallgraphEdge} from '@parca/client';

// TODO: Fix self-loops
interface Props {
  graph: CallgraphType;
  sampleUnit: string;
  width?: number;
}

interface HoveredNode {
  mouseX: number;
  mouseY: number;
  data: any;
}

const pixelsToInches = pixels => pixels / 96;

const parseEdgePos = ({
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

const Edge = ({edge, sourceNode, targetNode, xScale, yScale, nodeRadius}) => {
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
      onMouseOver={() => console.log({edge, scaledPoints, sourceNode, targetNode})}
    />
  );
};

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

const Callgraph = ({graph, sampleUnit, width}: Props): JSX.Element => {
  const containerRef = useRef<Element>(null);
  const [graphData, setGraphData] = useState<any>(null);
  const [layout, setLayout] = useState<'dot' | 'twopi'>('dot');
  const [hoveredNode, setHoveredNode] = useState<HoveredNode | null>(null);
  const {nodes: rawNodes, cumulative: total} = graph;
  const nodeRadius = 15;

  useEffect(() => {
    const getDataWithPositions = async () => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({graph, width, nodeRadius});

      // 2. Use Graphviz-WASM to translate the 'dot' graph to a 'JSON' graph
      await graphviz.loadWASM(); // need to load the WASM instance and wait for it

      const jsonGraph = graphviz.layout(dataAsDot, 'json', layout);

      setGraphData(jsonGraph);
    };

    if (width) {
      getDataWithPositions();
    }
  }, [width, layout]);

  // 3. Render the graph with calculated layout in Canvas container
  if (!width || !graphData) return <></>;

  const height = width;
  const {objects, edges: gvizEdges, bb: boundingBox} = JSON.parse(graphData);

  //   @ts-expect-error
  const valueRange = d3.extent(
    objects.map(node => parseInt(node.cumulative)).filter(node => node !== undefined)
  ) as [number, number];
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateRdGy)
    .domain([...valueRange])
    .range(['lightgrey', 'red']);
  const graphBB = boundingBox.split(',');
  const xScale = d3.scaleLinear().domain([0, graphBB[2]]).range([0, width]);
  const yScale = d3.scaleLinear().domain([0, graphBB[3]]).range([0, height]);

  const nodes = objects.map(object => {
    const pos = object.pos.split(',');
    return {
      ...object,
      id: object._gvid,
      x: xScale(parseInt(pos[0])),
      y: yScale(parseInt(pos[1])),
      color: colorScale(object.cumulative),
      data: rawNodes.find(n => n.id === object.name),
    };
  });

  const edges = gvizEdges.map(edge => ({
    ...edge,
    source: edge.head,
    target: edge.tail,
    points: edge.pos,
    color: colorScale(+edge.cumulative),
  }));

  return (
    <div className="relative">
      <div className="flex">
        <Button
          variant={`${layout === 'dot' ? 'primary' : 'neutral'}`}
          className="items-center rounded-tr-none rounded-br-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
          onClick={() => setLayout(layout === 'dot' ? 'twopi' : 'dot')}
        >
          "Dot" layout
        </Button>
        <Button
          variant={`${layout === 'twopi' ? 'primary' : 'neutral'}`}
          className="items-center rounded-tl-none rounded-bl-none w-auto px-8 whitespace-nowrap text-ellipsis no-outline-on-buttons"
          onClick={() => setLayout(layout === 'dot' ? 'twopi' : 'dot')}
        >
          "Twopi" layout
        </Button>
      </div>

      {/* @ts-expect-error */}
      <div className={`w-[${width}px] h-[${height}px]`} ref={containerRef}>
        <Stage width={width} height={height}>
          <Layer>
            {edges.map(edge => {
              const sourceNode = nodes.find(n => n.id === edge.source);
              const targetNode = nodes.find(n => n.id === edge.target);
              return (
                <Edge
                  key={`edge-${edge.source}-${edge.target}`}
                  edge={edge}
                  xScale={xScale}
                  yScale={yScale}
                  sourceNode={sourceNode}
                  targetNode={targetNode}
                  nodeRadius={nodeRadius}
                />
              );
            })}
            {nodes.map(node => (
              <Node
                key={`node-${node.data.id}`}
                node={node}
                hoveredNode={hoveredNode}
                setHoveredNode={setHoveredNode}
                nodeRadius={nodeRadius}
              />
            ))}
          </Layer>
        </Stage>
        <Tooltip
          hoveringNode={rawNodes.find(n => n.id === hoveredNode?.data.id) ?? null}
          unit={sampleUnit}
          total={+total}
          isFixed={false}
          x={hoveredNode?.mouseX ?? 0}
          y={hoveredNode?.mouseY ?? 0}
          contextElement={containerRef.current}
        />
      </div>
    </div>
  );
};

export default Callgraph;
