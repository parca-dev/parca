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
  isDarkMode: boolean;
}

const SourceSkeleton = ({isDarkMode}: Props): JSX.Element => (
  <svg
    fill="none"
    height="100%"
    viewBox="0 0 720 603"
    width="100%"
    xmlns="http://www.w3.org/2000/svg"
  >
    <defs>
      <linearGradient id="source" x1="0%" y1="0%" x2="100%" y2="0%">
        <stop offset="0.599964" stopColor={cx(isDarkMode ? '#111827' : '#f3f3f3')} stopOpacity="1">
          <animate
            attributeName="offset"
            values="-2; -2; 1"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="1.59996" stopColor={cx(isDarkMode ? '#1f2937' : '#ecebeb')} stopOpacity="1">
          <animate
            attributeName="offset"
            values="-1; -1; 2"
            keyTimes="0; 0.25; 1"
            dur="2s"
            repeatCount="indefinite"
          ></animate>
        </stop>
        <stop offset="2.59996" stopColor={cx(isDarkMode ? '#111827' : '#f3f3f3')} stopOpacity="1">
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

    <path d="m10 10h36v16h-36z" fill="url(#source)" />
    <path d="m58 10h62v16h-62z" fill="url(#source)" />
    <path d="m132 10h21v16h-21z" fill="url(#source)" />
    <path d="m165 10h42v16h-42z" fill="url(#source)" />

    <g fill="url(#source)">
      <rect height="8" rx="4" width="71" x="165" y="62" />
      <rect height="8" rx="4" width="36" x="10" y="62" />
      <rect height="8" rx="4" width="36" x="58" y="62" />
      <rect height="8" rx="4" width="157" x="165" y="82" />
      <rect height="8" rx="4" width="36" x="10" y="82" />
      <rect height="8" rx="4" width="109" x="165" y="102" />
      <rect height="8" rx="4" width="36" x="10" y="102" />
      <rect height="8" rx="4" width="306" x="165" y="152" />
      <rect height="8" rx="4" width="36" x="10" y="152" />
      <rect height="8" rx="4" width="157" x="195" y="172" />
      <rect height="8" rx="4" width="36" x="10" y="172" />
      <rect height="8" rx="4" width="36" x="10" y="192" />
      <rect height="8" rx="4" width="249" x="165" y="212" />
      <rect height="8" rx="4" width="36" x="10" y="212" />
      <rect height="8" rx="4" width="122" x="165" y="232" />
      <rect height="8" rx="4" width="36" x="10" y="232" />
      <rect height="8" rx="4" width="306" x="165" y="282" />
      <rect height="8" rx="4" width="36" x="58" y="282" />
      <rect height="8" rx="4" width="36" x="10" y="282" />
      <rect height="8" rx="4" width="157" x="195" y="302" />
      <rect height="8" rx="4" width="36" x="10" y="302" />
      <rect height="8" rx="4" width="71" x="195" y="322" />
      <rect height="8" rx="4" width="36" x="10" y="322" />
      <rect height="8" rx="4" width="249" x="165" y="342" />
      <rect height="8" rx="4" width="36" x="10" y="342" />
      <rect height="8" rx="4" width="122" x="165" y="362" />
      <rect height="8" rx="4" width="36" x="58" y="362" />
      <rect height="8" rx="4" width="36" x="10" y="362" />
      <rect height="8" rx="4" width="306" x="165" y="412" />
      <rect height="8" rx="4" width="36" x="58" y="412" />
      <rect height="8" rx="4" width="36" x="10" y="412" />
      <rect height="8" rx="4" width="157" x="195" y="432" />
      <rect height="8" rx="4" width="36" x="10" y="432" />
      <rect height="8" rx="4" width="71" x="195" y="452" />
      <rect height="8" rx="4" width="36" x="10" y="452" />
      <rect height="8" rx="4" width="249" x="165" y="472" />
      <rect height="8" rx="4" width="36" x="10" y="472" />
      <rect height="8" rx="4" width="122" x="165" y="492" />
      <rect height="8" rx="4" width="36" x="10" y="492" />
      <rect height="8" rx="4" width="122" x="225" y="512" />
      <rect height="8" rx="4" width="36" x="10" y="512" />
      <rect height="8" rx="4" width="122" x="195" y="532" />
      <rect height="8" rx="4" width="36" x="10" y="532" />
      <rect height="8" rx="4" width="122" x="165" y="552" />
      <rect height="8" rx="4" width="122" x="215" y="572" />
      <rect height="8" rx="4" width="36" x="10" y="552" />
      <rect height="8" rx="4" width="36" x="10" y="572" />
    </g>
  </svg>
);

export default SourceSkeleton;
