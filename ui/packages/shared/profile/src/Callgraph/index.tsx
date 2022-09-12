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

import {useState, useEffect, useRef} from 'react';
import graphviz from 'graphviz-wasm';
import * as d3 from 'd3';
import {Stage, Layer, Circle, Arrow, Label, Text} from 'react-konva';
import {Callgraph as CallgraphType, CallgraphEdge, CallgraphNode} from '@parca/client';
import {useAppSelector, selectSearchNodeString} from '@parca/store';
import {isSearchMatch} from '@parca/functions';
import {jsonToDot, parseEdgePos} from './utils';
import type {HoveringNode} from '../GraphTooltip';
import Tooltip from '../GraphTooltip';

interface INode {
  id: number;
  x: number;
  y: number;
  data: {id: string};
  functionName: string;
  color: string;
  mouseX?: number;
  mouseY?: number;
  radius: number;
}

interface NodeProps {
  node: INode;
  hoveredNode: INode | null;
  setHoveredNode: (node: INode | null) => void;
  nodeRadius: number;
  currentSearchString: string;
}

interface IEdge {
  source: number;
  target: number;
  color: string;
  opacity: number;
  points: string;
  nodeRadius: number;
}
interface EdgeProps {
  edge: IEdge;
  sourceNode: {x: number; y: number};
  targetNode: {x: number; y: number};
  xScale: (x: number) => number;
  yScale: (y: number) => number;
  nodeRadius: number;
}
interface Props {
  graph: CallgraphType;
  sampleUnit: string;
  width: number;
  colorRange: [string, string];
}

interface graphvizObject extends CallgraphNode {
  _gvid: number;
  name: string;
  pos: string;
  functionName: string;
}

interface graphvizEdge extends CallgraphEdge {
  _gvid: number;
  tail: number;
  head: number;
  pos: string;
}

interface graphvizType {
  edges: graphvizEdge[];
  objects: graphvizObject[];
  bb: string;
}

const Node = ({
  node,
  hoveredNode,
  setHoveredNode,
  nodeRadius: defaultRadius,
  currentSearchString,
}: NodeProps): JSX.Element => {
  const {
    data: {id},
    x,
    y,
    color,
    functionName,
  } = node;

  const hoverRadius = defaultRadius + 3;
  const isHovered = Boolean(hoveredNode) && hoveredNode?.data.id === id;

  const isCurrentSearchMatch = isSearchMatch(currentSearchString, functionName);

  return (
    <Label x={+x} y={+y}>
      <Circle
        draggable
        radius={isHovered || isCurrentSearchMatch ? hoverRadius : defaultRadius}
        fill={isCurrentSearchMatch ? 'red' : color}
        onMouseOver={() => {
          setHoveredNode({...node, mouseX: x, mouseY: y});
        }}
        onMouseOut={() => {
          setHoveredNode(null);
        }}
      />
      <Text
        text={functionName.substring(0, 1)}
        fontSize={16}
        fill="white"
        width={defaultRadius}
        height={defaultRadius}
        x={-defaultRadius / 2}
        y={-defaultRadius / 2}
        align="center"
        verticalAlign="middle"
        listening={false}
      />
    </Label>
  );
};

const Edge = ({
  edge,
  sourceNode,
  targetNode,
  xScale,
  yScale,
  nodeRadius,
}: EdgeProps): JSX.Element => {
  const {points, color, source, target, opacity} = edge;

  const scaledPoints = parseEdgePos({
    pos: points,
    xScale,
    yScale,
    source: [sourceNode.x, sourceNode.y],
    target: [targetNode.x, targetNode.y],
    nodeRadius,
    isSelfLoop: source === target,
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
      opacity={opacity}
    />
  );
};

const Callgraph = ({graph, sampleUnit, width, colorRange}: Props): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [graphData, setGraphData] = useState<any>(null);
  const [hoveredNode, setHoveredNode] = useState<INode | null>(null);
  const {nodes: rawNodes, cumulative: total} = graph;
  const nodeRadius = 12;
  const currentSearchString = useAppSelector(selectSearchNodeString);

  useEffect(() => {
    const getDataWithPositions = async (): Promise<void> => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({graph, width, nodeRadius});

      // 2. Use Graphviz-WASM to translate the 'dot' graph to a 'JSON' graph
      await graphviz.loadWASM(); // need to load the WASM instance and wait for it

      const jsonGraph = graphviz.layout(dataAsDot, 'json', 'dot');

      setGraphData(jsonGraph);
    };

    if (width !== null) {
      void getDataWithPositions();
    }
  }, [graph, width]);

  // 3. Render the graph with calculated layout in Canvas container
  if (width == null || graphData == null) return <></>;

  const height = (2 * width) / 3;
  const {objects, edges: gvizEdges, bb: boundingBox} = JSON.parse(graphData);

  const cumulatives: string[] = objects
    .filter(node => node !== undefined)
    .map(node => node.cumulative);
  if (cumulatives.length === 0) {
    cumulatives.push('0');
  }

  const valueRange = (d3.extent(cumulatives) as [string, string]).map(value => parseInt(value));

  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain(valueRange)
    .range(colorRange);
  const colorOpacityScale = d3.scaleSequentialLog().domain(valueRange).range([0.2, 1]);
  const nodeRadiusScale = d3
    .scaleLog()
    .domain(valueRange)
    .range([nodeRadius - 2, nodeRadius + 3]);

  const margin = 15;
  const graphBB = boundingBox.split(',');
  const xScale = d3
    .scaleLinear()
    .domain([0, graphBB[2]])
    .range([0, width - 2 * margin]);
  const yScale = d3
    .scaleLinear()
    .domain([0, graphBB[3]])
    .range([0, height - 2 * margin]);

  const nodes: INode[] = objects.map(object => {
    const pos = object.pos.split(',');
    return {
      ...object,
      id: object._gvid,
      x: xScale(parseInt(pos[0])),
      y: yScale(parseInt(pos[1])),
      color: colorScale(Number(object.cumulative)),
      data: rawNodes.find(n => n.id === object.name) ?? {id: 'n0'},
    };
  });

  const edges: IEdge[] = gvizEdges.map(edge => ({
    ...edge,
    source: edge.head,
    target: edge.tail,
    points: edge.pos,
    color: colorRange[1],
    opacity: colorOpacityScale(+edge.cumulative),
    nodeRadius: nodeRadiusScale(+edge.cumulative),
  }));

  return (
    <div className="relative">
      <div className={`w-[${width}px] h-[${height}px]`} ref={containerRef}>
        <Stage width={width} height={height}>
          <Layer offsetX={-margin} offsetY={-margin}>
            {edges.map((edge: IEdge) => {
              const sourceNode = nodes.find(n => n.id === edge.source) ?? {x: 0, y: 0};
              const targetNode = nodes.find(n => n.id === edge.target) ?? {x: 0, y: 0};
              return (
                <Edge
                  key={`edge-${edge.source}-${edge.target}`}
                  edge={edge}
                  xScale={xScale}
                  yScale={yScale}
                  sourceNode={sourceNode}
                  targetNode={targetNode}
                  nodeRadius={edge.nodeRadius}
                />
              );
            })}
            {nodes.map(node => (
              <Node
                key={`node-${node.data.id}`}
                node={node}
                hoveredNode={hoveredNode}
                setHoveredNode={setHoveredNode}
                nodeRadius={node.radius}
                currentSearchString={currentSearchString ?? ''}
              />
            ))}
          </Layer>
        </Stage>
        <Tooltip
          hoveringNode={rawNodes.find(n => n.id === hoveredNode?.data.id) as HoveringNode}
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
