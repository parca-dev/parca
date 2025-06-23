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

import React, {useEffect, useState} from 'react';

import {pointer} from 'd3-selection';
import { useFloating, offset, shift, flip, autoUpdate, VirtualElement } from '@floating-ui/react';

interface GraphTooltipProps {
  children: React.ReactNode;
  contextElement: Element | null;
}


function createPositionedVirtualElement(contextElement: Element, x = 0, y = 0): VirtualElement {
  const domRect = contextElement.getBoundingClientRect();
  return {
    getBoundingClientRect: () => ({
      width: 0,
      height: 0,
      top: domRect.y + y,
      left: domRect.x + x,
      right: domRect.x + x,
      bottom: domRect.y + y,
      x: domRect.x + x,
      y: domRect.y + y,
      toJSON: () => ({}),
    }),
  };
}

const GraphTooltip = ({
  children,
  contextElement,
}: GraphTooltipProps): React.JSX.Element => {
  const [isPositioned, setIsPositioned] = useState(false);

  const { refs, floatingStyles, update } = useFloating({
    placement: 'bottom-start',
    strategy: 'absolute',
    middleware: [
      offset({
        mainAxis: 30,
        crossAxis: 30,
      }),
      flip(),
      shift({
        padding: 5,
      }),
    ],
    whileElementsMounted: undefined,
  });


  useEffect(() => {
    if (contextElement === null) return;

    const onMouseMove: EventListenerOrEventListenerObject = (e: Event) => {
      const rel = pointer(e);
      const tooltipX = rel[0];
      const tooltipY = rel[1];
      const virtualElement = createPositionedVirtualElement(
        contextElement,
        tooltipX,
        tooltipY
      );
      refs.setReference(virtualElement);
      setIsPositioned(true);
      update();
    };

    contextElement.addEventListener('mousemove', onMouseMove);
    return () => {
      contextElement.removeEventListener('mousemove', onMouseMove);
    };
  }, [contextElement, update, refs]);

  return (
    <div
      ref={refs.setFloating}
      style={{
        ...floatingStyles,
        visibility: !isPositioned ? 'hidden' : 'visible'
      }}
      className="z-50 w-max"
    >
      {children}
    </div>
  );
};

export default GraphTooltip;
