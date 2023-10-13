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
import {Item, Menu, Submenu} from 'react-contexify';

import {Label} from '@parca/client';
import {selectDarkMode, useAppSelector} from '@parca/store';

import {HighlightedSeries} from '../';

interface MetricsContextMenuProps {
  menuId: string;
  onAddLabelMatcher: (
    labels: {key: string; value: string} | Array<{key: string; value: string}>
  ) => void;
  highlighted: HighlightedSeries | null;
  trackVisibility: (isVisible: boolean) => void;
}

const MetricsContextMenu = ({
  menuId,
  onAddLabelMatcher,
  highlighted,
  trackVisibility,
}: MetricsContextMenuProps): JSX.Element => {
  const isDarkMode = useAppSelector(selectDarkMode);
  const labels = highlighted?.labels.filter((label: Label) => label.name !== '__name__');

  const handleFocusOnSingleSeries = (): void => {
    const labelsToAdd = labels?.map((label: Label) => ({
      key: label.name,
      value: label.value,
    }));

    labelsToAdd !== undefined && onAddLabelMatcher(labelsToAdd);
  };

  return (
    <Menu id={menuId} onVisibilityChange={trackVisibility} className="dark:bg-gray-800">
      <Item
        id="focus-on-single-series"
        onClick={handleFocusOnSingleSeries}
        className="dark:bg-gray-800"
      >
        <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
          <Icon icon="ph:star" />
          <div>Focus only on this series</div>
        </div>
      </Item>
      <Submenu
        label={
          <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
            <Icon icon="material-symbols:add" />
            <div>Add to query</div>
          </div>
        }
        // Note: Submenu className prop does not change styles, so need to use style prop instead
        style={{
          maxHeight: '300px',
          overflow: 'scroll',
          backgroundColor: isDarkMode ? 'rgb(31 41 55)' : 'rgb(249 250 251)',
        }}
      >
        {labels?.map((label: Label) => (
          <Item
            key={label.name}
            id={label.name}
            onClick={() => onAddLabelMatcher({key: label.name, value: label.value})}
            className="max-w-[400px] overflow-hidden dark:bg-gray-800 dark:text-gray-300 hover:dark:text-gray-100"
          >
            <div className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-300">
              {`${label.name}="${label.value}"`}
            </div>
          </Item>
        ))}
      </Submenu>
    </Menu>
  );
};

export default MetricsContextMenu;
