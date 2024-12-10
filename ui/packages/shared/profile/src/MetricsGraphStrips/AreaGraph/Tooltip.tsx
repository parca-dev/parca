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

import {useEffect, useRef, useState} from 'react';

import {useParcaContext} from '@parca/components';
import {formatDateTimeDownToMS, valueFormatter} from '@parca/utilities';

interface TooltipProps {
  x: number;
  y: number;
  timestamp: Date;
  value: number;
  containerWidth: number;
}

export function Tooltip({x, y, timestamp, value, containerWidth}: TooltipProps): JSX.Element {
  const {timezone} = useParcaContext();
  const tooltipRef = useRef<HTMLDivElement>(null);
  const [tooltipPosition, setTooltipPosition] = useState({x, y});

  const baseOffset = {
    x: 16,
    y: -8,
  };

  useEffect(() => {
    if (tooltipRef.current != null) {
      const tooltipWidth = tooltipRef.current.offsetWidth;

      let newX = x + baseOffset.x;
      let newY = y + baseOffset.y;

      if (newX + tooltipWidth > containerWidth) {
        newX = x - tooltipWidth - baseOffset.x;
      }

      if (newY < 0) {
        newY = y + Math.abs(baseOffset.y);
      }

      setTooltipPosition({x: newX, y: newY});
    }
  }, [x, y, containerWidth, baseOffset.x, baseOffset.y]);

  return (
    <div
      ref={tooltipRef}
      className="absolute bg-white dark:bg-gray-800 rounded-md shadow-lg border border-gray-200 dark:border-gray-700 p-2 text-sm z-20"
      style={{
        left: `${tooltipPosition.x}px`,
        top: `${tooltipPosition.y}px`,
        pointerEvents: 'none',
      }}
    >
      <div className="flex flex-col gap-1">
        <div className="flex gap-1 items-center">
          <div>Timestamp:</div>
          <div className="text-gray-600 dark:text-gray-300">
            {formatDateTimeDownToMS(timestamp, timezone)}
          </div>
        </div>

        <div className="flex gap-1 items-center">
          <div>Value:</div>
          <div className="text-gray-600 dark:text-gray-300">
            {valueFormatter(value, 'nanoseconds', 2)}
          </div>
        </div>
      </div>
    </div>
  );
}
