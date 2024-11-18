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

import {Fragment} from 'react';

import * as d3 from 'd3';

import {NumberDuo} from '../utils';

interface Props {
  width: number;
  height: number;
  margin: number;
  bounds: NumberDuo;
}

const alignBeforeAxisCorrection = (val: number): number => {
  if (val < 10000) {
    return -24;
  }
  if (val < 100000) {
    return -28;
  }

  return 0;
};

export const TimelineGuide = ({bounds, width, height, margin}: Props): JSX.Element => {
  const xScale = d3.scaleLinear().domain(bounds).range([0, width]);

  return (
    <div className="relative h-4">
      <div className="absolute" style={{width, height}}>
        <svg style={{width: '100%', height: '100%'}}>
          <g
            className="x axis"
            fill="none"
            fontSize="10"
            textAnchor="middle"
            transform={`translate(0,${height - margin})`}
          >
            {xScale.ticks().map((d, i) => (
              <Fragment key={`${i.toString()}-${d.toString()}`}>
                <g
                  key={`tick-${i}`}
                  className="tick"
                  /* eslint-disable-next-line @typescript-eslint/restrict-template-expressions */
                  transform={`translate(${xScale(d) + alignBeforeAxisCorrection(d)}, ${-height})`}
                >
                  {/* <line y2={6} className="stroke-gray-300 dark:stroke-gray-500" /> */}
                  <text fill="currentColor" dy=".71em" y={9}>
                    {d} ms
                  </text>
                </g>
                <g key={`grid-${i}`}>
                  <line
                    className="stroke-gray-300 dark:stroke-gray-500"
                    x1={xScale(d)}
                    x2={xScale(d)}
                    y1={0}
                    y2={-height + margin}
                  />
                </g>
              </Fragment>
            ))}
            <line
              className="stroke-gray-300 dark:stroke-gray-500"
              x1={0}
              x2={width}
              y1={-height + 1}
              y2={-height + 1}
            />
            <line
              className="stroke-gray-300 dark:stroke-gray-500"
              x1={0}
              x2={width}
              y1={-height + 20}
              y2={-height + 20}
            />
            {/* <g transform={`translate(${(width - 2.5 * margin) / 2}, ${margin / 2})`}>
                <text fill="currentColor" dy=".71em" y={5} className="text-sm">
                    Time
                </text>
            </g> */}
          </g>
        </svg>
      </div>
    </div>
  );
};
