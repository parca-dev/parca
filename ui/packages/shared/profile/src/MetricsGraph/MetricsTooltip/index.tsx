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

/* eslint-disable react-hooks/refs */

import {useLayoutEffect, useRef, useState} from 'react';

import {usePopper} from 'react-popper';

import {TEST_IDS, testId} from '@parca/test-utils';

interface VirtualElement {
  getBoundingClientRect: () => DOMRect;
}

interface Props {
  x: number;
  y: number;
  contextElement: Element | null;
  content: React.ReactNode;
}

const emptyRect: DOMRect = {
  width: 0,
  height: 0,
  top: 0,
  right: 0,
  bottom: 0,
  left: 0,
  x: 0,
  y: 0,
  toJSON: () => ({}),
};

const createDomRect = (x: number, y: number): DOMRect => {
  const domRect: DOMRect = {
    width: 0,
    height: 0,
    top: y,
    right: x,
    bottom: y,
    left: x,
    x,
    y,
    toJSON: () => ({}),
  };
  return domRect;
};

const MetricsTooltip = ({x, y, contextElement, content}: Props): JSX.Element => {
  'use no memo';
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const virtualElementRef = useRef<VirtualElement>({
    getBoundingClientRect: () => emptyRect,
  });

  const {styles, attributes, update} = usePopper(virtualElementRef.current, popperElement, {
    placement: 'auto-start',
    strategy: 'absolute',
    modifiers: [
      {
        name: 'preventOverflow',
        options: {
          boundary: contextElement ?? undefined,
        },
      },
      {
        name: 'offset',
        options: {
          offset: [15, 15],
        },
      },
    ],
  });

  useLayoutEffect(() => {
    virtualElementRef.current.getBoundingClientRect = (): DOMRect => {
      const domRect: DOMRect = (contextElement as Element)?.getBoundingClientRect() ?? emptyRect;
      return createDomRect(domRect.x + x, domRect.y + y);
    };
    void update?.();
  }, [x, y, contextElement, update]);

  // Don't render anything if content is null or undefined
  if (content == null) {
    return <></>;
  }

  return (
    <div
      ref={setPopperElement}
      style={styles.popper}
      {...attributes.popper}
      {...testId(TEST_IDS.METRICS_GRAPH_TOOLTIP)}
      className="z-50"
    >
      <div className="flex max-w-lg">
        <div className="m-auto">
          <div className="border border-gray-300 bg-gray-50 dark:border-gray-500 dark:bg-gray-900 rounded-lg shadow-lg px-3 py-2">
            {content}
          </div>
        </div>
      </div>
    </div>
  );
};

export default MetricsTooltip;
