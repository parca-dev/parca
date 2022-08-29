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

import React, {useState} from 'react';
import {ChevronDoubleDownIcon, ChevronDoubleUpIcon} from '@heroicons/react/solid';
import {Pill, PillVariant} from '@parca/components';

const LabelsCell = ({key, labels, discoveredLabels}) => {
  const [areDiscoveredLabelsVisible, setAreDiscoveredLabelsVisible] = useState<boolean>(false);
  const allLabels = areDiscoveredLabelsVisible ? [...labels, ...discoveredLabels] : labels;
  const buttonClasses =
    'flex rounded-lg bg-gray-100 p-1 justify-center items-center mt-1 dark:bg-gray-700 dark:text-gray-300 cursor-pointer';

  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 flex flex-col w-96">
      <div className="flex flex-wrap">
        {allLabels.length > 0 &&
          allLabels.map((item: {name: string; value: string}) => {
            return (
              <div className="pb-1 pr-1">
                <Pill
                  key={item.name}
                  variant={'info' as PillVariant}
                >{`${item.name}="${item.value}"`}</Pill>
              </div>
            );
          })}
      </div>
      {areDiscoveredLabelsVisible ? (
        <div className={buttonClasses} onClick={() => setAreDiscoveredLabelsVisible(false)}>
          <span className="mr-1">Hide Discovered Labels</span>
          <ChevronDoubleUpIcon className="h-5 w-5" aria-hidden="true" />
        </div>
      ) : (
        <div className={buttonClasses} onClick={() => setAreDiscoveredLabelsVisible(true)}>
          <span className="mr-1">Show Discovered Labels</span>
          <ChevronDoubleDownIcon className="h-5 w-5" aria-hidden="true" />
        </div>
      )}
    </td>
  );
};

export default LabelsCell;
