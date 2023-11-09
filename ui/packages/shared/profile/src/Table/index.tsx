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

import React, {Fragment, useCallback, useEffect, useMemo, useState} from 'react';

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import {type VisibilityState} from '@tanstack/react-table';
import {Vector, tableFromIPC} from 'apache-arrow';

import {Button, Table as TableComponent, useURLState} from '@parca/components';
import {
  getLastItem,
  isSearchMatch,
  parseParams,
  valueFormatter,
  type NavigateFunction,
} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import {hexifyAddress} from '../utils';

const FIELD_MAPPING_FILE = 'mapping_file';
const FIELD_LOCATION_ADDRESS = 'location_address';
const FIELD_FUNCTION_NAME = 'function_name';
const FIELD_FUNCTION_SYSTEM_NAME = 'function_system_name';
const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
const FIELD_FLAT = 'flat';
const FIELD_FLAT_DIFF = 'flat_diff';
const FIELD_CUMULATIVE = 'cumulative';
const FIELD_CUMULATIVE_DIFF = 'cumulative_diff';

interface row {
  name: string;
  flat: bigint;
  flatDiff: bigint;
  cumulative: bigint;
  cumulativeDiff: bigint;
  mappingFile: string;
  functionSystemName: string;
  functionFileName: string;
}

interface ColumnDef {
  id: string;
  header: string;
  accessorKey: string;
  footer?: string;
  cell?: (info: any) => string | number;
  meta?: {align: 'right' | 'left'};
  invertSorting?: boolean;
  size?: number;
}

interface TableProps {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  navigateTo?: NavigateFunction;
  loading: boolean;
  currentSearchString?: string;
  setActionButtons?: (buttons: React.JSX.Element) => void;
}

export const Table = React.memo(function Table({
  data,
  total,
  filtered,
  sampleUnit: unit,
  navigateTo,
  loading,
  currentSearchString,
  setActionButtons,
}: TableProps): React.JSX.Element {
  const router = parseParams(window?.location.search);
  const [rawDashboardItems] = useURLState({param: 'dashboard_items'});
  const [filterByFunctionInput] = useURLState({param: 'filter_by_function'});

  const {compareMode} = useProfileViewContext();

  const dashboardItems = useMemo(() => {
    if (rawDashboardItems !== undefined) {
      return rawDashboardItems as string[];
    }
    return ['icicle'];
  }, [rawDashboardItems]);

  const percentageString = (value: bigint | number, total: bigint | number): string => {
    if (total === 0n) {
      return '0%';
    }

    const percentage = (Number(value) / Number(total)) * 100;
    return `${percentage.toFixed(2)}%`;
  };

  const ratioString = (value: bigint | number): string => {
    if (filtered === 0n) {
      return ` ${percentageString(value, total)}`;
    }

    return `${percentageString(value, total)} / ${percentageString(value, filtered)}`;
  };

  const columns = useMemo<ColumnDef[]>(() => {
    return [
      {
        id: 'flat',
        accessorKey: 'flat',
        header: 'Flat',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 80,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'flatPercentage',
        accessorKey: 'flat',
        header: 'Flat (%)',
        cell: info => ratioString(info.getValue()),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'flatDiff',
        accessorKey: 'flatDiff',
        header: 'Flat Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'flatDiffPercentage',
        accessorKey: 'flatDiff',
        header: 'Flat Diff (%)',
        cell: info => ratioString(info.getValue()),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'cumulative',
        accessorKey: 'cumulative',
        header: 'Cumulative',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'cumulativePercentage',
        accessorKey: 'cumulative',
        header: 'Cumulative (%)',
        cell: info => ratioString(info.getValue()),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'cumulativeDiff',
        accessorKey: 'cumulativeDiff',
        header: 'Cumulative Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'cumulativeDiffPercentage',
        accessorKey: 'cumulativeDiff',
        header: 'Cumulative Diff (%)',
        cell: info => ratioString(info.getValue()),
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'name',
        accessorKey: 'name',
        header: 'Name',
        cell: info => info.getValue(),
      },
      {
        id: 'functionSystemName',
        accessorKey: 'functionSystemName',
        header: 'Function System Name',
      },
      {
        id: 'functionFileName',
        accessorKey: 'functionFileName',
        header: 'Function File Name',
      },
      {
        id: 'mappingFile',
        accessorKey: 'mappingFile',
        header: 'Mapping File',
      },
    ];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unit]);

  const [columnVisibility, setColumnVisibility] = useState(() => {
    return {
      flat: true,
      flatPercentage: false,
      flatDiff: compareMode,
      flatDiffPercentage: false,
      cumulative: true,
      cumulativePercentage: false,
      cumulativeDiff: compareMode,
      cumulativeDiffPercentage: false,
      name: true,
      functionSystemName: false,
      functionFileName: false,
      mappingFile: false,
    };
  });

  const selectSpan = useCallback(
    (span: string): void => {
      if (navigateTo != null) {
        navigateTo(
          '/',
          {
            ...router,
            ...{search_string: span.trim()},
          },
          {replace: true}
        );
      }
    },
    [navigateTo, router]
  );

  const onRowClick = useCallback(
    (row: row) => {
      // If there is only one dashboard item, we don't want to select a span
      if (dashboardItems.length <= 1) {
        return;
      }
      selectSpan(row.name);
    },
    [selectSpan, dashboardItems.length]
  );

  const shouldHighlightRow = useCallback(
    (row: row) => {
      const name = row.name;
      return isSearchMatch(currentSearchString as string, name);
    },
    [currentSearchString]
  );

  const enableHighlighting = useMemo(() => {
    return currentSearchString != null && currentSearchString?.length > 0;
  }, [currentSearchString]);

  const clearSelection = useCallback((): void => {
    if (navigateTo != null) {
      navigateTo(
        '/',
        {
          ...router,
          ...{search_string: filterByFunctionInput ?? ''},
        },
        {replace: true}
      );
    }
  }, [navigateTo, router, filterByFunctionInput]);

  useEffect(() => {
    if (setActionButtons === undefined) {
      return;
    }
    setActionButtons(
      <>
        <ColumnsVisibility
          columns={columns}
          visibility={columnVisibility}
          setVisibility={(id, visible) => {
            setColumnVisibility({...columnVisibility, [id]: visible});
          }}
        />
        {dashboardItems.length > 1 && (
          <Button
            color="neutral"
            onClick={clearSelection}
            className="w-auto"
            variant="neutral"
            disabled={currentSearchString === undefined || currentSearchString.length === 0}
          >
            Clear selection
          </Button>
        )}
      </>
    );
  }, [
    dashboardItems,
    clearSelection,
    currentSearchString,
    setActionButtons,
    columns,
    columnVisibility,
  ]);

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

  if (loading) return <div className="mx-auto text-center">Loading...</div>;
  if (data === undefined) return <div className="mx-auto text-center">Profile has no samples</div>;

  const table = tableFromIPC(data);
  if (table.numRows === 0) return <div className="mx-auto text-center">Profile has no samples</div>;

  const flatColumn = table.getChild(FIELD_FLAT);
  const flatDiffColumn = table.getChild(FIELD_FLAT_DIFF);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const cumulativeDiffColumn = table.getChild(FIELD_CUMULATIVE_DIFF);
  const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
  const functionSystemNameColumn = table.getChild(FIELD_FUNCTION_SYSTEM_NAME);
  const functionFileNameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
  const mappingFileColumn = table.getChild(FIELD_MAPPING_FILE);
  const locationAddressColumn = table.getChild(FIELD_LOCATION_ADDRESS);

  const rows: row[] = [];
  // TODO: Figure out how to only read the data of the columns we need for the virtualized table
  for (let i = 0; i < table.numRows; i++) {
    const flat: bigint = flatColumn?.get(i) ?? 0n;
    const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
    const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
    const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
    const functionSystemName: string = functionSystemNameColumn?.get(i) ?? '';
    const functionFileName: string = functionFileNameColumn?.get(i) ?? '';
    const mappingFile: string = mappingFileColumn?.get(i) ?? '';
    rows.push({
      name: RowName(mappingFileColumn, locationAddressColumn, functionNameColumn, i),
      flat,
      flatDiff,
      cumulative,
      cumulativeDiff,
      functionSystemName,
      functionFileName,
      mappingFile,
    });
  }

  return (
    <div className="relative">
      <div className="font-robotoMono h-[80vh] w-full">
        <TableComponent
          data={rows}
          columns={columns}
          initialSorting={initialSorting}
          columnVisibility={columnVisibility}
          onRowClick={onRowClick}
          enableHighlighting={enableHighlighting}
          shouldHighlightRow={shouldHighlightRow}
          usePointerCursor={dashboardItems.length > 1}
        />
      </div>
    </div>
  );
});

const ColumnsVisibility = ({
  columns,
  visibility,
  setVisibility,
}: {
  columns: ColumnDef[];
  visibility: VisibilityState;
  setVisibility: (id: string, visible: boolean) => void;
}): React.JSX.Element => {
  return (
    <div>
      <Menu as="div" className="relative text-left">
        <div>
          <Menu.Button className="relative w-full cursor-default rounded-md border bg-white py-2 pl-3 pr-10 text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm">
            <span className="ml-3 block overflow-x-hidden text-ellipsis">Columns</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 ml-3 flex items-center pr-2 text-gray-400">
              <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
            </span>
          </Menu.Button>
        </div>

        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Menu.Items className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  {columns.map(col => (
                    <div key={col.id} className="relative flex items-start">
                      <div className="flex h-6 items-center">
                        <input
                          id={col.id}
                          name={col.id}
                          type="checkbox"
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                          checked={visibility[col.id ?? ''] ?? false}
                          onChange={() => {
                            setVisibility(col.id ?? '', !visibility[col.id ?? '']);
                          }}
                        />
                      </div>
                      <div className="ml-3 text-sm leading-6">
                        <label
                          htmlFor={col.id}
                          className="font-medium text-gray-900 dark:text-gray-200"
                        >
                          {col.header}
                        </label>
                      </div>
                    </div>
                  ))}
                </div>
              </fieldset>
            </div>
          </Menu.Items>
        </Transition>
      </Menu>
    </div>
  );
};

const addPlusSign = (num: string): string => {
  if (num.charAt(0) === '0' || num.charAt(0) === '-') {
    return num;
  }

  return `+${num}`;
};

export const RowName = (
  mappingFileColumn: Vector | null,
  locationAddressColumn: Vector | null,
  functionNameColumn: Vector | null,
  row: number
): string => {
  if (mappingFileColumn === null) {
    console.error('mapping_file column not found');
    return '';
  }

  const mappingFile: string | null = mappingFileColumn?.get(row);
  let mapping = '';
  // Show the last item in the mapping file only if there are more than 1 mappings
  if (mappingFile != null && mappingFileColumn.data.length > 1) {
    mapping = `[${getLastItem(mappingFile) ?? ''}]`;
  }
  const functionName: string | null = functionNameColumn?.get(row) ?? '';
  if (functionName !== null && functionName !== '') {
    return `${mapping} ${functionName}`;
  }

  const address: bigint = locationAddressColumn?.get(row) ?? 0;

  return hexifyAddress(address);
};

export default Table;
