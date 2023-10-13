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
import {Table} from 'apache-arrow';
import {Item, Menu, Separator, Submenu} from 'react-contexify';
import {Tooltip} from 'react-tooltip';

import {useParcaContext} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {selectDarkMode, useAppSelector} from '@parca/store';
import {type NavigateFunction} from '@parca/utilities';

import {useGraphTooltip} from '../../GraphTooltipArrow/useGraphTooltip';
import {useGraphTooltipMetaInfo} from '../../GraphTooltipArrow/useGraphTooltipMetaInfo';
import {hexifyAddress, truncateString} from '../../utils';

interface ContextMenuProps {
  menuId: string;
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number;
  level: number;
  navigateTo: NavigateFunction;
  trackVisibility: (isVisible: boolean) => void;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  hideMenu: () => void;
}

const ContextMenu = ({
  menuId,
  table,
  unit,
  total,
  totalUnfiltered,
  row,
  level,
  navigateTo,
  trackVisibility,
  curPath,
  setCurPath,
  hideMenu,
}: ContextMenuProps): JSX.Element => {
  const isDarkMode = useAppSelector(selectDarkMode);
  const {enableSourcesView} = useParcaContext();
  const [isGraphTooltipDocked, setIsDocked] = useUserPreference<boolean>(
    USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key
  );
  const contextMenuData = useGraphTooltip({
    table,
    unit,
    total,
    totalUnfiltered,
    row,
    level,
  });

  const {
    functionFilename,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingFile,
    mappingBuildID,
    inlined,
  } = useGraphTooltipMetaInfo({table, row, navigateTo});

  if (contextMenuData === null) {
    return <></>;
  }

  const {name, cumulativeText, diffText, diff} = contextMenuData;

  const isMappingBuildIDAvailable = mappingBuildID !== null && mappingBuildID !== '';

  const handleViewSourceFile = (): void => openFile();

  const handleResetView = (): void => {
    setCurPath([]);
    return hideMenu();
  };
  const handleDockTooltip = (): void => {
    return isGraphTooltipDocked ? setIsDocked(false) : setIsDocked(true);
  };
  const handleCopyItem = (text: string): void => {
    void navigator.clipboard.writeText(text);
  };

  const functionName =
    row === 0
      ? ''
      : name !== ''
      ? name
      : locationAddress !== 0n
      ? hexifyAddress(locationAddress)
      : '';

  const buildIdText = !isMappingBuildIDAvailable ? '' : mappingBuildID;
  const inlinedText = inlined === null ? 'merged' : inlined ? 'yes' : 'no';

  const valuesToCopy = [
    {id: 'Function name', value: functionName},
    {id: 'Cumulative', value: cumulativeText ?? ''},
    {id: 'Diff', value: diff !== 0n ? diffText : ''},
    {
      id: 'File',
      value: functionFilename === '' ? functionFilename : file,
    },
    {id: 'Address', value: locationAddress === 0n ? '' : hexifyAddress(locationAddress)},
    {id: 'Inlined', value: inlinedText},
    {id: 'Binary', value: mappingFile ?? ''},
    {id: 'Build Id', value: buildIdText},
  ];

  const nonEmptyValuesToCopy = valuesToCopy.filter(({value}) => value !== '');

  return (
    <Menu id={menuId} onVisibilityChange={trackVisibility} className="dark:bg-gray-800">
      <Item
        id="view-source-file"
        onClick={handleViewSourceFile}
        disabled={enableSourcesView === false || !isSourceAvailable}
        className="dark:bg-gray-800"
      >
        <div
          data-tooltip-id="view-source-file-help"
          data-tooltip-content="There is no source code uploaded for this build"
        >
          <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
            <Icon icon="wpf:view-file" />
            <div>View source file</div>
          </div>
        </div>
        {!isSourceAvailable ? <Tooltip id="view-source-file-help" /> : null}
      </Item>
      <Item
        id="reset-view"
        onClick={handleResetView}
        disabled={curPath.length === 0}
        className="dark:bg-gray-800"
      >
        <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
          <Icon icon="system-uicons:reset" />
          <div>Reset view</div>
        </div>
      </Item>
      <Submenu
        label={
          <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
            <Icon icon="ph:copy" />
            <div>Copy</div>
          </div>
        }
        // Note: Submenu className prop does not change styles, so need to use style prop instead
        style={{
          maxHeight: '300px',
          overflow: 'scroll',
          backgroundColor: isDarkMode ? 'rgb(31 41 55)' : 'rgb(249 250 251)',
        }}
      >
        {nonEmptyValuesToCopy.map(({id, value}: {id: string; value: string}) => (
          <Item key={id} id={id} onClick={() => handleCopyItem(value)} className="dark:bg-gray-800">
            <div className="flex flex-col dark:text-gray-300 hover:dark:text-gray-100">
              <div className="text-sm">{id}</div>
              <div className="text-xs">{truncateString(value, 30)}</div>
            </div>
          </Item>
        ))}
      </Submenu>
      <Separator />
      <Item id="dock-tooltip" onClick={handleDockTooltip} className="dark:bg-gray-800">
        <div className="flex w-full items-center gap-2 dark:text-gray-300 hover:dark:text-gray-100">
          <Icon icon="bx:dock-bottom" />
          {isGraphTooltipDocked ? 'Undock tooltip' : 'Dock tooltip'}
        </div>
      </Item>
    </Menu>
  );
};

export default ContextMenu;
