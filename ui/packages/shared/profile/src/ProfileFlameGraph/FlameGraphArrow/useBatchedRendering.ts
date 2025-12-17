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

interface UseBatchedRenderingOptions {
  batchSize?: number;
  // Delay between batches in ms (0 = next animation frame)
  batchDelay?: number;
}

interface UseBatchedRenderingResult<T> {
  items: T[];
  isComplete: boolean;
}

//useBatchedRendering - Helps in incrementally rendering items in batches to avoid UI blocking.
export const useBatchedRendering = <T>(
  items: T[],
  options: UseBatchedRenderingOptions = {}
): UseBatchedRenderingResult<T> => {
  const {batchSize = 500, batchDelay = 0} = options;

  const [renderedCount, setRenderedCount] = useState(0);
  const itemsRef = useRef(items);
  const rafRef = useRef<number | null>(null);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    if (itemsRef.current !== items) {
      itemsRef.current = items;
      setRenderedCount(prev => {
        if (items.length === 0) return 0;
        // If new items were added (scrolling down), keep current progress
        if (items.length > prev) return prev;
        // If items reduced, cap to new length
        return Math.min(prev, items.length);
      });
    }
  }, [items]);

  // Progressively render more items
  useEffect(() => {
    if (renderedCount === items.length) {
      return;
    }

    const scheduleNextBatch = (): void => {
      const incrementState = () => {
        setRenderedCount(prev => Math.min(prev + batchSize, items.length));
      };
      if (batchDelay > 0) {
        timeoutRef.current = setTimeout(incrementState, batchDelay);
      } else {
        rafRef.current = requestAnimationFrame(incrementState);
      }
    };
    scheduleNextBatch();

    return () => {
      if (rafRef.current !== null) {
        cancelAnimationFrame(rafRef.current);
      }
      if (timeoutRef.current !== null) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [renderedCount, items.length, batchSize, batchDelay]);

  return {
    items: items.slice(0, renderedCount),
    isComplete: renderedCount === items.length,
  };
};
