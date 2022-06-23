import {useEffect, useRef, useState} from 'react';
import * as d3Dag from 'd3-dag';
import * as d3 from 'd3';
import {FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {useContainerDimensions} from '@parca/dynamicsize';
interface Props {
  data: FlamegraphRootNode | undefined;
  width?: number;
  height?: number;
  arrows?: boolean;
}

const isRootNode = data => !data.meta;
const getMethodName = data => data.meta.function?.name;

const NODE_RADIUS = 20;
const ROOT_NODE_WIDTH = 80;

const Callgraph = ({
  data,
  width: customWidth,
  height: customHeight,
  arrows = true,
}: Props): JSX.Element => {
  if (!data) {
    return <div />;
  }

  const {ref: containerRef, dimensions: originalDimensions} = useContainerDimensions();
  const fullWidth = customWidth ?? originalDimensions?.width;
  const fullHeight = customHeight ?? 600;
  const svgRef = useRef(null);
  const defsRef = useRef(null);
  const [dimensions, setDimensions] = useState({width: 0, height: 0});

  useEffect(() => {
    if (svgRef.current) {
      const dag = d3Dag
        .dagHierarchy()
        .decycle(true)
        .children((d: FlamegraphNode) => d.children)(data);

      // When data is a list of links, we will need to use dagConnect instead of dagHierarchy
      //   const dag = d3Dag
      //     .dagConnect()
      //     .decycle(true)
      //     .sourceId((link: Link) => link.source)
      //     .targetId((link: Link) => link.target)(data.links);

      const layout = d3Dag
        .sugiyama() // base layout
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

      // Draw edges
      svgSelection
        .append('g')
        .selectAll('path')
        .data(dag.links())
        .enter()
        .append('path')
        .attr('d', (link: d3Dag.DagLink) => line(link.points.map(point => [point.x, point.y])))
        .attr('fill', 'none')
        .attr('stroke-width', 3)
        .attr('marker-end', link => {
          return arrows ? 'url(#arrow)' : '';
        })
        .attr('stroke', ({source, target, reversed}) => {
          return reversed
            ? colorScale(+source.data.cumulative)
            : colorScale(+target.data.cumulative);
        });

      // Select nodes
      const nodes = svgSelection
        .append('g')
        .selectAll('g')
        .data(dag.descendants() as d3Dag.DagNode<FlamegraphNode>[])
        .enter()
        .append('g')
        .attr('transform', ({x, y}) => `translate(${x}, ${y})`);

      // Plot node circles
      nodes
        .append('circle')
        .attr('r', NODE_RADIUS)
        .attr('fill', n => colorScale(+n.data.cumulative))
        .attr('stroke', n => colorScale(+n.data.cumulative));

      // Add text to nodes
      nodes
        .append('text')
        .text(d => (isRootNode(d.data) ? 'root' : getMethodName(d.data).substring(0, 1)))
        .attr('font-weight', 'bold')
        .attr('font-family', 'sans-serif')
        .attr('text-anchor', 'middle')
        .attr('alignment-baseline', 'middle')
        .attr('fill', 'white');

      // Plot Arrows
      if (arrows) {
        const arrowSize = (NODE_RADIUS * NODE_RADIUS) / 5.0;
        const arrowLen = Math.sqrt((4 * arrowSize) / Math.sqrt(3));
        const arrow = d3.symbol().type(d3.symbolTriangle).size(arrowSize);
        svgSelection
          .append('g')
          .selectAll('path')
          .data(dag.links())
          .enter()
          .append('path')
          .attr('d', arrow)
          .attr('transform', ({source, target, points, reversed}) => {
            //TODO: need to work on properly positioning the arrows for reversed links
            if (reversed) {
              const [start, end] = points.slice().reverse();
              const dx = end.x - start.x;
              const dy = end.y - start.y;
              const scale = (NODE_RADIUS * 1.15) / Math.sqrt(dx * dx + dy * dy);
              const angle = (Math.atan2(-dx, -dy) * 180) / Math.PI + 90;
              return `translate(${start.x + (dx / 2) * scale}, ${
                start.y + (dy / 2) * scale
              }) rotate(${angle})`;
            }
            // slice().reverse() is just used to make a copy of the array and reverse it without modding orig
            const [end, start] = points.slice().reverse();

            // This sets the arrows the node radius (20) + a little bit (3) away from the node center, on the last line segment of the edge. This means that edges that only span ine level will work perfectly, but if the edge bends, this will be a little off.
            const dx = start.x - end.x;
            const dy = start.y - end.y;

            const scale = (NODE_RADIUS * 1.15) / Math.sqrt(dx * dx + dy * dy);
            // This is the angle of the last line segment
            const angle = (Math.atan2(-dy, -dx) * 180) / Math.PI + 90;
            return `translate(${end.x + dx * scale}, ${end.y + dy * scale}) rotate(${angle})`;
          })
          //   .attr('fill-opacity', link => (link.data.isReversed ? 1 : 0))
          .attr('fill', ({source, target, reversed}) =>
            colorScale(reversed ? +source.data.cumulative : +target.data.cumulative)
          )
          //   .attr('stroke', 'white')
          //   .attr('stroke-width', 1.5)
          .attr('stroke-dasharray', `${arrowLen},${arrowLen}`);
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
      >
        <defs ref={defsRef}>
          <marker id="arrow" orient="auto" markerWidth="1" markerHeight="2" refX="0.2" refY="1">
            <path d="M0,0 V2 L1,1 Z" fill="black" fillOpacity=".3" />
          </marker>
        </defs>
      </svg>
    </div>
  );
};

export default Callgraph;
