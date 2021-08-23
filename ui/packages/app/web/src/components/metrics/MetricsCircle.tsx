interface MetricsCircleProps {
  cx: number
  cy: number
  radius?: number
}

const defaultOpacity = '0.85'
const defaultRadius = 3

const MetricsCircle = ({ cx, cy, radius }: MetricsCircleProps): JSX.Element => (
  <g className="circle">
    <circle
      cx={cx}
      cy={cy}
      r={radius !== undefined ? radius : defaultRadius}
      style={{ opacity: defaultOpacity }}
    >
    </circle>
  </g>
)

export default MetricsCircle
