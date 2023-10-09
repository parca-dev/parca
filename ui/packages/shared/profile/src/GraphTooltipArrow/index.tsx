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
import {usePopper} from 'react-popper';

interface GraphTooltipProps {
  children: React.ReactNode;
  x?: number;
  y?: number;
  contextElement: Element | null;
  isFixed?: boolean;
  virtualContextElement?: boolean;
  isContextMenuOpen?: boolean;
}

const virtualElement = {
  getBoundingClientRect: () =>
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    ({
      width: 0,
      height: 0,
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
    } as DOMRect),
};

function generateGetBoundingClientRect(contextElement: Element, x = 0, y = 0): () => DOMRect {
  const domRect = contextElement.getBoundingClientRect();
  return () =>
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    ({
      width: 0,
      height: 0,
      top: domRect.y + y,
      left: domRect.x + x,
      right: domRect.x + x,
      bottom: domRect.y + y,
    } as DOMRect);
}

const GraphTooltip = ({
  children,
  x,
  y,
  contextElement,
  isFixed = false,
  virtualContextElement = true,
  isContextMenuOpen = false,
}: GraphTooltipProps): React.JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(
    virtualContextElement ? virtualElement : contextElement,
    popperElement,
    {
      placement: 'bottom-start',
      strategy: 'absolute',
      modifiers: [
        {
          name: 'preventOverflow',
          options: {
            tether: false,
            altAxis: true,
          },
        },
        {
          name: 'offset',
          options: {
            offset: [30, 30],
          },
        },
      ],
    }
  );

  useEffect(() => {
    if (contextElement === null) return;
    const onMouseMove: EventListenerOrEventListenerObject = (e: Event) => {
      if (isContextMenuOpen) {
        return;
      }

      let tooltipX = x;
      let tooltipY = y;
      if (tooltipX == null || tooltipY == null) {
        const rel = pointer(e);
        tooltipX = rel[0];
        tooltipY = rel[1];
      }
      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(
        contextElement,
        tooltipX,
        tooltipY
      );

      void popperProps.update?.();
    };

    contextElement.addEventListener('mousemove', onMouseMove);
    return () => {
      contextElement.removeEventListener('mousemove', onMouseMove);
    };
  }, [contextElement, popperProps, x, y, isContextMenuOpen]);

  return isFixed ? (
    <>{children}</>
  ) : (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper} className="z-10">
      {children}
    </div>
  );
};

export default GraphTooltip;
