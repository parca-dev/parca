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

import {Icon} from '@iconify/react';
import cx from 'classnames';

interface Props {
  isHalfScreen: boolean;
  isDarkMode: boolean;
}

export const TableActionButtonPlaceholder = (): JSX.Element => {
  return (
    <div className="ml-2 flex w-full flex-col items-start justify-between gap-2 md:flex-row md:items-end">
      <div className="h-[38px] bg-[#f3f3f3] dark:bg-gray-900 animate-pulse w-[152px]"></div>
      <div className="h-[38px] bg-[#f3f3f3] dark:bg-gray-900 animate-pulse w-[110px]"></div>
    </div>
  );
};

const TableHeaderSkeleton = (): JSX.Element => {
  return (
    <div className="font-robotoMono font-bold sticky top-0 bg-gray-50 text-sm dark:bg-gray-800">
      <div className="flex">
        <div className="cursor-pointer p-2 w-[80px]">
          <span className="flex items-center gap-2 justify-end">
            Flat
            <Icon icon="pepicons:triangle-up-filled" />
          </span>
        </div>
        <div className="flex cursor-pointer p-2 w-[150px] justify-end">
          <span className="flex items-center gap-2">
            Cumulative
            <Icon icon="pepicons:triangle-up-filled" />
          </span>
        </div>
        <div className="flex cursor-pointer p-2">
          <span className="flex items-center gap-2 justify-end">
            Name
            <Icon icon="pepicons:triangle-down-filled" />
          </span>
        </div>
      </div>
    </div>
  );
};

const TableBodySkeleton = ({isHalfScreen, isDarkMode}: Props): JSX.Element => (
  <svg
    fill="none"
    height="100%"
    viewBox="0 0 1415 658"
    width={isHalfScreen ? '1455px' : '100%'}
    xmlns="http://www.w3.org/2000/svg"
    preserveAspectRatio="none"
  >
    <defs>
      <linearGradient id="table-data" x1="0%" y1="0%" x2="100%" y2="0%">
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

    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="10.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="10.5" />
    <path d="m.5 28.5 1412 .0003" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="10.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="40.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="40.5" />
    <path d="m.5 58.5 1412 .0003" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="40.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="70.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="70.5" />
    <path d="m.5 88.5 1412 .0003" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="70.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="100.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="100.5" />
    <path d="m.5 118.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="100.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="130.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="130.5" />
    <path d="m.5 148.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="130.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="160.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="160.5" />
    <path d="m.5 178.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="160.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="190.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="190.5" />
    <path d="m.5 208.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="190.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="220.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="220.5" />
    <path d="m.5 238.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="220.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="250.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="250.5" />
    <path d="m.5 268.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="250.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="280.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="280.5" />
    <path d="m.5 298.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="280.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="310.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="310.5" />
    <path d="m.5 328.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="310.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="340.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="340.5" />
    <path d="m.5 358.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="340.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="370.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="370.5" />
    <path d="m.5 388.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="370.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="400.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="400.5" />
    <path d="m.5 418.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="400.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="430.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="430.5" />
    <path d="m.5 448.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="430.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="460.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="460.5" />
    <path d="m.5 478.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="460.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="490.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="490.5" />
    <path d="m.5 508.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="490.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="520.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="520.5" />
    <path d="m.5 538.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="520.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="550.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="550.5" />
    <path d="m.5 568.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <rect fill="url(#table-data)" height="8" rx="4" width="400" x="267.5" y="550.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="39" x="17.5" y="580.5" />
    <rect fill="url(#table-data)" height="8" rx="4" width="71" x="109.5" y="580.5" />
    <path d="m.5 598.5h1412" stroke={cx(isDarkMode ? '#1f2937' : '#e5e7eb')} />
    <g fill="url(#table-data)" id="table-data">
      <rect height="8" rx="4" width="400" x="267.5" y="580.5" />
      <rect height="8" rx="4" width="39" x="17.5" y="610.5" />
      <rect height="8" rx="4" width="71" x="109.5" y="610.5" />
      <rect height="8" rx="4" width="400" x="267.5" y="610.5" />
    </g>
  </svg>
);

const TableSkeleton = ({isHalfScreen, isDarkMode}: Props): JSX.Element => {
  return (
    <>
      <TableHeaderSkeleton />
      <TableBodySkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
    </>
  );
};

export default TableSkeleton;
