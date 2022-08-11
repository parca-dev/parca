import {Stylesheet as CSStylesheet} from 'cytoscape';
import * as d3 from 'd3';
import data from './mockData';

const removeSpaces = (str: string): string => str.replaceAll(' ', '');

// @ts-ignore
const cumulativeValueRange = d3.extent(
  data.nodes.map(node => +node.data.cumulative).filter(node => node !== undefined)
) as [number, number];

const colorScale = d3
  .scaleSequentialLog(d3.interpolateRdGy)
  .domain([...cumulativeValueRange])
  .range(['lightgrey', 'red']);

export default [
  {
    selector: 'node',
    css: {
      'font-size': 20,
      width: 15,
      height: 15,
      'text-valign': 'center',
      'text-halign': 'center',
    },
    style: {
      label: 'data(id)',
      'background-color': d => colorScale(d.data().cumulative), //use d3 color scale helpers because cytoscape mapData is problemsome, like no white space allowed
    },
  },
  {
    selector: 'node:selected',
    css: {
      'background-color': 'blue',
    },
  },
  {
    selector: 'edge',
    css: {
      width: 2,
      'line-fill': 'linear-gradient', //filling style of the edge’s line; may be solid (default), linear-gradient (source to target), or radial-gradient (midpoint outwards)
      'line-gradient-stop-colors': d => {
        // const startColor = colorScale(d.source().data().cumulative);
        const startColor = 'lightgrey';
        const endColor = colorScale(d.target().data().cumulative);
        return `${removeSpaces(startColor)} ${removeSpaces(endColor)}`;
      },
      opacity: 0.5,
      'line-style': 'solid', // "solid", "dashed", "dotted"
      'target-arrow-color': d => colorScale(d.target().data().cumulative),
      'target-arrow-shape': 'triangle',
      'arrow-scale': 1.5,
      'curve-style': 'unbundled-bezier',
      'control-point-weight': '0.5', // '0': curve towards source node, '1': towards target node
      'control-point-distance': d => {
        const source = d.source().position();
        const target = d.target().position();
        const isHorizontalLine = source.y === target.y;

        const isTopToBottom = target.x <= source.x;

        if (isHorizontalLine) {
          return isTopToBottom ? '-50px' : '50px';
        }

        return isTopToBottom ? '30px' : '-30px';
      },
      'font-size': 3,
    },
  },
  {
    selector: 'edge:selected',
    css: {
      width: 4,
      'line-color': 'lightblue',
    },
  },
] as CSStylesheet[];
