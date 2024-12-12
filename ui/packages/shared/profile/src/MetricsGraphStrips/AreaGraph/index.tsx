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

import {useEffect, useMemo, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import * as d3 from 'd3';

import {NumberDuo} from '../../utils';
import {Tooltip} from './Tooltip';

export interface DataPoint {
  timestamp: number;
  value: number;
}

interface Props {
  width: number;
  height: number;
  marginLeft?: number;
  marginRight?: number;
  marginTop?: number;
  marginBottom?: number;
  fill?: string;
  data: DataPoint[];
  selectionBounds?: NumberDuo | undefined;
  setSelectionBounds: (newBounds: NumberDuo | undefined) => void;
}

const DraggingWindow = ({
  dragStart,
  currentX,
}: {
  dragStart: number | undefined;
  currentX: number | undefined;
}): JSX.Element | null => {
  const start = useMemo(() => Math.min(dragStart ?? 0, currentX ?? 0), [dragStart, currentX]);
  const width = useMemo(() => Math.abs((dragStart ?? 0) - (currentX ?? 0)), [dragStart, currentX]);

  if (dragStart === undefined || currentX === undefined) {
    return null;
  }

  return (
    <div
      style={{height: '100%', width, left: start}}
      className={cx(
        'bg-gray-500 absolute top-0 opacity-50 border-x-2 border-gray-900 dark:border-gray-100'
      )}
    ></div>
  );
};

const ZoomWindow = ({
  zoomWindow,
  width,
  onZoomWindowChange,
  setIsHoveringDragHandle,
}: {
  zoomWindow?: NumberDuo;
  width: number;
  onZoomWindowChange: (newWindow: NumberDuo) => void;
  setIsHoveringDragHandle: (arg: boolean) => void;
}): JSX.Element | null => {
  const windowStartHandleRef = useRef<HTMLDivElement>(null);
  const windowEndHandleRef = useRef<HTMLDivElement>(null);
  const [zoomWindowState, setZoomWindowState] = useState<NumberDuo | undefined>(zoomWindow);
  const [dragginStart, setDraggingStart] = useState(false);
  const [draggingEnd, setDraggingEnd] = useState(false);

  useEffect(() => {
    if (
      zoomWindow === undefined ||
      zoomWindowState === undefined ||
      zoomWindow[0] !== zoomWindowState[0] ||
      zoomWindow[1] !== zoomWindowState[1]
    ) {
      setZoomWindowState(zoomWindow);
    }

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [zoomWindow]);

  if (zoomWindowState === undefined) {
    return null;
  }
  const beforeStart = 0;
  const beforeWidth = zoomWindowState[0];
  const afterStart = zoomWindowState[1];
  const afterWidth = width - zoomWindowState[1];

  return (
    <div
      className="absolute w-full h-full"
      onMouseMove={e => {
        if (dragginStart) {
          const [x] = d3.pointer(e);
          if (x >= afterStart - 10) {
            return;
          }
          const newStart = Math.min(x, afterStart);
          const newEnd = Math.max(x, afterStart);
          setZoomWindowState([newStart, newEnd]);
        }
        if (draggingEnd) {
          const [x] = d3.pointer(e);
          if (x <= beforeWidth + 10) {
            return;
          }
          const newStart = Math.min(x, beforeWidth);
          const newEnd = Math.max(x, beforeWidth);
          setZoomWindowState([newStart, newEnd]);
        }
      }}
      onMouseLeave={() => {
        setDraggingStart(false);
        setDraggingEnd(false);
      }}
      onMouseUp={() => {
        if (dragginStart) {
          setDraggingStart(false);
        }
        if (draggingEnd) {
          setDraggingEnd(false);
        }
        if (zoomWindowState[0] === zoomWindow?.[0] && zoomWindowState[1] === zoomWindow?.[1]) {
          return;
        }
        onZoomWindowChange(zoomWindowState);
        setZoomWindowState(undefined);
      }}
    >
      <div
        style={{height: '100%', width: beforeWidth, left: beforeStart}}
        className={cx(
          'bg-gray-500/50 absolute top-0 border-r-2 border-gray-900 dark:border-gray-100 z-10'
        )}
      >
        <div
          className="w-3 h-4 absolute top-0 right-[-7px] rounded-b bg-gray-200  cursor-ew-resize flex justify-center"
          onMouseDown={e => {
            setDraggingStart(true);
            e.stopPropagation();
            e.preventDefault();
          }}
          ref={windowStartHandleRef}
          onMouseEnter={() => {
            setIsHoveringDragHandle(true);
          }}
          onMouseLeave={() => {
            setIsHoveringDragHandle(false);
          }}
        >
          <Icon icon="si:drag-handle-line" className="rotate-90" fontSize={16} />
        </div>
      </div>

      <div
        style={{height: '100%', width: afterWidth, left: afterStart}}
        className={cx(
          'bg-gray-500/50 absolute top-0 border-l-2 border-gray-900 dark:border-gray-100'
        )}
      >
        <div
          className="w-3 h-4 absolute top-0 rounded-b bg-gray-200 cursor-ew-resize flex justify-center left-[-7px]"
          onMouseDown={e => {
            setDraggingEnd(true);
            e.stopPropagation();
            e.preventDefault();
          }}
          ref={windowEndHandleRef}
          onMouseEnter={() => {
            setIsHoveringDragHandle(true);
          }}
          onMouseLeave={() => {
            setIsHoveringDragHandle(false);
          }}
        >
          <Icon icon="si:drag-handle-line" className="rotate-90" fontSize={16} />
        </div>
      </div>
    </div>
  );
};

export const AreaGraph = ({
  data,
  height,
  width,
  marginLeft = 0,
  marginRight = 0,
  marginBottom = 0,
  marginTop = 0,
  fill = 'gray',
  selectionBounds,
  setSelectionBounds,
}: Props): JSX.Element => {
  const [mousePosition, setMousePosition] = useState<NumberDuo | undefined>(undefined);
  const [dragStart, setDragStart] = useState<number | undefined>(undefined);
  const [isHoveringDragHandle, setIsHoveringDragHandle] = useState(false);
  const [hoverData, setHoverData] = useState<{timestamp: number; value: number} | null>(null);
  const [isMouseOverGraph, setIsMouseOverGraph] = useState(false);
  const isDragging = dragStart !== undefined;

  // Declare the x (horizontal position) scale.
  const x = d3.scaleUtc(d3.extent(data, d => d.timestamp) as NumberDuo, [
    marginLeft,
    width - marginRight,
  ]);

  // Declare the y (vertical position) scale.
  const y = d3.scaleLinear(
    [0, d3.max(data, d => d.value) as number],
    [height - marginBottom, marginTop]
  );
  const area = d3
    .area<DataPoint>()
    .curve(d3.curveMonotoneX)
    .x(d => x(d.timestamp))
    .y0(y(0))
    .y1(d => y(d.value));

  const zoomWindow: NumberDuo | undefined = useMemo(() => {
    if (selectionBounds === undefined) {
      return undefined;
    }
    return [x(selectionBounds[0]), x(selectionBounds[1])];

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectionBounds]);

  const setSelectionBoundsWithScaling = ([startPx, endPx]: NumberDuo): void => {
    setSelectionBounds([x.invert(startPx).getTime(), x.invert(endPx).getTime()]);
  };

  return (
    <div
      style={{height, width}}
      onMouseMove={e => {
        const [xPos, yPos] = d3.pointer(e);

        if (
          xPos >= marginLeft &&
          xPos <= width - marginRight &&
          yPos >= marginTop &&
          yPos <= height - marginBottom
        ) {
          setMousePosition([xPos, yPos]);

          // Find the closest data point
          if (!isHoveringDragHandle && !isDragging) {
            const xDate = x.invert(xPos);
            const bisect = d3.bisector((d: DataPoint) => d.timestamp).left;
            const index = bisect(data, xDate.getTime());
            const dataPoint = data[index];
            if (dataPoint !== undefined) {
              setHoverData(dataPoint);
            }
          }
        } else {
          setMousePosition(undefined);
          setHoverData(null);
        }
      }}
      onMouseEnter={() => {
        setIsMouseOverGraph(true);
      }}
      onMouseLeave={() => {
        setIsMouseOverGraph(false);
        setMousePosition(undefined);
        setDragStart(undefined);
        setHoverData(null);
      }}
      onMouseDown={e => {
        // only left mouse button
        if (e.button !== 0) {
          return;
        }

        // X/Y coordinate array relative to svg
        const rel = d3.pointer(e);

        const xCoordinate = rel[0];
        const xCoordinateWithoutMargin = xCoordinate - marginLeft;
        if (xCoordinateWithoutMargin >= 0) {
          setDragStart(xCoordinateWithoutMargin);
        }

        e.stopPropagation();
        e.preventDefault();
      }}
      onMouseUp={e => {
        if (dragStart === undefined) {
          return;
        }

        const rel = d3.pointer(e);
        const xCoordinate = rel[0];
        const xCoordinateWithoutMargin = xCoordinate - marginLeft;
        if (xCoordinateWithoutMargin >= 0 && dragStart !== xCoordinateWithoutMargin) {
          const start = Math.min(dragStart, xCoordinateWithoutMargin);
          const end = Math.max(dragStart, xCoordinateWithoutMargin);
          setSelectionBoundsWithScaling([start, end]);
        }
        setDragStart(undefined);
      }}
      className="relative"
    >
      {/* onHover guide, only visible when hovering and not dragging and not having an active zoom window */}
      <div
        style={{height, width: 2, left: mousePosition?.[0] ?? -1}}
        className={cx('bg-gray-700/75 dark:bg-gray-200/75 absolute top-0', {
          hidden: mousePosition === undefined || isDragging || isHoveringDragHandle,
        })}
      ></div>

      {/* drag guide, only visible when dragging */}
      <DraggingWindow dragStart={dragStart} currentX={mousePosition?.[0]} />

      {/* zoom window */}
      <ZoomWindow
        zoomWindow={zoomWindow}
        width={width}
        onZoomWindowChange={setSelectionBoundsWithScaling}
        setIsHoveringDragHandle={setIsHoveringDragHandle}
      />

      {/* Inactive indicator */}
      <div
        className={cx('absolute top-0 left-0 w-full h-full bg-gray-900/50 dark:bg-gray-200/50', {
          hidden: isDragging || selectionBounds !== undefined,
        })}
      ></div>

      {/* Update Tooltip conditional render */}
      {mousePosition !== undefined &&
        hoverData !== null &&
        !isDragging &&
        !isHoveringDragHandle &&
        isMouseOverGraph && (
          <Tooltip
            x={mousePosition[0]}
            y={mousePosition[1]}
            timestamp={new Date(hoverData.timestamp)}
            value={hoverData.value}
            containerWidth={width}
          />
        )}

      <svg style={{width: '100%', height: '100%'}}>
        <path fill={fill} d={area(data) as string} className="opacity-80" />
      </svg>
    </div>
  );
};
