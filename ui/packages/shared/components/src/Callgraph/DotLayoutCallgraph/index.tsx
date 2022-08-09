import {useState, useEffect} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer, Circle, Line, Shape} from 'react-konva';
import {Button} from '@parca/components';

const pixelsToInches = pixels => pixels / 96;

const parseEdgePos = pos => {
  const parts = pos.split(' ');
  const arrow = parts.shift();
  const partsAsArrays = parts.map(part => part.split(','));
  const [start, cp1, cp2, end] = partsAsArrays;
  const arrowEnd = arrow.replace('e,', '').split(',');
  return {start, cp1, cp2, end, arrowEnd};
};

export const jsonToDot = ({graph, width, height}) => {
  const {nodes, edges} = graph;
  const nodesAsStrings = nodes.map(
    node => `${node.id} [cumulative=${node.cumulative} root=${node.id === 'root'}]`
  );
  const edgesAsStrings = edges.map(edge => `${edge.source} -> ${edge.target}`);

  // "BT" will actually render top->bottom in our canvas
  const graphAsDot = `digraph "callgraph" { 
      rankdir="BT"  
      ratio="fill"
      size="${pixelsToInches(width)},${pixelsToInches(height)}!"
      margin=0
      edge [margin=0]
      node [margin=0 shape=circle style=filled]
      ${nodesAsStrings.join(' ')}
      ${edgesAsStrings.join(' ')}
    }`;

  return graphAsDot;
};

const Node = ({node}) => {
  const {x, y, color} = node;
  const defaultRadius = 19;
  const hoverRadius = defaultRadius + 3;
  const [radius, setRadius] = useState(defaultRadius);
  return (
    <Circle
      x={+x}
      y={+y}
      draggable
      radius={radius}
      fill={color}
      onMouseOver={() => {
        setRadius(hoverRadius);
      }}
      onMouseOut={() => {
        setRadius(defaultRadius);
      }}
    />
  );
};

const Edge = ({edge}) => {
  const {
    points: {start, cp1, cp2, end},
    color,
  } = edge;
  const pointsAsNumbers = [start, cp1, cp2, end].map(pos => [+pos[0], +[pos[1]]]);
  const pointsArray = [].concat.apply([], pointsAsNumbers);
  return <Line points={pointsArray} bezier={true} stroke={color} strokeWidth={3} />;
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

const DotLayoutCallgraph = ({data, height, width}) => {
  const [graph, setGraph] = useState<any>(null);
  const [layout, setLayout] = useState<'dot' | 'twopi'>('dot');

  useEffect(() => {
    const getDataWithPositions = async () => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({graph: data, width, height});

      // 2. Use Graphviz-WASM to translate the 'dot' graph to a 'JSON' graph
      await graphviz.loadWASM(); // need to load the WASM instance and wait for it

      const jsonGraph = graphviz.layout(dataAsDot, 'json', layout);

      setGraph(jsonGraph);
    };

    if (width) {
      getDataWithPositions();
    }
  }, [width, layout]);

  // 3. Render the laided out graph in Canvas container
  if (!width || !graph) return <></>;

  const {objects, edges: gvizEdges} = JSON.parse(graph);
  //   @ts-ignore
  const valueRange = d3.extent(
    objects.map(node => parseInt(node.cumulative)).filter(node => node !== undefined)
  ) as [number, number];
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateRdGy)
    .domain([...valueRange])
    .range(['lightgrey', 'red']);

  const nodes = objects.map(object => {
    const pos = object.pos.split(',');
    return {
      ...object,
      id: object._gvid,
      x: parseInt(pos[0]),
      y: parseInt(pos[1]),
      color: colorScale(object.cumulative),
    };
  });

  const edges = gvizEdges.map(edge => ({
    ...edge,
    source: edge.head,
    target: edge.tail,
    points: parseEdgePos(edge.pos),
    color: colorScale(nodes.find(node => node.id === edge.head).cumulative),
  }));

  return (
    <>
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

      <Stage width={width} height={height}>
        <Layer>
          {edges.map(edge => (
            <Edge edge={edge} />
          ))}
          {nodes.map(node => (
            <Node node={node} />
          ))}
          {edges.map(edge => (
            <Arrow edge={edge} />
          ))}
        </Layer>
      </Stage>
    </>
  );
};

export default DotLayoutCallgraph;
