interface MetricsCircleProps {
  cx: number
  cy: number
  radius?: number
}

const defaultRadius = 3

const MetricsCircle = ({ cx, cy, radius }: MetricsCircleProps): JSX.Element => (
  <g className="circle">
    <circle
      cx={cx}
      cy={cy}
      r={radius !== undefined ? radius : defaultRadius}
    >
    </circle>
  </g>
)

export default MetricsCircle
