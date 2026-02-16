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

import {useCallback, useEffect, useRef, useState} from 'react';

import {flushSync} from 'react-dom';

const MIN_ZOOM = 1.0;
const MAX_ZOOM = 20.0;
const BUTTON_ZOOM_STEP = 1.5;
// Sensitivity for trackpad/wheel zoom - smaller = smoother
const WHEEL_ZOOM_SENSITIVITY = 0.01;

interface UseZoomResult {
  zoomLevel: number;
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
}

const clampZoom = (zoom: number): number => {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom));
};

export const useZoom = (containerRef: React.RefObject<HTMLDivElement | null>): UseZoomResult => {
  const [zoomLevel, setZoomLevel] = useState(MIN_ZOOM);
  const zoomLevelRef = useRef(MIN_ZOOM);

  // Adjust scrollLeft so the content under focalX stays fixed after zoom change.
  const adjustScroll = useCallback(
    (oldZoom: number, newZoom: number, focalX: number) => {
      const container = containerRef.current;
      if (container === null) return;

      const contentX = container.scrollLeft + focalX;
      const ratio = contentX / oldZoom;
      container.scrollLeft = ratio * newZoom - focalX;
    },
    [containerRef]
  );

  // Apply a new zoom level around a focal point
  const applyZoom = useCallback(
    (newZoom: number, focalX: number) => {
      const oldZoom = zoomLevelRef.current;
      if (newZoom === oldZoom) return;
      zoomLevelRef.current = newZoom;

      // flushSync ensures the DOM updates with the new content width before adjustScroll reads it
      flushSync(() => setZoomLevel(newZoom));
      adjustScroll(oldZoom, newZoom, focalX);
    },
    [adjustScroll]
  );

  const zoomIn = useCallback(() => {
    const newZoom = clampZoom(zoomLevelRef.current * BUTTON_ZOOM_STEP);
    const container = containerRef.current;
    applyZoom(newZoom, container !== null ? container.clientWidth / 2 : 0);
  }, [containerRef, applyZoom]);

  const zoomOut = useCallback(() => {
    const newZoom = clampZoom(zoomLevelRef.current / BUTTON_ZOOM_STEP);
    const container = containerRef.current;
    applyZoom(newZoom, container !== null ? container.clientWidth / 2 : 0);
  }, [containerRef, applyZoom]);

  const resetZoom = useCallback(() => {
    zoomLevelRef.current = MIN_ZOOM;
    setZoomLevel(MIN_ZOOM);
    const container = containerRef.current;
    if (container !== null) {
      container.scrollLeft = 0;
    }
  }, [containerRef]);

  useEffect(() => {
    const container = containerRef.current;
    if (container === null) return;

    const handleWheel = (e: WheelEvent): void => {
      if (!e.ctrlKey && !e.metaKey) return;
      e.preventDefault();

      let delta = e.deltaY;
      if (e.deltaMode === 1) {
        delta *= 20;
      }

      // Limiting the max zoom step per event to 15%, so to fix the huge jumps in Linux OS.
      const MAX_FACTOR = 0.15;
      const rawFactor = -delta * WHEEL_ZOOM_SENSITIVITY;
      const zoomFactor = 1 + Math.max(-MAX_FACTOR, Math.min(MAX_FACTOR, rawFactor));

      const newZoom = clampZoom(zoomLevelRef.current * zoomFactor);
      applyZoom(newZoom, e.clientX - container.getBoundingClientRect().left);
    };

    container.addEventListener('wheel', handleWheel, {passive: false});
    return () => {
      container.removeEventListener('wheel', handleWheel);
    };
  }, [containerRef, applyZoom]);

  return {zoomLevel, zoomIn, zoomOut, resetZoom};
};
