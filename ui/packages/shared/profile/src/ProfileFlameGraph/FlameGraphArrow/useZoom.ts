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
export const MAX_ZOOM = 100.0;
const BUTTON_ZOOM_STEP = 1.5;
// Sensitivity for trackpad/wheel zoom - smaller = smoother
const WHEEL_ZOOM_SENSITIVITY = 0.01;

interface UseZoomResult {
  zoomLevel: number;
  zoomIn: () => void;
  zoomOut: () => void;
  resetZoom: () => void;
  zoomToPosition: (normalizedX: number, targetZoom: number) => void;
  setZoomWithScroll: (zoom: number, scrollLeft: number) => void;
  scrollLeftRef: React.RefObject<number>;
}

const clampZoom = (zoom: number): number => {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom));
};

export const useZoom = (containerRef: React.RefObject<HTMLDivElement | null>): UseZoomResult => {
  const [zoomLevel, setZoomLevel] = useState(MIN_ZOOM);
  const zoomLevelRef = useRef(MIN_ZOOM);
  const scrollLeftRef = useRef(0);

  // Keep scrollLeftRef in sync with actual scroll position during regular scrolling
  useEffect(() => {
    const container = containerRef.current;
    if (container === null) return;

    const onScroll = (): void => {
      scrollLeftRef.current = container.scrollLeft;
    };
    container.addEventListener('scroll', onScroll, {passive: true});
    return () => container.removeEventListener('scroll', onScroll);
  }, [containerRef]);

  // Apply a new zoom level around a focal point
  const applyZoom = useCallback(
    (newZoom: number, focalX: number) => {
      const container = containerRef.current;
      if (container === null) return;

      const oldZoom = zoomLevelRef.current;
      if (newZoom === oldZoom) return;

      // Pre-compute intended scrollLeft BEFORE flushSync so MiniMap reads correct value during render
      const contentX = container.scrollLeft + focalX;
      const ratio = contentX / oldZoom;
      const newScrollLeft = ratio * newZoom - focalX;
      scrollLeftRef.current = newScrollLeft;

      zoomLevelRef.current = newZoom;
      flushSync(() => setZoomLevel(newZoom));

      // Apply scroll to DOM after render (content is now wide enough)
      container.scrollLeft = newScrollLeft;
    },
    [containerRef]
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
    scrollLeftRef.current = 0;
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

  const zoomToPosition = useCallback(
    (normalizedX: number, targetZoom: number) => {
      const container = containerRef.current;
      if (container === null) return;

      const newZoom = clampZoom(targetZoom);
      if (newZoom === zoomLevelRef.current) return;

      const containerWidth = container.clientWidth;
      const contentWidth = containerWidth * newZoom;
      const targetScrollLeft = Math.max(
        0,
        Math.min(normalizedX * contentWidth - containerWidth / 2, contentWidth - containerWidth)
      );

      // Pre-set scrollLeftRef before flushSync so MiniMap reads correct value during render
      scrollLeftRef.current = targetScrollLeft;
      zoomLevelRef.current = newZoom;
      flushSync(() => setZoomLevel(newZoom));

      container.scrollLeft = targetScrollLeft;
    },
    [containerRef]
  );

  const setZoomWithScroll = useCallback(
    (zoom: number, newScrollLeft: number) => {
      const container = containerRef.current;
      if (container === null) return;

      const clamped = clampZoom(zoom);
      const contentWidth = container.clientWidth * clamped;
      const clampedScroll = Math.max(
        0,
        Math.min(newScrollLeft, contentWidth - container.clientWidth)
      );

      // Pre-set scrollLeftRef before flushSync so MiniMap reads correct value during render
      scrollLeftRef.current = clampedScroll;
      zoomLevelRef.current = clamped;
      flushSync(() => setZoomLevel(clamped));

      container.scrollLeft = clampedScroll;
    },
    [containerRef]
  );

  return {zoomLevel, zoomIn, zoomOut, resetZoom, zoomToPosition, setZoomWithScroll, scrollLeftRef};
};
