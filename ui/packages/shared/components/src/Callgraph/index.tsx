import {useState, useEffect, useRef} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer, Circle, Arrow} from 'react-konva';
import {Button, GraphTooltip as Tooltip} from '@parca/components';
import {Callgraph as CallgraphType, CallgraphNode, CallgraphEdge} from '@parca/client';
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
}: {
  pos: string;
  xScale?: (number) => void;
  yScale?: (number) => void;
}): number[] => {
  const parts = pos.split(' ');
  const arrow = parts.shift() ?? '';
  const partsAsArrays = parts.map(part => part.split(','));
  const scalePosArray = (posArr): number[] => [+xScale(+posArr[0]), +yScale(+posArr[1])];
  const [start, cp1, cp2, end] = partsAsArrays.map(posArr => scalePosArray(posArr));
  const arrowEnd: number[] = scalePosArray(arrow.replace('e,', '').split(','));

  return [...arrowEnd, ...end, ...cp2, ...cp1, ...start];
};

export const jsonToDot = ({graph, width}) => {
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
      margin=0
      edge [margin=0]
      node [margin=0 shape=circle style=filled]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};

const Edge = ({edge, xScale, yScale}) => {
  const {points, color} = edge;

  const scaledPoints = parseEdgePos({pos: points, xScale, yScale});
  return (
    <Arrow
      points={scaledPoints}
      bezier={true}
      stroke={color}
      strokeWidth={3}
      pointerLength={10}
      pointerWidth={10}
      fill={color}
    />
  );
};

const Node = ({node, hoveredNode, setHoveredNode}) => {
  const {
    data: {id},
    x,
    y,
    color,
    width: nodeWidth,
  } = node;

  const defaultRadius = +nodeWidth;
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

  useEffect(() => {
    const getDataWithPositions = async () => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({graph, width});

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
            {edges.map(edge => (
              <Edge
                key={`edge-${edge.source}-${edge.target}`}
                edge={edge}
                xScale={xScale}
                yScale={yScale}
              />
            ))}
            {nodes.map(node => (
              <Node
                key={`node-${node.data.id}`}
                node={node}
                hoveredNode={hoveredNode}
                setHoveredNode={setHoveredNode}
              />
            ))}
          </Layer>
        </Stage>
      </div>

      {hoveredNode && (
        <Tooltip
          hoveringNode={rawNodes.find(n => n.id === hoveredNode.data.id)}
          unit={sampleUnit}
          total={+total}
          isFixed={false}
          x={hoveredNode.mouseX}
          y={hoveredNode.mouseY}
          contextElement={containerRef.current}
        />
      )}
    </div>
  );
};

export default Callgraph;
