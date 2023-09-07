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

import React, {useCallback, useEffect, useMemo} from 'react';

import {createColumnHelper, type ColumnDef} from '@tanstack/react-table';
import {Table as ArrowTable, tableFromIPC} from 'apache-arrow';

import {Button, Table as TableComponent, useURLState} from '@parca/components';
import {
  getLastItem,
  isSearchMatch,
  parseParams,
  valueFormatter,
  type NavigateFunction,
} from '@parca/utilities';

import {hexifyAddress} from '../utils';

const FIELD_MAPPING_FILE = 'mapping_file';
const FIELD_LOCATION_ADDRESS = 'location_address';
const FIELD_FUNCTION_NAME = 'function_name';
const FIELD_FLAT = 'flat';
const FIELD_FLAT_DIFF = 'flat_diff';
const FIELD_CUMULATIVE = 'cumulative';
const FIELD_CUMULATIVE_DIFF = 'cumulative_diff';

const columnHelper = createColumnHelper<row>();

interface row {
  name: string;
  flat: bigint;
  flatDiff: bigint;
  cumulative: bigint;
  cumulativeDiff: bigint;
}

interface TableProps {
  data?: Uint8Array;
  sampleUnit: string;
  navigateTo?: NavigateFunction;
  loading: boolean;
  currentSearchString?: string;
  setActionButtons?: (buttons: React.JSX.Element) => void;
}

export const Table = React.memo(function Table({
  data,
  sampleUnit: unit,
  navigateTo,
  loading,
  currentSearchString,
  setActionButtons,
}: TableProps): React.JSX.Element {
  const router = parseParams(window?.location.search);
  const [rawDashboardItems] = useURLState({param: 'dashboard_items'});
  const [rawcompareMode] = useURLState({param: 'compare_a'});
  const [filterByFunctionInput] = useURLState({param: 'filter_by_function'});

  const compareMode: boolean = rawcompareMode === undefined ? false : rawcompareMode === 'true';

  const dashboardItems = useMemo(() => {
    if (rawDashboardItems !== undefined) {
      return rawDashboardItems as string[];
    }
    return ['icicle'];
  }, [rawDashboardItems]);

  const columns = useMemo(() => {
    const cols: Array<ColumnDef<row, any>> = [
      columnHelper.accessor('flat', {
        header: () => 'Flat',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 80,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flatDiff', {
        header: () => 'Flat Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        header: () => 'Cumulative',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 130,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulativeDiff', {
        header: () => 'Cumulative Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('name', {
        header: () => <span className="text-left">Name</span>,
        cell: info => info.getValue(),
      }),
    ];
    return cols;
  }, [unit]);

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
      dashboardItems.length > 1 ? (
        <Button
          color="neutral"
          onClick={clearSelection}
          className="w-auto"
          variant="neutral"
          disabled={currentSearchString === undefined || currentSearchString.length === 0}
        >
          Clear selection
        </Button>
      ) : (
        <></>
      )
    );
  }, [dashboardItems, clearSelection, currentSearchString, setActionButtons]);

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

  const columnVisibility = useMemo(() => {
    // TODO: Make this configurable via the UI and add more columns.
    return {
      flat: true,
      flatDiff: compareMode,
      cumulative: true,
      cumulativeDiff: compareMode,
      name: true,
    };
  }, [compareMode]);

  if (loading) return <div className="mx-auto text-center">Loading...</div>;
  if (data === undefined) return <div className="mx-auto text-center">Profile has no samples</div>;

  const table = tableFromIPC(data);
  const flatColumn = table.getChild(FIELD_FLAT);
  const flatDiffColumn = table.getChild(FIELD_FLAT_DIFF);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const cumulativeDiffColumn = table.getChild(FIELD_CUMULATIVE_DIFF);

  if (table.numRows === 0) return <div className="mx-auto text-center">Profile has no samples</div>;

  const rows: row[] = [];
  // TODO: Figure out how to only read the data of the columns we need for the virtualized table
  for (let i = 0; i < table.numRows; i++) {
    const flat: bigint = flatColumn?.get(i) ?? 0n;
    const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
    const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
    const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
    rows.push({
      name: RowName(table, i),
      flat,
      flatDiff,
      cumulative,
      cumulativeDiff,
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

const addPlusSign = (num: string): string => {
  if (num.charAt(0) === '0' || num.charAt(0) === '-') {
    return num;
  }

  return `+${num}`;
};

export const RowName = (table: ArrowTable, row: number): string => {
  const mappingFileColumn = table.getChild(FIELD_MAPPING_FILE);
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
  const functionName: string | null = table.getChild(FIELD_FUNCTION_NAME)?.get(row) ?? '';
  if (functionName !== null && functionName !== '') {
    return `${mapping} ${functionName}`;
  }

  const address: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0;

  return hexifyAddress(address);
};

export default Table;
