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
import {CopyToClipboard} from 'react-copy-to-clipboard';
import {Tooltip} from 'react-tooltip';

import {useParcaContext} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {getLastItem, type NavigateFunction} from '@parca/utilities';

import {useGraphTooltip} from '../../GraphTooltipArrow/useGraphTooltip';
import {useGraphTooltipMetaInfo} from '../../GraphTooltipArrow/useGraphTooltipMetaInfo';
import {hexifyAddress, truncateString, truncateStringReverse} from '../../utils';

interface ContextMenuProps {
  menuId: string;
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
  navigateTo: NavigateFunction;
  trackVisibility: (isVisible: boolean) => void;
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
}: ContextMenuProps) => {
  const contextMenuData = useGraphTooltip({
    table,
    unit,
    total,
    totalUnfiltered,
    row,
    level,
  });

  if (contextMenuData === null) {
    return <></>;
  }

  const {name, cumulativeText, diffText, diff, row: rowNumber} = contextMenuData;
  const {
    functionFilename,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingFile,
    mappingBuildID,
  } = useGraphTooltipMetaInfo({table, row: rowNumber, navigateTo});
  const isMappingBuildIDAvailable = mappingBuildID !== null && mappingBuildID !== '';
  const {enableSourcesView} = useParcaContext();
  const [_, setIsDocked] = useUserPreference(USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key);

  const handleViewSourceFile = () => {
    () => openFile();
  };
  const handleResetView = () => {};
  const handleDockTooltip = () => {
    () => setIsDocked(true);
  };
  const handleCopyItem = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const functionName =
    row === 0
      ? ''
      : name !== ''
      ? name
      : locationAddress !== 0n
      ? hexifyAddress(locationAddress)
      : '';

  const buildIdValue = !isMappingBuildIDAvailable ? '' : mappingBuildID;

  const valuesToCopy = [
    {id: 'Function name', value: functionName},
    {id: 'Cumulative', value: cumulativeText || ''},
    {id: 'Diff', value: diff !== 0n ? diffText : ''},
    {
      id: 'File',
      value: functionFilename === '' ? functionFilename : file,
    },
    {id: 'Address', value: locationAddress === 0n ? '' : hexifyAddress(locationAddress)},
    {id: 'Binary', value: mappingFile || ''},
    {
      id: 'Build Id',
      value: buildIdValue,
    },
  ];

  const nonEmptyValuesToCopy = valuesToCopy.filter(({value}) => value !== '');

  return (
    <Menu id={menuId} onVisibilityChange={trackVisibility}>
      <Item
        id="view-source-file"
        onClick={handleViewSourceFile}
        disabled={enableSourcesView === false || !isSourceAvailable}
      >
        <div
          data-tooltip-id="view-source-file-help"
          data-tooltip-content="There is no source code uploaded for this build"
        >
          View source file
        </div>
        {!isSourceAvailable ? <Tooltip id="view-source-file-help" /> : null}
      </Item>
      <Item id="reset-view" onClick={handleResetView}>
        Reset view
      </Item>
      <Submenu label="Copy">
        {nonEmptyValuesToCopy.map(({id, value}: {id: string; value: string}) => (
          <Item key={id} id={id} onClick={() => handleCopyItem(value)}>
            {id}
          </Item>
        ))}
      </Submenu>
      <Separator />
      <Item id="dock-tooltip" onClick={handleDockTooltip}>
        <Icon icon="mdi:dock-bottom" />
        Dock tooltip
      </Item>
    </Menu>
  );
};

export default ContextMenu;
