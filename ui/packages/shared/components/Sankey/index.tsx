import * as d3 from 'd3';
import sankeyCircular from './sankeyCircular';
import {sankeyJustify, sankeyLeft, sankeyCenter, sankeyRight} from './align';

const NODE_WIDTH = 15;

const SankeyNode = node => {
  const {name, x0, x1, y0, y1, color} = node;

  return (
    <>
      {/* <rect
        x={x0}
        y={y0}
        width={NODE_WIDTH}
        height={NODE_WIDTH}
        rx={5}
        ry={5}
        style={{fill: color, stroke: 'blue'}}
      >
        <title>{name}</title>
      </rect> */}
      <circle
        cx={x0 + NODE_WIDTH / 2}
        cy={y0}
        r={NODE_WIDTH / 2}
        fill={color}
        style={{stroke: 'black', strokeWidth: 4}}
      >
        <title>{name}</title>
      </circle>
      <text x={x1} y={y0} dx="0.2em" dy="1em" textAnchor="start" transform="null">
        {name}
      </text>
    </>
  );
};

const SankeyLink = ({link, color, width, opacity}) => {
  return (
    <>
      <path
        markerEnd={`url(#${link.circular ? 'arrow' : 'arrow'})`}
        markerMid={`url(#${link.circular ? 'mid' : 'mid'})`}
        d={link.path}
        style={{
          fill: 'none',
          strokeOpacity: opacity,
          stroke: color,
          strokeWidth: width,
          //   strokeDasharray: 4,
        }}
      />
      <title>
        {`${link.source.name} → ${link.target.name}\n${
          link.value < 1000 ? `Value: ${link.value.toLocaleString()}` : ''
        }`}
      </title>
    </>
  );
};

const Sankey = ({data, width, height}) => {
  const {nodes, links} = sankeyCircular(NODE_WIDTH)
    .nodePadding(20)
    .nodeAlign(sankeyJustify)
    .nodeId(d => d.name)
    .extent([
      [10, 10],
      [width - 10, height - 10],
    ])(data);

  const color = d3.interpolateWarm;
  const colorScale = d3.scaleLinear().domain([0, nodes.length]).range([0, 1]);
  const linkWidthScale = d3
    .scaleLinear()
    .domain([0, Math.max(...links.map(link => link.value))])
    .range([1, 5]);

  const getNodeColor = node => (node.partOfCycle ? 'red' : 'black');
  const getLinkColor = link => (link.value > 30 ? 'red' : 'black');
  const getLinkWidth = link => {
    return linkWidthScale(link.value);
  };
  const getLinkOpacity = link => 1;

  return (
    <svg width={width} height={height}>
      <defs>
        <marker id="black-head" orient="auto" markerWidth="1" markerHeight="2" refX="0.2" refY="1">
          <path d="M0,0 V2 L1,1 Z" fill="black" fillOpacity=".3" />
        </marker>
        <marker id="red-head" orient="auto" markerWidth="1" markerHeight="2" refX="0.2" refY="1">
          <path d="M0,0 V2 L1,1 Z" fill="red" fillOpacity=".3" />
        </marker>
        <marker id="mid" orient="auto" markerWidth="20" markerHeight="40" refX="6" refY="6">
          <path d="M0,0 V1 L0.5,0.5 Z" fill="blue" fillOpacity="0.7" />
        </marker>
        <marker
          id="arrow"
          orient="auto"
          markerWidth="20"
          markerHeight="20"
          markerUnits="userSpaceOnUse"
          refX="6"
          refY="6"
        >
          <path d="M 0 0 12 6 0 12 3 6" fill="black" fillOpacity="0.7" />
        </marker>
        {/* {links.map(link, i) => (

        )} */}
      </defs>
      <g transform={`translate(10,10)`} style={{mixBlendMode: 'multiply'}}>
        {nodes.map((node, i) => (
          <SankeyNode {...node} color={getNodeColor(node)} key={node.name} />
        ))}
        {links.map((link, i) => (
          <SankeyLink
            opacity={getLinkOpacity(link)}
            link={link}
            color={getLinkColor(link)}
            width={getLinkWidth(link)}
          />
        ))}
      </g>
    </svg>
  );
};

export default Sankey;
