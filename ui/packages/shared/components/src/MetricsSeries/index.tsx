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

import * as d3 from 'd3';

interface MetricsSeriesProps {
  data: any;
  line: d3.Line<[number, number]>;
  color: string;
  strokeWidth: string;
  xScale: (input: number) => number;
  yScale: (input: number) => number;
}

const MetricsSeries = ({data, line, color, strokeWidth}: MetricsSeriesProps): JSX.Element => (
  <g className="line-group">
    <path
      className="line"
      d={line(data.values) || undefined}
      style={{
        stroke: color,
        strokeWidth: strokeWidth,
      }}
    />
  </g>
);

export default MetricsSeries;
