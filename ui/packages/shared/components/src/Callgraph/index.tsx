import {useState, useEffect} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer, Circle, Line, Shape} from 'react-konva';
import {Button, GraphTooltipContent as Tooltip} from '@parca/components';
import {Callgraph as CallgraphType} from '@parca/client';
interface Props {
  graph: CallgraphType;
  sampleUnit: string;
  width?: number;
}

const pixelsToInches = pixels => pixels / 96;

const transformPosArr = (posArray, scale) => posArray.map(str => scale(+str));

const parseEdgePos = (pos, sizeScale) => {
  const parts = pos.split(' ');
  const arrow = parts.shift();
  const partsAsArrays = parts.map(part => part.split(','));
  const [start, cp1, cp2, end] = partsAsArrays.map(posArr => transformPosArr(posArr, sizeScale));
  // console.log(start);
  const arrowEnd = arrow.replace('e,', '').split(',');
  return {start, cp1, cp2, end, arrowEnd};
};

export const jsonToDot = ({graph, width}) => {
  const {nodes, edges} = graph;

  const objectAsDotAttributes = obj =>
    Object.entries(obj)
      .map(entry => `${entry[0]}="${entry[1]}"`)
      .join(' ');

  const nodesAsStrings = nodes.map(node => {
    const dataAttributes = {
      address: node.meta.location.address,
      functionName: node.meta.function.name,
      cumulative: 10,
      root: node.id === 'root',
    };

    //TODO: remove hard coded cumulative temporary value
    return `"${node.id}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  const edgesAsStrings = edges.map(edge => {
    const dataAttributes = {
      cumulative: edge.cumulative,
    };
    return `"${edge.source}" -> "${edge.target}" [${objectAsDotAttributes(dataAttributes)}]`;
  });

  // "BT" will actually render top->bottom in our canvas
  const graphAsDot = `digraph "callgraph" { 
      rankdir="BT"  
      ratio="fill"
      size="${pixelsToInches(width)}"
      bb="0 0 ${width} ${width}"
      margin=0
      edge [margin=0]
      node [margin=0 shape=circle style=filled]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};

const Edge = ({edge}) => {
  const {
    points: {start, cp1, cp2, end},
    color,
  } = edge;
  const pointsAsNumbers = [start, cp1, cp2, end].map(pos => [+pos[0], +[pos[1]]]);
  const pointsArray = [].concat.apply([], pointsAsNumbers);
  return <Line points={pointsArray} bezier={true} stroke={color} strokeWidth={1} />;
};

const Arrow = ({edge}) => {
  const {
    points: {end, arrowEnd},
    color,
  } = edge;

  return (
    <Shape
      sceneFunc={(context, shape) => {
        const PI2 = Math.PI * 2;
        const dx = arrowEnd[0] - end[0];
        const dy = arrowEnd[1] - end[1];

        const radians = (Math.atan2(dy, dx) + PI2) % PI2;
        const arrowLength = 15;
        const arrowWidth = 20;

        context.beginPath();
        context.translate(+arrowEnd[0], +arrowEnd[1]);
        context.rotate(radians);
        context.moveTo(0, 0);
        context.lineTo(-arrowLength, arrowWidth / 2);
        context.lineTo(-arrowLength, -arrowWidth / 2);
        context.closePath();
        context.fillStrokeShape(shape);
      }}
      fill={color}
      stroke="white"
      strokeWidth={2}
    />
  );
};

const Callgraph = ({graph, sampleUnit, width}: Props): JSX.Element => {
  // TODO: remove this placeholder value
  const total = 1000;

  const [graphData, setGraphData] = useState<any>(null);
  const [layout, setLayout] = useState<'dot' | 'twopi'>('dot');
  const [hoveredNode, setHoveredNode] = useState<{data: any} | null>(null);
  const {nodes: rawNodes} = graph;

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

  // 3. Render the laided out graph in Canvas container
  if (!width || !graphData) return <></>;

  const {objects, edges: gvizEdges, bb: boundingBox} = JSON.parse(graphData);
  //   @ts-ignore
  const valueRange = d3.extent(
    objects.map(node => parseInt(node.cumulative)).filter(node => node !== undefined)
  ) as [number, number];
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateRdGy)
    .domain([...valueRange])
    .range(['lightgrey', 'red']);

  const graphBB = boundingBox.split(',');
  //TODO: Separate x and y scale!
  const sizeScale = d3.scaleLinear().domain([0, graphBB[2]]).range([0, width]);

  const nodes = objects.map(object => {
    const pos = object.pos.split(',');
    return {
      ...object,
      id: object._gvid,
      x: sizeScale(parseInt(pos[0])),
      y: sizeScale(parseInt(pos[1])),
      color: colorScale(object.cumulative),
      data: rawNodes.find(n => n.id === object.name),
    };
  });

  const edges = gvizEdges.map(edge => ({
    ...edge,
    source: edge.head,
    target: edge.tail,
    points: parseEdgePos(edge.pos, sizeScale),
    color: colorScale(+edge.cumulative),
  }));

  // TODO: need to fix on hover, doesnt recognize mouse out
  // TODO: should make this a memo
  const Node = ({node}) => {
    const {
      data: {id},
      x,
      y,
      color,
      width: nodeWidth,
    } = node;
    console.log(node);
    const defaultRadius = sizeScale(+nodeWidth);
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
          setHoveredNode(node);
        }}
        onMouseOut={() => {
          console.log('left');
          setHoveredNode(null);
        }}
      />
    );
  };

  if (!width) {
    return <div>no width</div>;
  }

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

      <Stage width={width} height={width}>
        <Layer>
          {edges.map(edge => (
            <Edge key={`edge-${edge.source}-${edge.target}`} edge={edge} />
          ))}
          {nodes.map(node => (
            <Node key={`node-${node.data.id}`} node={node} />
          ))}
          {/* {edges.map(edge => (
            <Arrow key={`arrow-${edge.source}-${edge.target}`} edge={edge} />
          ))} */}
        </Layer>
      </Stage>

      {/* TODO: need to reposition tooltip to be next to the node */}
      {hoveredNode && (
        // <div className={`absolute top-[${hoveredNode.x}px] left-[${hoveredNode.y}px]`}>
        <div className={`absolute top-0`}>
          <Tooltip
            hoveringNode={rawNodes.find(n => n.id === hoveredNode.data.id)}
            unit={sampleUnit}
            total={total}
            isFixed={false}
          />
        </div>
      )}
    </div>
  );
};

export default Callgraph;
