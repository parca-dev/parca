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

      const newViewport = {
        scrollTop: container.scrollTop,
        scrollLeft: container.scrollLeft,
        containerHeight: container.clientHeight,
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

    // ResizeObserver Strategy:
    // Monitor container size changes (window resize, layout shifts)
    // to update viewport dimensions for accurate culling calculations
    const resizeObserver = new ResizeObserver(() => {
      throttledUpdateViewport();
    });

    // Container Scroll Event Strategy:
    // Use passive event listeners for better scroll performance
    // Throttle with requestAnimationFrame to maintain 60fps target
    container.addEventListener('scroll', throttledUpdateViewport, {passive: true});
    resizeObserver.observe(container);

    // Initialize viewport state on mount
    updateViewport();

    return () => {
      // Cleanup: Remove event listeners and cancel pending animations
      container.removeEventListener('scroll', throttledUpdateViewport);
      resizeObserver.disconnect();
      if (throttleRef.current !== null) {
        cancelAnimationFrame(throttleRef.current);
      }
    };
  }, [containerRef, throttledUpdateViewport, updateViewport]);

  return viewport;
};
