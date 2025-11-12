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
import cx from 'classnames';
import {Item, Menu, Separator, Submenu} from 'react-contexify';
import {Tooltip} from 'react-tooltip';

import {useParcaContext, useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {getLastItem} from '@parca/utilities';

import {useGraphTooltip} from '../../GraphTooltipArrow/useGraphTooltip';
import {useGraphTooltipMetaInfo} from '../../GraphTooltipArrow/useGraphTooltipMetaInfo';
import {hexifyAddress, truncateString} from '../../utils';

interface ContextMenuProps {
  menuId: string;
  table: Table<any>;
  profileType?: ProfileType;
  unit?: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number;
  compareAbsolute: boolean;
  resetPath: () => void;
  hideMenu: () => void;
  hideBinary: (binaryToRemove: string) => void;
  isSandwich?: boolean;
}

const ContextMenu = ({
  menuId,
  table,
  total,
  totalUnfiltered,
  row,
  compareAbsolute,
  hideMenu,
  profileType,
  unit,
  hideBinary,
  resetPath,
  isSandwich = false,
}: ContextMenuProps): JSX.Element => {
  const {isDarkMode} = useParcaContext();
  const {enableSourcesView, enableSandwichView, checkDebuginfoStatusHandler} = useParcaContext();
  const [isGraphTooltipDocked, setIsDocked] = useUserPreference<boolean>(
    USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key
  );
  const contextMenuData = useGraphTooltip({
    table,
    profileType,
    unit,
    total,
    totalUnfiltered,
    row,
    compareAbsolute,
  });

  const {
    functionFilename,
    functionSystemName,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingFile,
    mappingBuildID,
    inlined,
  } = useGraphTooltipMetaInfo({table, row});

  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [sandwichFunctionName, setSandwichFunctionName] = useURLState<string | undefined>(
    'sandwich_function_name'
  );

  if (contextMenuData === null) {
    return <></>;
  }

  const {name, cumulativeText, diffText, diff} = contextMenuData;

  const isMappingBuildIDAvailable = mappingBuildID !== null && mappingBuildID !== '';

  const handleViewSourceFile = (): void => openFile();

  const handleResetView = (): void => {
    resetPath();
    return hideMenu();
  };
  const handleDockTooltip = (): void => {
    return isGraphTooltipDocked ? setIsDocked(false) : setIsDocked(true);
  };
  const handleCopyItem = (text: string): void => {
    if (navigator.clipboard?.writeText) {
      void navigator.clipboard.writeText(text).catch(() => {
        // Fallback to legacy method if modern API fails
        copyWithLegacyMethod(text);
      });
    } else {
      // Use legacy method if modern API not available
      copyWithLegacyMethod(text);
    }
  };

  const copyWithLegacyMethod = (text: string): void => {
    const textArea = document.createElement('textarea');
    textArea.value = text;
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();

    let successful = false;
    try {
      successful = document.execCommand('copy');
    } catch {
      // Copy failed
    } finally {
      document.body.removeChild(textArea);
    }

    if (!successful) {
      alert('Copy failed. Please copy manually: ' + text.substring(0, 100) + (text.length > 100 ? '...' : ''));
    }
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
    {
      id: 'Function system name',
      value: functionSystemName === functionName ? '' : functionSystemName,
    }, // an empty string will be filtered out below
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
    <Menu
      id={menuId}
      theme={isDarkMode ? 'dark' : ''}
      className={cx(
        dashboardItems.includes('sandwich')
          ? 'min-w-[350px] w-[350px]'
          : 'min-w-[260px] w-fit-content'
      )}
    >
      <Item
        id="view-source-file"
        onClick={handleViewSourceFile}
        disabled={enableSourcesView === false || !isSourceAvailable}
      >
        <div
          data-tooltip-id="view-source-file-help"
          data-tooltip-content="There is no source code uploaded for this build"
        >
          <div className="flex w-full items-center gap-2">
            <Icon icon="wpf:view-file" />
            <div>View source file</div>
          </div>
        </div>
        {!isSourceAvailable ? <Tooltip id="view-source-file-help" /> : null}
      </Item>
      <Item
        id="show-in-table"
        onClick={() => {
          if (isSandwich) {
            setDashboardItems(['table']);
          } else {
            setDashboardItems([...dashboardItems, 'table']);
          }
        }}
      >
        <div className="flex w-full items-center gap-2">
          <Icon icon="ph:table" />
          <div>Show in table</div>
        </div>
      </Item>
      {enableSandwichView === true && (
        <Item
          id="show-in-sandwich"
          onClick={() => {
            if (functionName === '' || functionName == null) {
              return;
            }

            if (dashboardItems.includes('sandwich')) {
              setSandwichFunctionName(functionName);
              return;
            }

            setSandwichFunctionName(functionName);
            setDashboardItems([...dashboardItems, 'sandwich']);
          }}
          disabled={functionName === '' || functionName == null}
        >
          <div className="flex w-full items-center gap-2">
            <Icon icon="tdesign:sandwich-filled" />
            <div className="relative">
              {dashboardItems.includes('sandwich')
                ? 'Focus sandwich on this frame.'
                : 'Show in sandwich'}
              <span className="absolute top-[-2px] text-xs lowercase text-red-500">
                &nbsp;alpha
              </span>
            </div>
          </div>
        </Item>
      )}
      <Item id="reset-view" onClick={handleResetView}>
        <div className="flex w-full items-center gap-2">
          <Icon icon="system-uicons:reset" />
          <div>Reset graph</div>
        </div>
      </Item>
      <Item
        id="hide-binary"
        onClick={() => hideBinary(getLastItem(mappingFile) as string)}
        disabled={mappingFile === null || mappingFile === ''}
      >
        <div
          data-tooltip-id="hide-binary-help"
          data-tooltip-content="Hide all frames for this binary"
        >
          <div className="flex w-full items-center gap-2">
            <Icon icon="bx:bxs-hide" />
            <div>
              Hide binary {mappingFile !== null && `(${getLastItem(mappingFile) as string})`}
            </div>
          </div>
        </div>
        <Tooltip place="left" id="hide-binary-help" />
      </Item>
      <Submenu
        label={
          <div className="flex w-full items-center gap-2">
            <Icon icon="ph:copy" />
            <div>Copy</div>
          </div>
        }
      >
        <div className="max-h-[300px] overflow-scroll">
          {nonEmptyValuesToCopy.map(({id, value}: {id: string; value: string}) => (
            <Item
              key={id}
              id={id}
              onClick={() => handleCopyItem(value)}
              className="dark:bg-gray-800"
            >
              <div className="flex flex-col dark:text-gray-300 hover:dark:text-gray-100">
                <div className="text-sm">{id}</div>
                <div className="text-xs">{truncateString(value, 30)}</div>
              </div>
            </Item>
          ))}
        </div>
      </Submenu>
      {checkDebuginfoStatusHandler !== undefined ? (
        <Item
          id="check-debuginfo-status"
          onClick={() => checkDebuginfoStatusHandler(mappingBuildID as string)}
          disabled={!isMappingBuildIDAvailable}
        >
          <div className="flex w-full items-center gap-2">
            <Icon icon="bx:bx-info-circle" />
            <div className="relative pr-4">Check debuginfo status</div>
          </div>
        </Item>
      ) : null}
      <Separator />
      <Item id="dock-tooltip" onClick={handleDockTooltip}>
        <div className="flex w-full items-center gap-2">
          <Icon icon="bx:dock-bottom" />
          {isGraphTooltipDocked ? 'Undock tooltip' : 'Dock tooltip'}
        </div>
      </Item>
    </Menu>
  );
};

export default ContextMenu;
