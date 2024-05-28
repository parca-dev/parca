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

import React, {useCallback, useEffect, useMemo, useState} from 'react';

import {Int64, Type, Vector, tableFromIPC, vectorFromArray} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';

import {
  Button,
  TableActionButtonPlaceholder,
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {
  getLastItem,
  isSearchMatch,
  parseParams,
  valueFormatter,
  type NavigateFunction,
} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import {hexifyAddress} from '../utils';
import ColumnsVisibility from './ColumnsVisibility';
import {NameCell} from './NameCell';

const FIELD_MAPPING_FILE = 'mapping_file';
const FIELD_LOCATION_ADDRESS = 'location_address';
const FIELD_FUNCTION_NAME = 'function_name';
const FIELD_FUNCTION_SYSTEM_NAME = 'function_system_name';
const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
const FIELD_FLAT = 'flat';
const FIELD_FLAT_DIFF = 'flat_diff';
const FIELD_CUMULATIVE = 'cumulative';
const FIELD_CUMULATIVE_DIFF = 'cumulative_diff';
const FIELD_CALLERS = 'callers';
const FIELD_CALLEES = 'callees';

export interface RowWithoutCallView {
  name: string;
  flat: bigint;
  flatDiff: bigint;
  cumulative: bigint;
  cumulativeDiff: bigint;
  mappingFile: string;
  functionSystemName: string;
  functionFileName: string;
}

export interface Row extends RowWithoutCallView {
  callers: RowWithoutCallView[];
  callees: RowWithoutCallView[];
}

export interface ColumnDef {
  id: string;
  header: string;
  accessorKey: string;
  footer?: string;
  cell?: (info: any) => string | number | React.ReactNode;
  meta?: {align: 'right' | 'left'};
  invertSorting?: boolean;
  size?: number;
}

interface TableProps {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  navigateTo?: NavigateFunction;
  loading: boolean;
  currentSearchString?: string;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  isHalfScreen: boolean;
}

export const Table = React.memo(function Table({
  data,
  total,
  filtered,
  profileType,
  navigateTo,
  loading,
  currentSearchString,
  setActionButtons,
  isHalfScreen,
}: TableProps): React.JSX.Element {
  const router = parseParams(window?.location.search);
  const [rawDashboardItems] = useURLState({param: 'dashboard_items'});
  const [filterByFunctionInput] = useURLState({param: 'filter_by_function'});
  const {isDarkMode} = useParcaContext();

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
        cell: info => valueFormatter(info.getValue(), profileType?.sampleUnit ?? '', 2),
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
        cell: info =>
          addPlusSign(valueFormatter(info.getValue(), profileType?.sampleUnit ?? '', 2)),
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
        cell: info => valueFormatter(info.getValue(), profileType?.sampleUnit ?? '', 2),
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
        cell: info =>
          addPlusSign(valueFormatter(info.getValue(), profileType?.sampleUnit ?? '', 2)),
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
        cell: NameCell,
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
  }, [profileType]);

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
    (row: Row) => {
      // If there is only one dashboard item, we don't want to select a span
      if (dashboardItems.length <= 1) {
        return;
      }
      selectSpan(row.name);
    },
    [selectSpan, dashboardItems.length]
  );

  const shouldHighlightRow = useCallback(
    (row: Row) => {
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
    if (loading && setActionButtons !== undefined) {
      setActionButtons(<TableActionButtonPlaceholder />);
      return;
    }

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
    loading,
  ]);

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

  if (loading)
    return (
      <div className="overflow-clip h-[700px] min-h-[700px]">
        <TableSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
      </div>
    );

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
  const callersColumn = table.getChild(FIELD_CALLERS);
  const calleesColumn = table.getChild(FIELD_CALLEES);

  const getRowWithoutCallView = (i: number): RowWithoutCallView => {
    const flat: bigint = flatColumn?.get(i) ?? 0n;
    const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
    const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
    const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
    const functionSystemName: string = functionSystemNameColumn?.get(i) ?? '';
    const functionFileName: string = functionFileNameColumn?.get(i) ?? '';
    const mappingFile: string = mappingFileColumn?.get(i) ?? '';

    return {
      name: RowName(mappingFileColumn, locationAddressColumn, functionNameColumn, i),
      flat,
      flatDiff,
      cumulative,
      cumulativeDiff,
      functionSystemName,
      functionFileName,
      mappingFile,
    };
  };

  const rows: Row[] = [];
  // TODO: Figure out how to only read the data of the columns we need for the virtualized table
  for (let i = 0; i < table.numRows; i++) {
    const row = getRowWithoutCallView(i);
    const callerIndices: Vector<Int64> = callersColumn?.get(i) ?? vectorFromArray([]);
    const callers: RowWithoutCallView[] = Array.from(callerIndices.toArray().values()).map(row => {
      return getRowWithoutCallView(Number(row));
    });

    const calleeIndices: Vector<Int64> = calleesColumn?.get(i) ?? vectorFromArray([]);
    const callees: RowWithoutCallView[] = Array.from(calleeIndices.toArray().values()).map(row => {
      return getRowWithoutCallView(Number(row));
    });

    rows.push({
      ...row,
      callers,
      callees,
    });
  }

  return (
    <AnimatePresence>
      <motion.div
        className="h-full w-full"
        key="table-loaded"
        initial={{display: 'none', opacity: 0}}
        animate={{display: 'block', opacity: 1}}
        transition={{duration: 0.5}}
      >
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
      </motion.div>
    </AnimatePresence>
  );
});

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
