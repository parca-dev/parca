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
import cx from 'classnames';

interface Props {
  heightStyle?: string;
  isDarkMode: boolean;
}

const MetricsGraphSkeleton = ({heightStyle, isDarkMode}: Props) => (
  <div className="relative overflow-hidden" style={{height: heightStyle}}>
    <div className="absolute top-0 left-0 w-full h-full bg-shimmer-gradient dark:bg-shimmer-gradient-dark animate-shimmer"></div>
    <svg
      fill="none"
      viewBox="0 0 1435 452"
      width="100%"
      xmlns="http://www.w3.org/2000/svg"
      className="absolute top-0 left-0 z-[1]"
    >
      <defs>
        <linearGradient id="y-chart-shimmer" x1="0%" y1="0%" x2="0%" y2="100%">
          <stop
            offset="0.599964"
            stop-opacity="1"
            stop-color={cx(isDarkMode ? '#1f2937' : '#ebebeb')}
          >
            <animate
              attributeName="offset"
              values="-2; -2; 1"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop
            offset="1.59996"
            stop-opacity="1"
            stop-color={cx(isDarkMode ? '#374151' : '#F6F6F6')}
          >
            <animate
              attributeName="offset"
              values="-1; -1; 2"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop
            offset="2.59996"
            stop-opacity="1"
            stop-color={cx(isDarkMode ? '#1f2937' : '#ebebeb')}
          >
            <animate
              attributeName="offset"
              values="0; 0; 3"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
        </linearGradient>
        <linearGradient id="x-chart-shimmer" x1="0%" y1="0%" x2="100%" y2="0%">
          <stop
            offset="0.599964"
            stop-color={cx(isDarkMode ? '#1f2937' : '#f3f3f3')}
            stop-opacity="1"
          >
            <animate
              attributeName="offset"
              values="-2; -2; 1"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop
            offset="1.59996"
            stop-color={cx(isDarkMode ? '#374151' : '#ecebeb')}
            stop-opacity="1"
          >
            <animate
              attributeName="offset"
              values="-1; -1; 2"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
          <stop
            offset="2.59996"
            stop-color={cx(isDarkMode ? '#1f2937' : '#f3f3f3')}
            stop-opacity="1"
          >
            <animate
              attributeName="offset"
              values="0; 0; 3"
              keyTimes="0; 0.25; 1"
              dur="2s"
              repeatCount="indefinite"
            ></animate>
          </stop>
        </linearGradient>
      </defs>

      <path d="m3.5 146h19v111h-19z" fill="url(#y-chart-shimmer)" />

      <g stroke={cx(isDarkMode ? '#6b7280' : '#ececec')}>
        <path d="m53 19h1378v365h-1378z" />
        <path d="m52.5 139.039h1379" />
        <path d="m52.5 79.8652h1379" />
        <path d="m52.5 198.213h1379" />
        <path d="m52.5 257.387h1379" />
        <path d="m52.5 316.561h1379" />
        <path d="m284.412 18.5v366" />
        <path d="m512.765 18.5v366" />
        <path d="m739.322 18.5v366" />
        <path d="m967.669 18.5v366" />
        <path d="m1196.01 18.5v366" />
      </g>

      <path d="m635 413.5h165v19h-165z" fill="url(#x-chart-shimmer)" />
    </svg>
  </div>
);

export default MetricsGraphSkeleton;
