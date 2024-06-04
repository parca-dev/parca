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

import {flexRender} from '@tanstack/react-table';
import {
  createColumnHelper,
  type CellContext,
  type ColumnDef,
  type ExpandedState,
  type Row as RowType,
} from '@tanstack/table-core';
import {Int64, Vector, tableFromIPC, vectorFromArray} from 'apache-arrow';
import cx from 'classnames';
import {AnimatePresence, motion} from 'framer-motion';

import {
  Button,
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {type RowRendererProps} from '@parca/components/dist/Table';
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
import {getTopAndBottomExpandedRowModel} from './utils/topAndBottomExpandedRowModel';

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

export interface DataRow {
  id: number;
  name: string;
  flat: bigint;
  flatDiff: bigint;
  cumulative: bigint;
  cumulativeDiff: bigint;
  mappingFile: string;
  functionSystemName: string;
  functionFileName: string;
  callers?: DataRow[];
  callees?: DataRow[];
  subRows?: Row[];
  isTopSubRow?: boolean;
  isBottomSubRow?: boolean;
}

interface DummyRow {
  size: number;
  message?: string;
  isTopSubRow?: boolean;
  isBottomSubRow?: boolean;
}

export type Row = DataRow | DummyRow;

const isDummyRow = (row: Row): row is DummyRow => {
  return 'size' in row;
};

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

const rowBgClassNames = (isExpanded: boolean, isSubRow: boolean): Record<string, boolean> => {
  return {
    'bg-indigo-100 dark:bg-gray-600': isSubRow,
    'bg-indigo-50 dark:bg-gray-700': isExpanded,
  };
};

const ROW_HEIGHT = 29;

const sizeToHeightStyle = (size: number): Record<string, string> => {
  return {
    height: size * ROW_HEIGHT + 'px',
  };
};

const sizeToWidthStyle = (size: number): Record<string, string> => {
  return {
    width: size * ROW_HEIGHT + 'px',
  };
};

const sizeToTopStyle = (size: number): Record<string, string> => {
  return {
    top: size * ROW_HEIGHT + 10 + 'px',
  };
};

const getCallerLabelWidthStyle = (subRows: Row[]): Record<string, string> => {
  let callerRows = subRows.filter(row => row.isTopSubRow).length;
  if (callerRows < 3) {
    callerRows = 3;
  }

  return sizeToWidthStyle(callerRows);
};

const getCalleeLabelWidthStyle = (subRows: Row[]): Record<string, string> => {
  let calleeRows = subRows.filter(row => row.isBottomSubRow).length;
  if (calleeRows < 3) {
    calleeRows = 3;
  }

  return {...sizeToWidthStyle(calleeRows), ...sizeToTopStyle(calleeRows)};
};

const CustomRowRenderer = ({
  row,
  usePointerCursor,
  onRowClick,
  onRowDoubleClick,
  enableHighlighting,
  shouldHighlightRow,
  rows,
}: RowRendererProps<Row>): React.JSX.Element => {
  const data = row.original;
  const isExpanded = row.getIsExpanded();
  const _isSubRow = isSubRow(data);
  const bgClassNames = rowBgClassNames(isExpanded, _isSubRow);
  if (isDummyRow(data)) {
    return (
      <tr key={row.id} className={cx(bgClassNames)}>
        <td colSpan={100} className={`text-center`} style={sizeToHeightStyle(data.size)}>
          {data.message}
        </td>
      </tr>
    );
  }

  return (
    <tr
      key={row.id}
      className={cx(
        usePointerCursor === true ? 'cursor-pointer' : 'cursor-auto',
        'relative',
        bgClassNames,
        {
          'hover:bg-[#62626212] dark:hover:bg-[#ffffff12] ': !isExpanded && !_isSubRow,
          'hover:bg-indigo-200 dark:hover:bg-indigo-500': isExpanded || _isSubRow,
        }
      )}
      onClick={onRowClick != null ? () => onRowClick(row.original) : undefined}
      onDoubleClick={onRowDoubleClick != null ? () => onRowDoubleClick(row, rows) : undefined}
      style={
        enableHighlighting !== true || shouldHighlightRow === undefined
          ? undefined
          : {opacity: shouldHighlightRow(row.original) ? 1 : 0.5}
      }
    >
      {row.getVisibleCells().map((cell, idx) => {
        return (
          <td
            key={cell.id}
            className={cx('p-1.5 align-top', {
              /* @ts-expect-error */
              'text-right': cell.column.columnDef.meta?.align === 'right',
              /* @ts-expect-error */
              'text-left': cell.column.columnDef.meta?.align === 'left',
            })}
          >
            {idx === 0 && isExpanded ? (
              <>
                <div
                  className={`absolute top-0 left-0 bg-white dark:bg-indigo-500 px-1 uppercase -rotate-90 origin-top-left z-10 text-[10px] border-l border-y border-gray-200 dark:border-gray-700 text-left `}
                  style={getCallerLabelWidthStyle(row.originalSubRows ?? [])}
                >
                  Callers {'->'}
                </div>
                <div
                  className={`absolute left-[18px] bg-white dark:bg-indigo-500 px-1 uppercase -rotate-90 origin-bottom-left z-10 text-[10px] border-r border-y border-gray-200 dark:border-gray-700 `}
                  style={getCalleeLabelWidthStyle(row.originalSubRows ?? [])}
                >
                  {'<-'} Callees
                </div>
              </>
            ) : null}
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        );
      })}
    </tr>
  );
};

const getCallerRows = (callers: DataRow[]): Row[] => {
  if (callers.length === 0) {
    return [{size: 3, message: 'No callers.', isTopSubRow: true}];
  }

  const rows = callers.map(row => {
    return {...row, isTopSubRow: true};
  });
  if (rows.length >= 3) {
    return rows;
  }

  return [...rows, {size: 3 - rows.length, message: '', isTopSubRow: true}];
};

const getCalleeRows = (callees: DataRow[]): Row[] => {
  if (callees.length === 0) {
    return [{size: 3, message: 'No callees.', isBottomSubRow: true}];
  }

  const rows = callees.map(row => {
    return {...row, isBottomSubRow: true};
  });
  if (rows.length >= 3) {
    return rows;
  }

  return [{size: 3 - rows.length, message: '', isBottomSubRow: true}, ...rows];
};

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
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [scrollToIndex, setScrollToIndex] = useState<number | undefined>(undefined);

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

  const columnHelper = createColumnHelper<Row>();

  const columns = useMemo<Array<ColumnDef<Row>>>(() => {
    return [
      columnHelper.accessor('flat', {
        id: 'flat',
        header: 'Flat',
        cell: info =>
          valueFormatter(
            (info as CellContext<DataRow, bigint>).getValue(),
            profileType?.sampleUnit ?? '',
            2
          ),
        size: 80,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flat', {
        id: 'flatPercentage',
        header: 'Flat (%)',
        cell: info => {
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return ratioString((info as CellContext<DataRow, bigint>).getValue());
        },
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flatDiff', {
        id: 'flatDiff',
        header: 'Flat Diff',
        cell: info =>
          addPlusSign(
            valueFormatter(
              (info as CellContext<DataRow, bigint>).getValue(),
              profileType?.sampleUnit ?? '',
              2
            )
          ),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flatDiff', {
        id: 'flatDiffPercentage',
        header: 'Flat Diff (%)',
        cell: info => {
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return ratioString((info as CellContext<DataRow, bigint>).getValue());
        },
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        id: 'cumulative',
        header: 'Cumulative',
        cell: info =>
          valueFormatter(
            (info as CellContext<DataRow, bigint>).getValue(),
            profileType?.sampleUnit ?? '',
            2
          ),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        id: 'cumulativePercentage',
        header: 'Cumulative (%)',
        cell: info => {
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return ratioString((info as CellContext<DataRow, bigint>).getValue());
        },
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulativeDiff', {
        id: 'cumulativeDiff',
        header: 'Cumulative Diff',
        cell: info =>
          addPlusSign(
            valueFormatter(
              (info as CellContext<DataRow, bigint>).getValue(),
              profileType?.sampleUnit ?? '',
              2
            )
          ),
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulativeDiff', {
        id: 'cumulativeDiffPercentage',
        header: 'Cumulative Diff (%)',
        cell: info => {
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return ratioString((info as CellContext<DataRow, bigint>).getValue());
        },
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('name', {
        id: 'name',
        header: 'Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('functionSystemName', {
        id: 'functionSystemName',
        header: 'Function System Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('functionFileName', {
        id: 'functionFileName',
        header: 'Function File Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('mappingFile', {
        id: 'mappingFile',
        header: 'Mapping File',
        cell: info => info.getValue(),
      }),
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
      if (isDummyRow(row)) {
        return;
      }

      // If there is only one dashboard item, we don't want to select a span
      if (dashboardItems.length <= 1) {
        return;
      }
      selectSpan(row.name);
    },
    [selectSpan, dashboardItems.length]
  );

  const onRowDoubleClick = useCallback(
    (row: RowType<Row>, rows: Array<RowType<Row>>) => {
      if (isDummyRow(row.original)) {
        return;
      }
      if (!isSubRow(row.original)) {
        row.toggleExpanded();
        return;
      }
      // find the original row for this subrow and toggle it
      const newRow = rows.find(
        r =>
          !isDummyRow(r.original) &&
          !isDummyRow(row.original) &&
          r.original.name === row.original.name &&
          !isSubRow(r.original)
      );
      const parentRow = rows.find(r => {
        const parent = row.getParentRow()!;
        if (isDummyRow(parent.original) || isDummyRow(r.original)) {
          return false;
        }
        return r.original.name === parent.original.name;
      });
      if (parentRow == null || newRow == null) {
        return;
      }

      newRow.toggleExpanded();

      let scrollTarget = getScrollTargetIndex(rows, parentRow, newRow);
      setScrollToIndex(scrollTarget);
    },
    [scrollToIndex]
  );

  const shouldHighlightRow = useCallback(
    (row: Row) => {
      if (!('name' in row)) {
        return false;
      }
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
    setActionButtons?.(
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

  const table = useMemo(() => {
    if (loading || data == null) {
      return null;
    }

    return tableFromIPC(data);
  }, [data, loading]);

  const rows: DataRow[] = useMemo(() => {
    if (table == null || table.numRows === 0) {
      return [];
    }

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

    const getRow = (i: number): DataRow => {
      const flat: bigint = flatColumn?.get(i) ?? 0n;
      const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
      const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
      const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
      const functionSystemName: string = functionSystemNameColumn?.get(i) ?? '';
      const functionFileName: string = functionFileNameColumn?.get(i) ?? '';
      const mappingFile: string = mappingFileColumn?.get(i) ?? '';

      return {
        id: i,
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

    const rows: DataRow[] = [];
    for (let i = 0; i < table.numRows; i++) {
      const row = getRow(i);
      const callerIndices: Vector<Int64> = callersColumn?.get(i) ?? vectorFromArray([]);
      const callers: DataRow[] = Array.from(callerIndices.toArray().values()).map(rowIdx => {
        return getRow(Number(rowIdx));
      });

      const calleeIndices: Vector<Int64> = calleesColumn?.get(i) ?? vectorFromArray([]);
      const callees: DataRow[] = Array.from(calleeIndices.toArray().values()).map(rowIdx => {
        return getRow(Number(rowIdx));
      });

      row.callers = callers;
      row.callees = callees;
      row.subRows = [...getCallerRows(callers), ...getCalleeRows(callees)];

      rows.push(row);
    }

    return rows;
  }, [table]);

  if (loading) {
    return (
      <div className="overflow-clip h-[700px] min-h-[700px]">
        <TableSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
      </div>
    );
  }

  if (rows.length === 0) {
    return <div className="mx-auto text-center">Profile has no samples</div>;
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
              onRowDoubleClick={onRowDoubleClick}
              getSubRows={row => (isDummyRow(row) ? [] : row.subRows ?? [])}
              getCustomExpandedRowModel={getTopAndBottomExpandedRowModel}
              expandedState={expanded}
              onExpandedChange={getNewState => {
                // We only want the new expanded row so passing the exisitng state as empty
                // @ts-expect-error
                let newState = getNewState({});
                if (Object.keys(newState)[0] === Object.keys(expanded)[0]) {
                  newState = {};
                }
                setExpanded(newState);
              }}
              CustomRowRenderer={CustomRowRenderer}
              scrollToIndex={scrollToIndex}
              estimatedRowHeight={ROW_HEIGHT}
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

const getRowsCount = (rows: RowType<Row>[]): number => {
  if (rows.length < 6) {
    return 6;
  }

  return rows.length;
};

function getScrollTargetIndex(rows: RowType<Row>[], parentRow: RowType<Row>, newRow: RowType<Row>) {
  const parentIndex = rows.indexOf(parentRow);
  const newRowIndex = rows.indexOf(newRow);
  let targetIndex = newRowIndex;
  if (parentIndex > newRowIndex) {
    // Adjusting the number of subs rows to scroll to the main row after expansion.
    targetIndex -= getRowsCount(newRow.subRows);
  }
  if (parentIndex < newRowIndex) {
    // If the parent row is above the new row, we need to adjust the number of subrows of the parent.
    targetIndex += getRowsCount(parentRow.subRows);
  }
  if (targetIndex < 0) {
    targetIndex = 0;
  }
  return targetIndex;
}

function isSubRow(row: Row) {
  return row.isTopSubRow === true || row.isBottomSubRow === true;
}

export default Table;
