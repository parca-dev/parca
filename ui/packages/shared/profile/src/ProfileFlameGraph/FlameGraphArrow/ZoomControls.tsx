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

import React from 'react';

import {Icon} from '@iconify/react';
import {createPortal} from 'react-dom';

interface ZoomControlsProps {
  zoomLevel: number;
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
  portalRef?: React.RefObject<HTMLDivElement | null>;
}

export const ZoomControls = ({
  zoomLevel,
  zoomIn,
  zoomOut,
  resetZoom,
  portalRef,
}: ZoomControlsProps): React.JSX.Element => {
  const controls = (
    <div className="flex items-center gap-1 rounded-md border border-gray-200 bg-white/90 px-1 py-0.5 shadow-sm backdrop-blur-sm dark:border-gray-600 dark:bg-gray-800/90">
      <button
        onClick={zoomOut}
        disabled={zoomLevel <= 1}
        className="rounded p-1 text-gray-600 hover:bg-gray-100 disabled:opacity-30 dark:text-gray-300 dark:hover:bg-gray-700"
        title="Zoom out"
      >
        <Icon icon="mdi:minus" width={16} height={16} />
      </button>
      <button
        onClick={resetZoom}
        className="min-w-[3rem] px-1 text-center text-xs text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700 rounded"
        title="Reset zoom"
      >
        {Math.round(zoomLevel * 100)}%
      </button>
      <button
        onClick={zoomIn}
        disabled={zoomLevel >= 20}
        className="rounded p-1 text-gray-600 hover:bg-gray-100 disabled:opacity-30 dark:text-gray-300 dark:hover:bg-gray-700"
        title="Zoom in"
      >
        <Icon icon="mdi:plus" width={16} height={16} />
      </button>
    </div>
  );

  if (portalRef?.current != null) {
    return createPortal(controls, portalRef.current);
  }

  return controls;
};
