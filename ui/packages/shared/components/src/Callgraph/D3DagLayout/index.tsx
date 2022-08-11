import React, {useEffect, useRef, useState} from 'react';
import * as d3Dag from 'd3-dag';
import * as d3 from 'd3';
import {pointer} from 'd3-selection';
import {FlamegraphNode, FlamegraphRootNode, Flamegraph} from '@parca/client';
import {useContainerDimensions} from '@parca/dynamicsize';
import {GraphTooltip} from '@parca/components';

// TODO: REMOVE THESE NOTES
// 1. Data is hierarchical, filter nodes to get unique nodes, remove duplicates, then take a look at the links
// for those that had a node removed, replace their "source" position, with the position of the original node
// and make these paths go outside the graph entirely
// 2. Use d3-dag to define "reverse" links. But this changes the entire layout of the dag. Modify the position of the root node, pin to the top of the screen.
// 3. (CHOSEN SOLUTION) Define "reverse" links on the BE/FE. Data should be passed in as a list of nodes, links, and reverseLinks.
// If the backend isn't capable of differentiating between links and reveresed links, than this can be done on the frontend.
// Render the links and arrows slightly differently for these reversed links.

// TODO: Use real data instead of mock data

interface Link {
  source: string;
  target: string;
}
interface Props {
  graph: Flamegraph;
  width?: number;
  height?: number;
  arrows?: boolean;
}

interface EventTargetWithData extends EventTarget {
  __data__: any;
}

const NODE_RADIUS = 20;

const D3DagGraph = ({
  graph,
  width: customWidth,
  height: customHeight,
  arrows = true,
}: Props): JSX.Element => {
  // if data.links, we must use dagConnect method (link data)
  // if root, we must use dagHierarchy method (hierarchical data)
  // @ts-ignore
  const {data, root, total, unit} = graph;

  if (!root && !data) {
    return <div />;
  }

  const [hoveringNode, setHoveringNode] = useState<FlamegraphNode | FlamegraphRootNode | null>(
    null
  );
  const [pos, setPos] = useState([0, 0]);
  const [contextNode, setContextNode] = useState<Element | null>(null);

  const {ref: containerRef, dimensions: originalDimensions} = useContainerDimensions();
  const fullWidth = customWidth ?? originalDimensions?.width;
  const fullHeight = customHeight ?? 600;
  const svgRef = useRef(null);
  const defsRef = useRef(null);
  const [dimensions, setDimensions] = useState({width: 0, height: 0});

  const onMouseOver = (e: React.MouseEvent<SVGCircleElement>): void => {
    setPos(pointer(e, svgRef.current));
    setContextNode(e.target as Element);
    setHoveringNode((e.target as EventTargetWithData).__data__.data);
  };

  const onMouseOut = (e: React.MouseEvent<SVGCircleElement>): void => {
    setContextNode(null);
    setHoveringNode(null);
  };

  const generateDagFromLinksData = data => {
    // When data is a list of links, we will need to use dagConnect instead of dagHierarchy
    const dag = d3Dag
      .dagConnect()
      .decycle(true)
      .nodeDatum(id => data.nodes[id])
      .sourceId((link: Link) => link.source)
      .targetId((link: Link) => link.target)(data.links);

    const layout = d3Dag
      .sugiyama() // base layout
      .layering(d3Dag.layeringCoffmanGraham())
      .coord(d3Dag.coordGreedy())
      .decross(d3Dag.decrossTwoLayer()) // heuristic to minimize number of crossings (not optimal, but opt is too expensive/causes crashes)
      .nodeSize(node => [(node ? 3.6 : 0.25) * NODE_RADIUS, 3 * NODE_RADIUS]); // set node size instead of constraining to fit

    const {width, height} = layout(dag as any);
    setDimensions({width, height});
    const svgSelection = d3.select(svgRef.current);

    const cumulativeValueRange = d3.extent(
      dag
        .descendants()
        .map(node => +node.data.cumulative)
        .filter(node => node !== undefined)
    ) as [number, number];
    const colorScale = d3
      .scaleSequentialLog(d3.interpolateRdGy)
      .domain([...cumulativeValueRange])
      .range(['lightgrey', 'red']);

    // Define how to draw edges
    const line = d3
      .line()
      .curve(d3.curveCatmullRom)
      .x((d: any) => d[0])
      .y((d: any) => d[1]);

    const reversedLine = d3
      .line()
      .curve(d3.curveCatmullRom)
      .x((d: any) => d[0])
      .y((d: any) => d[1]);

    const reversedLinks = data.reversedLinks.map(({source, target}) => {
      const sourceNode = dag.descendants().find(node => node.data.id === source);
      const targetNode = dag.descendants().find(node => node.data.id === target);
      return {
        source: sourceNode,
        target: targetNode,
        reversed: true,
      };
    });

    // Define arrowheads
    svgSelection
      .append('svg:defs')
      .selectAll('marker')
      .data([...dag.links(), ...reversedLinks])
      .enter()
      .append('svg:marker')
      .attr('id', ({target}) => {
        console.log(target.data.id);
        return `arrow-${target.data.id}`;
      })
      .attr('refX', 8.5)
      .attr('refY', 2)
      .attr('markerWidth', 50)
      .attr('markerHeight', 50)
      .attr('orient', 'auto')
      .append('path')
      .attr('d', 'M0,0 V4 L2,2 Z')
      .style('fill', ({target}) => colorScale(+target.data.cumulative));

    console.log({links: dag.links(), reversedLinks});

    // Draw edges
    svgSelection
      .append('g')
      .selectAll('path')
      .data(dag.links().filter(link => !link.reversed))
      .enter()
      .append('path')
      .attr('d', ({points}: d3Dag.DagLink) => line(points.map(point => [point.x, point.y])))
      .attr('fill', 'none')
      .attr('stroke-width', 3)
      .attr('stroke-opacity', 0.7)
      .style('pointer-events', 'none')
      .attr('marker-end', ({target}) => (arrows ? `url(#arrow-${target.data.id})` : ''))
      .attr('stroke', ({target}) => colorScale(+target.data.cumulative));

    // Draw REVERSE edges
    svgSelection
      .append('g')
      .selectAll('path')
      .data(reversedLinks)
      .enter()
      .append('path')
      .attr('d', ({source, target}) => {
        const points = [
          {x: source.x, y: source.y},
          {x: target.x, y: target.y},
        ];

        // Sets the link orientation and width based on the vertical distance between the source and target node
        const horizontalSpace = source.y - target.y;

        // Scales the link
        const scaledSpace =
          (horizontalSpace > 0 ? horizontalSpace + NODE_RADIUS : horizontalSpace - NODE_RADIUS) *
          0.3;

        const additionalPoints = [
          {x: source.x + Math.max(scaledSpace, NODE_RADIUS), y: source.y - NODE_RADIUS},
          {x: target.x + Math.max(scaledSpace, NODE_RADIUS), y: target.y + NODE_RADIUS},
        ];

        const newPoints = [points[0], additionalPoints[0], additionalPoints[1], points[1]];

        return reversedLine(newPoints.map(point => [point.x, point.y]));
      })
      .attr('fill', 'none')
      .attr('stroke-width', 3)
      .attr('stroke-opacity', 0.7)
      .style('pointer-events', 'none')
      .attr('marker-end', ({target}) => (arrows ? `url(#arrow-${target.data.id})` : ''))
      .attr('stroke', ({target}) => colorScale(+target.data.cumulative));

    // Select nodes
    const nodes = svgSelection
      .append('g')
      .selectAll('g')
      .data(dag.descendants() as d3Dag.DagNode<FlamegraphNode>[])
      .enter()
      .append('g')
      // TODO: pin the root node to the top of the svg
      .attr('transform', ({x, y, data}) =>
        // @ts-ignore
        data.id === 'root' ? `translate(${x}, ${y})` : `translate(${x}, ${y})`
      );

    // Plot node circles
    nodes
      .append('circle')
      .attr('r', NODE_RADIUS)
      .attr('fill', n => colorScale(+n.data.cumulative))
      .attr('stroke', n => colorScale(+n.data.cumulative))
      .on('mouseover', onMouseOver)
      .on('mouseout', onMouseOut);

    // Plot Arrows
    // if (arrows) {
    //   const arrowSize = (NODE_RADIUS * NODE_RADIUS) / 5.0;
    //   const arrowLen = Math.sqrt((4 * arrowSize) / Math.sqrt(3));
    //   const arrow = d3.symbol().type(d3.symbolTriangle).size(arrowSize);

    //   svgSelection
    //     .append('g')
    //     .selectAll('path')
    //     .data([...dag.links(), ...reversedLinks])
    //     .enter()
    //     .append('path')
    //     .attr('d', arrow)
    //     .attr('transform', ({source, target, sourceNode, targetNode, points, reversed}) => {
    //       let start = source;
    //       let end = target;
    //       // sets the arrow the "node radius + a little bit" away from the node center, on the last link line segment
    //       let dx = start.x - end.x;
    //       let dy = start.y - end.y;
    //       // This is the angle of the last line segment
    //       let angle = (Math.atan2(-dy, -dx) * 180) / Math.PI + 90;
    //       let scale = (NODE_RADIUS * 1.15) / Math.sqrt(dx * dx + dy * dy);
    //       let shift = {x: end.x + dx * scale, y: end.y + dy * scale};

    //       if (reversed) {
    //         // Reversed links enter the node at a 90 degree angle horizontally
    //         angle = (Math.atan2(-dy, -dx) * 180) / Math.PI;
    //         shift = {x: end.x + NODE_RADIUS + dx * scale, y: end.y - NODE_RADIUS + dy * scale};
    //       }

    //       return `translate(${shift.x}, ${shift.y}) rotate(${angle})`;
    //     })
    //     .attr('fill', ({target}) => colorScale(+target.data.cumulative))
    //     .attr('stroke', 'white')
    //     .attr('stroke-width', 0.5)
    //     .attr('stroke-opacity', 0.5)
    //     .attr('stroke-dasharray', `${arrowLen},${arrowLen}`);
    // }
  };

  useEffect(() => {
    if (svgRef.current) {
      if (data.links) {
        generateDagFromLinksData(data);
      }
    }
  }, [svgRef.current]);

  return (
    <div ref={containerRef}>
      <svg
        ref={svgRef}
        width={fullWidth}
        height={fullHeight}
        viewBox={`0 0 ${dimensions.width} ${dimensions.height}`}
      />
      <GraphTooltip
        unit={unit}
        total={parseFloat(total)}
        x={pos[0]}
        y={pos[1]}
        hoveringNode={hoveringNode}
        contextElement={contextNode}
        virtualContextElement={false}
      />
    </div>
  );
};

export default D3DagGraph;
