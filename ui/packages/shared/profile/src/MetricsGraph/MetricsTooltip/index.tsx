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

import {useEffect, useState} from 'react';

import type {VirtualElement} from '@popperjs/core';
import {usePopper} from 'react-popper';

import {Label} from '@parca/client';
import {TextWithTooltip} from '@parca/components';
import {formatDate, valueFormatter} from '@parca/utilities';

import {HighlightedSeries} from '../';
import {timeFormat} from '../../';

interface Props {
  x: number;
  y: number;
  highlighted: HighlightedSeries;
  onLabelClick: (labelName: string, labelValue: string) => void;
  contextElement: Element | null;
  sampleUnit: string;
  delta: boolean;
}

const virtualElement: VirtualElement = {
  getBoundingClientRect: () => {
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    return {
      width: 0,
      height: 0,
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
    } as DOMRect;
  },
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

const MetricsTooltip = ({
  x,
  y,
  highlighted,
  onLabelClick,
  contextElement,
  sampleUnit,
  delta,
}: Props): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(virtualElement, popperElement, {
    placement: 'auto-start',
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
  });

  const update = popperProps.update;

  useEffect(() => {
    if (contextElement != null) {
      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(contextElement, x, y);
      void update?.();
    }
  }, [x, y, contextElement, update]);

  const nameLabel: Label | undefined = highlighted?.labels.find(e => e.name === '__name__');
  const highlightedNameLabel: Label = nameLabel !== undefined ? nameLabel : {name: '', value: ''};

  return (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper} className="z-10">
      <div className="flex max-w-md">
        <div className="m-auto">
          <div
            className="rounded-lg border-gray-300 bg-gray-50 p-3 opacity-90 shadow-lg dark:border-gray-500 dark:bg-gray-900"
            style={{borderWidth: 1}}
          >
            <div className="flex flex-row">
              <div className="ml-2 mr-6">
                <span className="font-semibold">{highlightedNameLabel.value}</span>
                <span className="my-2 block text-gray-700 dark:text-gray-300">
                  <table className="table-auto">
                    <tbody>
                      <tr>
                        <td className="w-1/4">Value</td>
                        <td className="w-3/4">
                          {valueFormatter(highlighted.valuePerSecond, sampleUnit, 5)}
                        </td>
                      </tr>
                      {delta && (
                        <tr>
                          <td className="w-1/4">Total</td>
                          <td className="w-3/4">
                            {valueFormatter(highlighted.value, sampleUnit, 2)}
                          </td>
                        </tr>
                      )}
                      {highlighted.duration > 0 && (
                        <tr>
                          <td className="w-1/4">Duration</td>
                          <td className="w-3/4">
                            {valueFormatter(highlighted.duration, 'nanoseconds', 2)}
                          </td>
                        </tr>
                      )}
                      <tr>
                        <td className="w-1/4">At</td>
                        <td className="w-3/4">{formatDate(highlighted.timestamp, timeFormat)}</td>
                      </tr>
                    </tbody>
                  </table>
                </span>
                <span className="my-2 block text-gray-500">
                  {highlighted.labels
                    .filter((label: Label) => label.name !== '__name__')
                    .map(function (label: Label) {
                      return (
                        <button
                          key={label.name}
                          type="button"
                          className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
                          onClick={() => onLabelClick(label.name, label.value)}
                        >
                          <TextWithTooltip
                            text={`${label.name}="${label.value}"`}
                            maxTextLength={37}
                            id={`tooltip-${label.name}-${label.value}`}
                          />
                        </button>
                      );
                    })}
                </span>
                <span className="block text-xs text-gray-500">
                  Hold shift and click label to add to query.
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default MetricsTooltip;
