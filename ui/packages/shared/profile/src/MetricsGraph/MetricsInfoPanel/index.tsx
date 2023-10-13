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

import {Icon} from '@iconify/react';

interface MetricsInfoPanelProps {
  isInfoPanelOpen: boolean;
  onInfoIconClick: () => void;
}

const MetricsInfoPanel = ({
  isInfoPanelOpen,
  onInfoIconClick,
}: MetricsInfoPanelProps): JSX.Element => {
  const items: Array<{header: string; description: string; icon: string}> = [
    {
      header: 'Click',
      description: 'To select a profile at a specific point in time',
      icon: 'iconoir:mouse-button-left',
    },
    {
      header: 'Click & drag',
      description: 'To select profile samples over a period of time',
      icon: 'bi:arrows',
    },
    {
      header: 'Right click',
      description: 'To easily add labels to the query',
      icon: 'iconoir:mouse-button-right',
    },
  ];

  return (
    <div>
      {isInfoPanelOpen ? (
        <div className="flex flex-col items-end gap-1">
          <Icon
            icon="material-symbols:info"
            width={25}
            height={25}
            className="cursor-pointer text-gray-400"
          />
          <div className="items-space-around flex flex-col justify-start gap-4 rounded-md border border-gray-200 bg-gray-50 p-4 shadow-md dark:border-gray-500 dark:bg-gray-800">
            {items.map(({header, description, icon}) => (
              <div className="flex items-center gap-2" key={header}>
                <div>
                  <Icon
                    icon={icon}
                    width={30}
                    height={30}
                    className="text-indigo-600 dark:text-indigo-500"
                  />
                </div>
                <div className="flex flex-col items-start">
                  <div className="text-md font-medium dark:text-gray-300">{header}</div>
                  <div className="text-sm text-gray-600 dark:text-gray-400">{description}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <Icon
          icon="material-symbols:info-outline"
          width={25}
          height={25}
          onClick={onInfoIconClick}
          className="cursor-pointer text-gray-400"
        />
      )}
    </div>
  );
};

export default MetricsInfoPanel;
