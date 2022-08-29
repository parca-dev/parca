/* eslint-disable */
import {useState, useEffect, useRef} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer} from 'react-konva';
import {GraphTooltip as Tooltip} from '@parca/components';
import {Callgraph as CallgraphType} from '@parca/client';
import {jsonToDot} from './utils';
import Node, {INode} from './Node';
import Edge from './Edge';
interface Props {
  graph: CallgraphType;
  sampleUnit: string;
  width: number;
}

const Callgraph = ({graph, sampleUnit, width}: Props): JSX.Element => {
  const containerRef = useRef<Element>(null);
  const [graphData, setGraphData] = useState<any>(null);
  const [hoveredNode, setHoveredNode] = useState<INode | null>(null);
  const {nodes: rawNodes, cumulative: total} = graph;
  const nodeRadius = 12;

  useEffect(() => {
    const getDataWithPositions = async () => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({graph, width, nodeRadius});

      // 2. Use Graphviz-WASM to translate the 'dot' graph to a 'JSON' graph
      await graphviz.loadWASM(); // need to load the WASM instance and wait for it

      const jsonGraph = graphviz.layout(dataAsDot, 'json', 'dot');

      setGraphData(jsonGraph);
    };

    if (width) {
      getDataWithPositions();
    }
  }, [width]);

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
