import ResponsiveSvg from '../ResponsiveSvg';
import {parsePos, getPathDataFromPos} from './utils';

interface Props {
  data: any;
}

const data = {
  nodes: [
    {
      _gvid: 0,
      name: 'a',
      height: '0.5',
      label: '\\N',
      pos: '27,90',
      width: '0.75',
    },
    {
      _gvid: 1,
      name: 'b',
      height: '0.5',
      label: '\\N',
      pos: '27,18',
      width: '0.75',
    },
  ],
  edges: [
    {
      _gvid: 0,
      tail: 0,
      head: 1,
      pos: 'e,27,36.104 27,71.697 27,63.983 27,54.712 27,46.112',
    },
  ],
};

const Content = ({width, height, margin}) => {
  const {nodes, edges} = data;
  const transformedNodes = nodes.map(node => ({
    ...node,
    x: parsePos(node.pos).x,
    y: parsePos(node.pos).y,
  }));

  const transformedLinks = edges.map(edge => {
    const startPos = nodes[edge.head].pos;
    const endPos = nodes[edge.tail].pos;
    return {
      ...edge,
      path: getPathDataFromPos(edge.pos, startPos, endPos),
    };
  });

  if (width && height) {
    return (
      width &&
      height && (
        <g transform={`translate(${margin}, ${margin})`}>
          <rect fill="pink" width={width - margin} height={height - margin} />
          <text transform={`translate(${margin}, ${margin})`}>
            width: {width}, height: {height}{' '}
          </text>

          {transformedNodes.map(node => {
            const {x, y} = node;
            return <rect fill="blue" x={x} y={y} width={20} height={20} />;
          })}
          {transformedLinks.map(link => (
            <path d={link.path} stroke="green" />
          ))}
        </g>
      )
    );
  }

  return <></>;
};

const Xaxis = ({xScale, formatValue = (d: any) => d, translateY, tickCount}) => (
  <g
    className="x axis"
    fill="none"
    fontSize="10"
    textAnchor="middle"
    transform={`translate(0,${translateY})`}
  >
    {xScale.ticks(tickCount).map((d, i) => (
      <g
        key={i}
        className="tick"
        /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
        transform={`translate(${xScale(d)}, 0)`}
      >
        <line y2={6} stroke="currentColor" />
        <text fill="currentColor" dy=".71em" y={9}>
          {formatValue(d)}
        </text>
      </g>
    ))}
  </g>
);

const Yaxis = ({yScale, formatValue = (d: any) => d, tickCount}) => (
  <g className="y axis" textAnchor="end" fontSize="10" fill="none">
    {yScale.ticks(tickCount).map((d, i) => (
      <g
        key={i}
        className="tick"
        /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
        transform={`translate(0, ${yScale(d)})`}
      >
        <line stroke="currentColor" x2={-6} />
        <text fill="currentColor" x={-9} dy={'0.32em'}>
          {formatValue(d)}
        </text>
      </g>
    ))}
  </g>
);

const Callgraph = ({data}: Props): JSX.Element => {
  return (
    <ResponsiveSvg>
      {/* @ts-ignore */}
      <Content margin={20} data={data} />
    </ResponsiveSvg>
  );
};

export default Callgraph;
