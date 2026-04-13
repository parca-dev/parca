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
import cx from 'classnames';
import {Item, Menu, Submenu} from 'react-contexify';

import 'react-contexify/dist/ReactContexify.css';

import {useParcaContext, useURLState, useURLStateBatch} from '@parca/components';
import {valueFormatter} from '@parca/utilities';

import {type Row} from '.';
import {getTextForCumulative} from '../ProfileFlameGraph/FlameGraphArrow/utils';
import {truncateString} from '../utils';
import {type ColumnName} from './utils/functions';

interface TableContextMenuProps {
  menuId: string;
  row: Row | null;
  unit?: string;
  total?: bigint;
  totalUnfiltered?: bigint;
  columnVisibility?: Record<ColumnName, boolean>;
}

const TableContextMenu = ({
  menuId,
  row,
  unit,
  total,
  totalUnfiltered,
  columnVisibility,
}: TableContextMenuProps): React.JSX.Element => {
  const [_, setSandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const {enableSandwichView, isDarkMode} = useParcaContext();
  const batchUpdates = useURLStateBatch();

  const onSandwichViewSelect = (): void => {
    if (row?.name != null && row.name.length > 0) {
      // Batch updates to combine setSandwichFunctionName + setDashboardItems into single URL navigation
      batchUpdates(() => {
        setSandwichFunctionName(row.name.trim());
        if (!dashboardItems.includes('sandwich')) {
          setDashboardItems([...dashboardItems, 'sandwich']);
        }
      });
    }
  };

  const handleCopyItem = (text: string): void => {
    void navigator.clipboard.writeText(text);
  };

  const isColumnVisible = (columnName: ColumnName): boolean => {
    return columnVisibility?.[columnName] ?? true;
  };

  const valuesToCopy =
    row !== null
      ? [
          ...(isColumnVisible('flat')
            ? [
                {
                  id: 'Flat',
                  value:
                    total !== null &&
                    total !== undefined &&
                    totalUnfiltered !== null &&
                    totalUnfiltered !== undefined
                      ? getTextForCumulative(row.flat, total, totalUnfiltered, unit ?? '')
                      : valueFormatter(row.flat, unit ?? '', 1),
                },
              ]
            : []),
          ...(isColumnVisible('flatPercentage')
            ? [
                {
                  id: 'Flat (%)',
                  value:
                    total !== null &&
                    total !== undefined &&
                    totalUnfiltered !== null &&
                    totalUnfiltered !== undefined
                      ? getTextForCumulative(row.flat, total, totalUnfiltered, unit ?? '')
                      : valueFormatter(row.flat, unit ?? '', 1),
                },
              ]
            : []),
          ...(isColumnVisible('flatDiff')
            ? [
                {
                  id: 'Flat Diff',
                  value: row.flatDiff !== 0n ? valueFormatter(row.flatDiff, unit ?? '', 1) : '',
                },
              ]
            : []),
          ...(isColumnVisible('flatDiffPercentage')
            ? [
                {
                  id: 'Flat Diff (%)',
                  value: row.flatDiff !== 0n ? valueFormatter(row.flatDiff, unit ?? '', 1) : '',
                },
              ]
            : []),
          ...(isColumnVisible('cumulative')
            ? [
                {
                  id: 'Cumulative',
                  value:
                    total !== null &&
                    total !== undefined &&
                    totalUnfiltered !== null &&
                    totalUnfiltered !== undefined
                      ? getTextForCumulative(row.cumulative, total, totalUnfiltered, unit ?? '')
                      : valueFormatter(row.cumulative, unit ?? '', 1),
                },
              ]
            : []),
          ...(isColumnVisible('cumulativePercentage')
            ? [
                {
                  id: 'Cumulative (%)',
                  value:
                    total !== null &&
                    total !== undefined &&
                    totalUnfiltered !== null &&
                    totalUnfiltered !== undefined
                      ? getTextForCumulative(row.cumulative, total, totalUnfiltered, unit ?? '')
                      : valueFormatter(row.cumulative, unit ?? '', 1),
                },
              ]
            : []),
          ...(isColumnVisible('cumulativeDiff')
            ? [
                {
                  id: 'Cumulative Diff',
                  value:
                    row.cumulativeDiff !== 0n
                      ? valueFormatter(row.cumulativeDiff, unit ?? '', 1)
                      : '',
                },
              ]
            : []),
          ...(isColumnVisible('cumulativeDiffPercentage')
            ? [
                {
                  id: 'Cumulative Diff (%)',
                  value:
                    row.cumulativeDiff !== 0n
                      ? valueFormatter(row.cumulativeDiff, unit ?? '', 1)
                      : '',
                },
              ]
            : []),
          ...(isColumnVisible('name')
            ? [
                {
                  id: 'Name',
                  value: row.name ?? '',
                },
              ]
            : []),
          ...(isColumnVisible('functionSystemName')
            ? [{id: 'Function System Name', value: row.functionSystemName ?? ''}]
            : []),
          ...(isColumnVisible('functionFileName')
            ? [{id: 'Function File Name', value: row.functionFileName ?? ''}]
            : []),
          ...(isColumnVisible('mappingFile')
            ? [{id: 'Mapping File', value: row.mappingFile ?? ''}]
            : []),
        ].flat()
      : [];

  const nonEmptyValuesToCopy = valuesToCopy.filter(({value}) => value !== '');

  const isMenuDisabled = row === null || enableSandwichView !== true;

  return (
    <Menu
      id={menuId}
      theme={isDarkMode ? 'dark' : ''}
      className={cx(
        dashboardItems.includes('sandwich') ? 'min-w-[350px] w-[350px]' : 'min-w-[260px] w-[260px]'
      )}
    >
      <Item id="sandwich-view" onClick={onSandwichViewSelect} disabled={isMenuDisabled}>
        <div className="flex w-full items-center gap-2">
          <Icon icon="tdesign:sandwich-filled" />
          <div className="relative">
            {dashboardItems.includes('sandwich')
              ? 'Focus sandwich on this frame.'
              : 'Show in sandwich'}
            <span className="absolute top-[-2px] text-xs lowercase text-red-500">&nbsp;alpha</span>
          </div>
        </div>
      </Item>
      <Submenu
        label={
          <div className="flex w-full items-center gap-2">
            <Icon icon="ph:copy" />
            <div>Copy</div>
          </div>
        }
        disabled={row === null}
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
    </Menu>
  );
};

export default TableContextMenu;
