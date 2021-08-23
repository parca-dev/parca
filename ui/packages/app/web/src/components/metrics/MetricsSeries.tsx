import * as d3 from 'd3'

interface MetricsSeriesProps {
  data: any
  line: d3.line<[number, number]>
  color: string
  strokeWidth: string
  xScale: (input: number) => number
  yScale: (input: number) => number
}

const MetricsSeries = ({ data, line, color, strokeWidth }: MetricsSeriesProps): JSX.Element => (
  <g className="line-group">
    <path
      className="line"
      d={line(data.values)}
      style={{
        stroke: color,
        strokeWidth: strokeWidth
      }}/>
  </g>
)

export default MetricsSeries
