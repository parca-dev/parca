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

export interface ViewportState {
  scrollTop: number;
  scrollLeft: number;
  containerHeight: number;
  containerWidth: number;
}

// Find the scrollable ancestor (the element with overflow: auto/scroll)
const findScrollableParent = (element: HTMLElement | null): HTMLElement | undefined => {
  if (element === null) return undefined;
  let current: HTMLElement | null = element.parentElement;
  while (current !== null) {
    const style = window.getComputedStyle(current);
    const overflowY = style.overflowY;
    if (overflowY === 'auto' || overflowY === 'scroll') {
      return current;
    }
    current = current.parentElement;
  }
  return undefined;
};

export const useScrollViewport = (containerRef: React.RefObject<HTMLDivElement>): ViewportState => {
  const [viewport, setViewport] = useState<ViewportState>({
    scrollTop: 0,
    scrollLeft: 0,
    containerHeight: 0,
    containerWidth: 0,
  });

  const throttleRef = useRef<number | null>(null);

  const updateViewport = useCallback(() => {
    if (containerRef.current !== null) {
      const container = containerRef.current;
      const rect = container.getBoundingClientRect();

      // Restrict container height to the visible portion on screen
      // This handles cases where the container is partially off-screen
      // We only want to consider the visible part for culling calculations

      const containerTop = rect.top;
      const containerBottom = rect.bottom;
      const viewportTop = 0;
      const viewportBottom = window.innerHeight;
      const visibleTop = Math.max(containerTop, viewportTop);
      const visibleBottom = Math.min(containerBottom, viewportBottom);
      const visibleHeight = Math.max(0, visibleBottom - visibleTop);
      const scrollOffset = Math.max(0, viewportTop - containerTop);

      const newViewport = {
        scrollTop: scrollOffset,
        scrollLeft: container.scrollLeft,
        containerHeight: visibleHeight, // Only the visible portion
        containerWidth: container.clientWidth,
      };

      setViewport(newViewport);
    }
  }, [containerRef]);

  // Throttling Strategy:
  // Use requestAnimationFrame to throttle scroll events to 60fps max
  // This ensures smooth performance while preventing excessive re-renders
  const throttledUpdateViewport = useCallback(() => {
    if (throttleRef.current !== null) {
      cancelAnimationFrame(throttleRef.current);
    }
    throttleRef.current = requestAnimationFrame(updateViewport);
  }, [updateViewport]);

  useEffect(() => {
    const container = containerRef.current;
    if (container === null) return;

    const scrollableParent = findScrollableParent(container);

    // ResizeObserver Strategy:
    // Monitor container size changes (window resize, layout shifts)
    // to update viewport dimensions for accurate culling calculations
    const resizeObserver = new ResizeObserver(() => {
      throttledUpdateViewport();
    });

    // Listen to scroll on the actual scrollable parent

    scrollableParent?.addEventListener('scroll', throttledUpdateViewport, {passive: true});
    container.addEventListener('scroll', throttledUpdateViewport, {passive: true});
    window.addEventListener('scroll', throttledUpdateViewport, {passive: true});

    resizeObserver.observe(container);

    // Initialize viewport state on mount
    updateViewport();

    return () => {
      // Cleanup: Remove event listeners and cancel pending animations
      scrollableParent?.removeEventListener('scroll', throttledUpdateViewport);
      container.removeEventListener('scroll', throttledUpdateViewport);
      window.removeEventListener('scroll', throttledUpdateViewport);
      resizeObserver.disconnect();
      if (throttleRef.current !== null) {
        cancelAnimationFrame(throttleRef.current);
      }
    };
  }, [containerRef, throttledUpdateViewport, updateViewport]);

  return viewport;
};
