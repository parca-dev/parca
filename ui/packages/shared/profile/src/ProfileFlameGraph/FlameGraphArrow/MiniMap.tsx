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

import React, {useCallback, useEffect, useRef} from 'react';

import {Table} from '@uwdata/flechette';

import {EVERYTHING_ELSE} from '@parca/store';
import {getLastItem} from '@parca/utilities';

import {ProfileSource} from '../../ProfileSource';
import {RowHeight, type colorByColors} from './FlameGraphNodes';
import {
  FIELD_CUMULATIVE,
  FIELD_DEPTH,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_MAPPING_FILE,
  FIELD_TIMESTAMP,
} from './index';
import {arrowToString, boundsFromProfileSource} from './utils';

const MINIMAP_HEIGHT = 20;

interface MiniMapProps {
  containerRef: React.RefObject<HTMLDivElement | null>;
  table: Table;
  width: number;
  zoomedWidth: number;
  totalHeight: number;
  maxDepth: number;
  colorByColors: colorByColors;
  colorBy: string;
  profileSource: ProfileSource;
  isDarkMode: boolean;
  scrollLeft: number;
}

export const MiniMap = React.memo(function MiniMap({
  containerRef,
  table,
  width,
  zoomedWidth,
  totalHeight,
  maxDepth,
  colorByColors: colors,
  colorBy,
  profileSource,
  isDarkMode,
  scrollLeft,
}: MiniMapProps): React.JSX.Element | null {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerElRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);
  const dragStartX = useRef(0);
  const dragStartScrollLeft = useRef(0);

  // Render minimap canvas
  useEffect(() => {
    const canvas = canvasRef.current;
    if (canvas == null || width <= 0 || zoomedWidth <= 0) return;

    const dpr = window.devicePixelRatio !== 0 ? window.devicePixelRatio : 1;
    canvas.width = width * dpr;
    canvas.height = MINIMAP_HEIGHT * dpr;

    const ctx = canvas.getContext('2d');
    if (ctx == null) return;

    ctx.scale(dpr, dpr);
    ctx.clearRect(0, 0, width, MINIMAP_HEIGHT);

    // Background
    ctx.fillStyle = isDarkMode ? '#374151' : '#f3f4f6';
    ctx.fillRect(0, 0, width, MINIMAP_HEIGHT);

    const xScale = width / zoomedWidth;
    const yScale = MINIMAP_HEIGHT / totalHeight;

    const tsBounds = boundsFromProfileSource(profileSource);
    const tsRange = Number(tsBounds[1]) - Number(tsBounds[0]);
    if (tsRange <= 0) return;

    const depthCol = table.getChild(FIELD_DEPTH);
    const cumulativeCol = table.getChild(FIELD_CUMULATIVE);
    const tsCol = table.getChild(FIELD_TIMESTAMP);
    const mappingCol = table.getChild(FIELD_MAPPING_FILE);
    const filenameCol = table.getChild(FIELD_FUNCTION_FILE_NAME);

    if (depthCol == null || cumulativeCol == null) return;

    const numRows = table.numRows;

    for (let row = 0; row < numRows; row++) {
      const depth = depthCol.get(row) ?? 0;
      if (depth === 0) continue; // skip root

      if (depth > maxDepth) continue;

      const cumulative = Number(cumulativeCol.get(row) ?? 0n);
      if (cumulative <= 0) continue;

      const nodeWidth = (cumulative / tsRange) * zoomedWidth * xScale;
      if (nodeWidth < 0.5) continue;

      const ts = tsCol != null ? Number(tsCol.get(row)) : 0;
      const x = ((ts - Number(tsBounds[0])) / tsRange) * zoomedWidth * xScale;
      const y = (depth - 1) * RowHeight * yScale;
      const h = Math.max(1, RowHeight * yScale);

      // Get color using same logic as useNodeColor
      const colorAttribute =
        colorBy === 'filename'
          ? arrowToString(filenameCol?.get(row))
          : colorBy === 'binary'
          ? arrowToString(mappingCol?.get(row))
          : null;

      const color = colors[getLastItem(colorAttribute ?? '') ?? EVERYTHING_ELSE];
      ctx.fillStyle = color ?? (isDarkMode ? '#6b7280' : '#9ca3af');
      ctx.fillRect(x, y, Math.max(0.5, nodeWidth), h);
    }
  }, [
    table,
    width,
    zoomedWidth,
    totalHeight,
    maxDepth,
    colorBy,
    colors,
    isDarkMode,
    profileSource,
  ]);

  const isZoomed = zoomedWidth > width;
  const sliderWidth = Math.max(20, (width / zoomedWidth) * width);
  const sliderLeft = Math.min((scrollLeft / zoomedWidth) * width, width - sliderWidth);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const rect = containerElRef.current?.getBoundingClientRect();
      if (rect == null) return;

      const clickX = e.clientX - rect.left;

      // Check if clicking inside the slider
      if (clickX >= sliderLeft && clickX <= sliderLeft + sliderWidth) {
        // Start dragging
        isDragging.current = true;
        dragStartX.current = e.clientX;
        dragStartScrollLeft.current = scrollLeft;
      } else {
        // Click-to-jump: center viewport at click position
        const targetCenter = (clickX / width) * zoomedWidth;
        const containerWidth = containerRef.current?.clientWidth ?? width;
        const newScrollLeft = targetCenter - containerWidth / 2;
        if (containerRef.current != null) {
          containerRef.current.scrollLeft = Math.max(
            0,
            Math.min(newScrollLeft, zoomedWidth - containerWidth)
          );
        }
        // Also start dragging from new position
        isDragging.current = true;
        dragStartX.current = e.clientX;
        dragStartScrollLeft.current = containerRef.current?.scrollLeft ?? 0;
      }

      const handleMouseMove = (moveEvent: MouseEvent): void => {
        if (!isDragging.current) return;
        const delta = moveEvent.clientX - dragStartX.current;
        const scrollDelta = delta * (zoomedWidth / width);
        const containerWidth = containerRef.current?.clientWidth ?? width;
        if (containerRef.current != null) {
          containerRef.current.scrollLeft = Math.max(
            0,
            Math.min(dragStartScrollLeft.current + scrollDelta, zoomedWidth - containerWidth)
          );
        }
      };

      const handleMouseUp = (): void => {
        isDragging.current = false;
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      };

      document.addEventListener('mousemove', handleMouseMove);
      document.addEventListener('mouseup', handleMouseUp);
    },
    [sliderLeft, sliderWidth, scrollLeft, width, zoomedWidth, containerRef]
  );

  // Forward wheel events to the container so zoom (Ctrl+scroll) works on the minimap
  useEffect(() => {
    const el = containerElRef.current;
    if (el == null) return;

    const handleWheel = (e: WheelEvent): void => {
      if (!e.ctrlKey && !e.metaKey) return;
      e.preventDefault();
      containerRef.current?.dispatchEvent(
        new WheelEvent('wheel', {
          deltaY: e.deltaY,
          deltaX: e.deltaX,
          ctrlKey: e.ctrlKey,
          metaKey: e.metaKey,
          clientX: e.clientX,
          clientY: e.clientY,
          bubbles: true,
        })
      );
    };

    el.addEventListener('wheel', handleWheel, {passive: false});
    return () => {
      el.removeEventListener('wheel', handleWheel);
    };
  }, [containerRef]);

  if (width <= 0) return null;

  return (
    <div
      ref={containerElRef}
      className="relative select-none"
      style={{width, height: MINIMAP_HEIGHT, cursor: isZoomed ? 'pointer' : 'default'}}
      onMouseDown={isZoomed ? handleMouseDown : undefined}
    >
      <canvas
        ref={canvasRef}
        style={{
          width,
          height: MINIMAP_HEIGHT,
          display: 'block',
          visibility: isZoomed ? 'visible' : 'hidden',
        }}
      />
      {isZoomed && (
        <>
          {/* Left overlay */}
          <div
            className="absolute top-0 bottom-0 bg-black/30 dark:bg-black/50"
            style={{left: 0, width: Math.max(0, sliderLeft)}}
          />
          {/* Viewport slider */}
          <div
            className="absolute top-0 bottom-0 border-x-2 border-gray-500"
            style={{left: sliderLeft, width: sliderWidth}}
          />
          {/* Right overlay */}
          <div
            className="absolute top-0 bottom-0 bg-black/30 dark:bg-black/50"
            style={{left: sliderLeft + sliderWidth, right: 0}}
          />
        </>
      )}
    </div>
  );
});
