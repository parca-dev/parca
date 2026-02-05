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

import {Fragment, useState} from 'react';

import {Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import {usePopper} from 'react-popper';

import {useParcaContext} from '@parca/components';

import {OPTIMAL_LABELS} from '../hooks/useAutoConfigureFlamechart';

interface FlameChartAutoConfigPopoverProps {
  isOpen: boolean;
  onDismiss: () => void;
  anchorRef: React.RefObject<HTMLElement>;
  currentSumByLabels: string[];
}

export const FlameChartAutoConfigPopover = ({
  isOpen,
  onDismiss,
  anchorRef,
  currentSumByLabels,
}: FlameChartAutoConfigPopoverProps): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);
  const {flamechartHelpText} = useParcaContext();

  const autoConfiguredLabels = OPTIMAL_LABELS.filter(label => !currentSumByLabels.includes(label));

  const {styles, attributes} = usePopper(anchorRef.current, popperElement, {
    placement: 'bottom-start',
    strategy: 'absolute',
    modifiers: [
      {
        name: 'offset',
        options: {
          offset: [0, 8],
        },
      },
      {
        name: 'flip',
        options: {
          fallbackPlacements: ['top-start', 'bottom-end', 'top-end'],
        },
      },
    ],
  });

  if (!isOpen) return <></>;

  return (
    <Transition
      show={isOpen}
      as={Fragment}
      enter="transition ease-out duration-200"
      enterFrom="opacity-0 translate-y-1"
      enterTo="opacity-100 translate-y-0"
      leave="transition ease-in duration-150"
      leaveFrom="opacity-100 translate-y-0"
      leaveTo="opacity-0 translate-y-1"
    >
      <div
        ref={setPopperElement}
        style={styles.popper}
        {...attributes.popper}
        className="z-50 w-96 rounded-lg bg-blue-50 dark:bg-gray-900 border border-blue-200 dark:border-gray-700 shadow-lg p-4"
        role="alert"
        aria-live="polite"
        aria-atomic="true"
      >
        {/* Close Button */}
        <button
          onClick={onDismiss}
          className="absolute top-4 right-2 text-gray-600 dark:text-gray-300 hover:text-gray-800 dark:hover:text-gray-200"
          aria-label="Dismiss"
        >
          <Icon icon="mdi:close" width={20} height={20} />
        </button>

        {/* Icon */}
        <div className="flex items-start gap-3">
          {/* Content */}
          <div className="flex-1 pr-6">
            <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100 mb-2">
              Flamechart Settings Auto-Configured
            </h3>
            <p className="text-sm text-gray-800 dark:text-gray-200 mb-3">
              We&apos;ve automatically adjusted your settings for optimal flamechart viewing:
            </p>
            <ul className="text-sm text-gray-800 dark:text-gray-200 mb-3 list-disc list-inside space-y-1">
              <li>Time range reduced to 1 minute</li>
              {autoConfiguredLabels.length > 0 && (
                <li>
                  Added sum-by labels:{' '}
                  {autoConfiguredLabels.map((label, index) => (
                    <span key={label}>
                      <code className="text-xs text-gray-200 dark:text-gray-800 bg-indigo-600 dark:bg-indigo-500 p-1 rounded">
                        {label}
                      </code>
                      {index < autoConfiguredLabels.length - 1 ? ', ' : ''}
                    </span>
                  ))}
                </li>
              )}
            </ul>
            {flamechartHelpText != null && (
              <p className="text-sm text-gray-800 dark:text-gray-200 [&_a]:underline">
                {flamechartHelpText}
              </p>
            )}
          </div>
        </div>
      </div>
    </Transition>
  );
};
