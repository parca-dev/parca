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
import {Stage, Layer, Rect, Arrow, Text, Label} from 'react-konva';
import {CallgraphNode, CallgraphEdge, Callgraph as CallgraphType} from '@parca/client';
import {jsonToDot, getCurvePoints} from './utils';
import type {HoveringNode} from '../GraphTooltip';
import {useAppSelector, selectSearchNodeString} from '@parca/store';
import {isSearchMatch} from '@parca/functions';
import Tooltip from '../GraphTooltip';
import {DEFAULT_NODE_HEIGHT, GRAPH_MARGIN} from './constants';

interface NodeProps {
  node: INode;
  hoveredNode: INode | null;
  setHoveredNode: (node: INode | null) => void;
}
interface EdgeProps {
  edge: GraphvizEdge;
  sourceNode: {x: number; y: number};
  targetNode: {x: number; y: number};
  xScale: (x: number) => number;
  yScale: (y: number) => number;
}
interface Props {
  graph: CallgraphType;
  sampleUnit: string;
  width: number;
  colorRange: [string, string];
}

interface GraphvizNode extends CallgraphNode {
  _gvid: number;
  name: string;
  pos: string;
  functionName: string;
  color: string;
  width: string | number;
  height: string | number;
}

interface INode extends GraphvizNode {
  x: number;
  y: number;
  data: {id: string};
  mouseX?: number;
  mouseY?: number;
}

interface GraphvizEdge extends CallgraphEdge {
  _gvid: number;
  tail: number;
  head: number;
  pos: string;
  color: string;
  opacity: string;
  boxHeight: number;
}

interface GraphvizType {
  edges: GraphvizEdge[];
  objects: GraphvizNode[];
  bb: string;
}

const Node = ({node, hoveredNode, setHoveredNode}: NodeProps): JSX.Element => {
  const {
    data: {id},
    x,
    y,
    color,
    functionName,
    width: widthString,
    height: heightString,
  } = node;
  const currentSearchString = useAppSelector(selectSearchNodeString);
  const isCurrentSearchMatch = isSearchMatch(currentSearchString, functionName);
  const isHovered = Boolean(hoveredNode) && hoveredNode?.data.id === id;
  const width = Number(widthString);
  const height = Number(heightString);
  const textPadding = 6;

  return (
    <Label x={x - width / 2} y={y - height / 2}>
      <Rect
        width={width}
        height={height}
        fill={color}
        cornerRadius={3}
        stroke={isHovered ? 'black' : color}
        strokeWidth={2}
        onMouseOver={() => {
          setHoveredNode({...node, mouseX: x, mouseY: y});
        }}
        onMouseOut={() => {
          setHoveredNode(null);
        }}
      />
      {width > DEFAULT_NODE_HEIGHT + 10 && (
        <Text
          text={functionName}
          fontSize={10}
          fill="white"
          width={width - textPadding}
          height={height - textPadding}
          x={textPadding / 2}
          y={textPadding / 2}
          align="center"
          verticalAlign="middle"
          listening={false}
        />
      )}
    </Label>
  );
};

const Edge = ({edge, sourceNode, targetNode, xScale, yScale}: EdgeProps): JSX.Element => {
  const {pos, color, head, tail, opacity, boxHeight} = edge;

  const points = getCurvePoints({
    pos,
    xScale,
    yScale,
    source: [sourceNode.x, sourceNode.y],
    target: [targetNode.x, targetNode.y],
    offset: boxHeight / 2,
    isSelfLoop: head === tail,
  });

  return (
    <Arrow
      points={points}
      bezier={true}
      stroke={color}
      strokeWidth={3}
      pointerLength={10}
      pointerWidth={10}
      fill={color}
      opacity={Number(opacity)}
    />
  );
};

const Callgraph = ({graph, sampleUnit, width, colorRange}: Props): JSX.Element => {
  const containerRef = useRef<HTMLDivElement>(null);
  const [graphData, setGraphData] = useState<any>(null);
  const [hoveredNode, setHoveredNode] = useState<INode | null>(null);
  const {nodes: rawNodes, cumulative: total} = graph;

  useEffect(() => {
    const getDataWithPositions = async (): Promise<void> => {
      // 1. Translate JSON to 'dot' graph string
      const dataAsDot = jsonToDot({
        graph,
        width,
        colorRange,
      });

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
  const {objects: gvizNodes, edges, bb: boundingBox} = JSON.parse(graphData) as GraphvizType;

  const graphBB = boundingBox.split(',');
  const bbWidth = Number(graphBB[2]);
  const bbHeight = Number(graphBB[3]);
  const height = (width * bbHeight) / bbWidth;
  const xScale = d3
    .scaleLinear()
    .domain([0, bbWidth])
    .range([0, width - 2 * GRAPH_MARGIN]);
  const yScale = d3
    .scaleLinear()
    .domain([0, bbHeight])
    .range([0, height - 2 * GRAPH_MARGIN]);

  const nodes: INode[] = gvizNodes.map((node: GraphvizNode, i) => {
    const [x, y] = node.pos.split(',');
    return {
      ...node,
      x: xScale(Number(x)),
      y: yScale(Number(y)),
      data: rawNodes.find(n => n.id === node.name) ?? {id: 'n0'},
    };
  });

  return (
    <div className="relative">
      <div className={`w-[${width}px] h-[${height}px]`} ref={containerRef}>
        <Stage width={width} height={height}>
          <Layer offsetX={-GRAPH_MARGIN} offsetY={-GRAPH_MARGIN}>
            {edges.map((edge: GraphvizEdge) => {
              // 'tail' in graphviz-wasm means 'source' and 'head' means 'target'
              const sourceNode = nodes.find(n => n._gvid === edge.tail) ?? {x: 0, y: 0};
              const targetNode = nodes.find(n => n._gvid === edge.head) ?? {x: 0, y: 0};
              return (
                <Edge
                  key={`edge-${edge.tail}-${edge.head}`}
                  edge={edge}
                  xScale={xScale}
                  yScale={yScale}
                  sourceNode={sourceNode}
                  targetNode={targetNode}
                />
              );
            })}
            {nodes.map(node => (
              <Node
                key={`node-${node._gvid}`}
                node={node}
                hoveredNode={hoveredNode}
                setHoveredNode={setHoveredNode}
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
