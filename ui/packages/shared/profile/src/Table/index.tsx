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

import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';

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
import {Tooltip} from 'react-tooltip';

import {
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {type RowRendererProps} from '@parca/components/dist/Table';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {getLastItem, isSearchMatch, valueFormatter} from '@parca/utilities';

import {getFilenameColors, getMappingColors} from '../ProfileIcicleGraph/IcicleGraphArrow/';
import {colorByColors} from '../ProfileIcicleGraph/IcicleGraphArrow/IcicleGraphNodes';
import useMappingList, {
  useFilenamesList,
} from '../ProfileIcicleGraph/IcicleGraphArrow/useMappingList';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {
  ColumnName,
  DataRow,
  DummyRow,
  ROW_HEIGHT,
  RowName,
  addPlusSign,
  getCalleeRows,
  getCallerRows,
  getRowColor,
  getScrollTargetIndex,
  isFirstSubRow,
  isLastSubRow,
  isSubRow,
  ratioString,
  rowBgClassNames,
  sizeToBottomStyle,
  sizeToHeightStyle,
  sizeToWidthStyle,
} from './utils/functions';
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

export type Row = DataRow | DummyRow;

export const isDummyRow = (row: Row): row is DummyRow => {
  return 'size' in row;
};

let doubleClickTimer: NodeJS.Timeout | null = null;

interface TableProps {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  loading: boolean;
  currentSearchString?: string;
  setSearchString?: (searchString: string) => void;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  isHalfScreen: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
}

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
  const _isLastSubRow = isLastSubRow(row, rows);
  const _isFirstSubRow = isFirstSubRow(row, rows);
  const bgClassNames = rowBgClassNames(isExpanded, _isSubRow);
  const ref = useRef<HTMLTableRowElement>(null);
  const [rowHeight, setRowHeight] = useState<number>(ROW_HEIGHT);

  useEffect(() => {
    if (ref.current != null) {
      setRowHeight(ref.current.clientHeight + 1); // +1 to account for the bottom border
    }
  }, []);

  const paddingElement = (
    <div
      className={cx(
        'bg-white dark:bg-indigo-500 w-[18px] absolute top-[-1px] left-0 border-x border-gray-200 dark:border-gray-700',
        {
          'border-b': _isLastSubRow,
          'border-t': _isFirstSubRow,
        }
      )}
      style={{height: `${rowHeight}px`}}
    />
  );

  if (isDummyRow(data)) {
    return (
      <tr key={row.id} className={cx(bgClassNames)} ref={ref}>
        {paddingElement}
        <td colSpan={100} className={`text-center`} style={sizeToHeightStyle(data.size)}>
          {data.message}
        </td>
      </tr>
    );
  }

  return (
    <tr
      key={row.id}
      ref={ref}
      className={cx(usePointerCursor === true ? 'cursor-pointer' : 'cursor-auto', bgClassNames, {
        'hover:bg-[#62626212] dark:hover:bg-[#ffffff12] ': !isExpanded && !_isSubRow,
        'hover:bg-indigo-200 dark:hover:bg-indigo-500': isExpanded || _isSubRow,
      })}
      onClick={e => {
        if (typeof onRowClick !== 'function') {
          return;
        }
        if (e.detail === 2 && doubleClickTimer != null) {
          // Prevent the click event from being triggered as it is part of a double click
          clearTimeout(doubleClickTimer);
          doubleClickTimer = null;
          return;
        }
        if (e.detail === 1) {
          // Schedule a single click event to be triggered after 150ms
          doubleClickTimer = setTimeout(() => {
            doubleClickTimer = null;
            onRowClick(row.original);
          }, 150);
        }
      }}
      onDoubleClick={
        onRowDoubleClick != null
          ? () => {
              onRowDoubleClick(row, rows);
              window.getSelection()?.removeAllRanges();
            }
          : undefined
      }
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
            className={cx('p-1.5 align-top relative', {
              /* @ts-expect-error */
              'text-right': cell.column.columnDef.meta?.align === 'right',
              /* @ts-expect-error */
              'text-left': cell.column.columnDef.meta?.align === 'left',
              'pl-5 whitespace-nowrap': idx === 0,
            })}
          >
            {idx === 0 && isExpanded ? (
              <>
                <div
                  className={`absolute top-0 left-0 bg-white dark:bg-indigo-500 px-1 uppercase -rotate-90 origin-top-left z-[9] text-[10px] border-l border-y border-gray-200 dark:border-gray-700 text-left`}
                  style={{...sizeToWidthStyle(3)}}
                >
                  Callers {'->'}
                </div>
                <div
                  className={`absolute left-[18px] bg-white dark:bg-indigo-500 px-1 uppercase -rotate-90 origin-bottom-left z-[9] text-[10px] border-r border-y border-gray-200 dark:border-gray-700`}
                  style={{
                    ...sizeToWidthStyle(3),
                    ...sizeToBottomStyle(3),
                  }}
                >
                  {'<-'} Callees
                </div>
              </>
            ) : null}
            {idx === 0 && _isSubRow ? paddingElement : null}
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        );
      })}
    </tr>
  );
};

export const Table = React.memo(function Table({
  data,
  total,
  filtered,
  profileType,
  loading,
  currentSearchString,
  setSearchString = () => {},
  isHalfScreen,
  unit,
  metadataMappingFiles,
}: TableProps): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });

  const [tableColumns] = useURLState<string[]>('table_columns', {
    alwaysReturnArray: true,
  });
  const [colorBy, setColorBy] = useURLState('color_by');
  const {isDarkMode} = useParcaContext();
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [scrollToIndex, setScrollToIndex] = useState<number | undefined>(undefined);

  const {compareMode} = useProfileViewContext();

  const table = useMemo(() => {
    if (loading || data == null) {
      return null;
    }

    return tableFromIPC(data);
  }, [data, loading]);

  const mappingsList = useMappingList(metadataMappingFiles);
  const filenamesList = useFilenamesList(table);
  const colorByValue = colorBy === undefined || colorBy === '' ? 'binary' : (colorBy as string);

  const mappingsListCount = useMemo(
    () => mappingsList.filter(m => m !== '').length,
    [mappingsList]
  );

  // If there is only one mapping file, we want to color by filename by default.
  useEffect(() => {
    if (mappingsListCount === 1 && colorBy !== 'filename') {
      setColorBy('filename');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mappingsListCount]);

  const filenameColors = useMemo(() => {
    const colors = getFilenameColors(filenamesList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, filenamesList, currentColorProfile]);

  const mappingColors = useMemo(() => {
    const colors = getMappingColors(mappingsList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, mappingsList, currentColorProfile]);

  const colorByList = {
    filename: filenameColors,
    binary: mappingColors,
  };

  type ColorByKey = keyof typeof colorByList;

  const colorByColors: colorByColors = colorByList[colorByValue as ColorByKey];

  const columnHelper = createColumnHelper<Row>();

  unit = useMemo(() => unit ?? profileType?.sampleUnit ?? '', [unit, profileType?.sampleUnit]);

  const columns = useMemo<Array<ColumnDef<Row>>>(() => {
    return [
      columnHelper.accessor('colorProperty', {
        id: 'color',
        header: '',
        cell: info => {
          const color = info.getValue() as {color: string; mappingFile: string};
          return (
            <>
              <div
                className="w-4 h-4 rounded-[4px]"
                style={{backgroundColor: color.color}}
                data-tooltip-id="table-color-tooltip"
                data-tooltip-content={getLastItem(color.mappingFile)}
              />
              <Tooltip id="table-color-tooltip" />
            </>
          );
        },
        size: 10,
      }),
      columnHelper.accessor('flat', {
        id: 'flat',
        header: 'Flat',
        cell: info => valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2),
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
          addPlusSign(valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2)),
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
        cell: info => valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2),
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
          addPlusSign(valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2)),
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
  }, [profileType, unit]);

  const [columnVisibility, setColumnVisibility] = useState(() => {
    return {
      color: true,
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

  useEffect(() => {
    if (Array.isArray(tableColumns)) {
      setColumnVisibility(prevState => {
        const newState = {...prevState};
        (Object.keys(newState) as ColumnName[]).forEach(column => {
          newState[column] = tableColumns.includes(column);
        });
        return newState;
      });
    }
  }, [tableColumns]);

  const selectSpan = useCallback(
    (span: string): void => {
      setSearchString(span.trim());
    },
    [setSearchString]
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

      setScrollToIndex(getScrollTargetIndex(rows, parentRow, newRow));
    },
    [setScrollToIndex]
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

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

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
        colorProperty: {
          color: getRowColor(
            colorByColors,
            mappingFileColumn,
            i,
            functionFileNameColumn,
            colorBy as string
          ),
          mappingFile,
        },
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
  }, [table, colorByColors, colorBy]);

  useEffect(() => {
    setTimeout(() => {
      if (currentSearchString == null || rows.length === 0) return;

      const firstHighlightedRowIndex = rows.findIndex(row => {
        return !isDummyRow(row) && isSearchMatch(currentSearchString, row.name);
      });

      if (firstHighlightedRowIndex !== -1) {
        setScrollToIndex(firstHighlightedRowIndex);
      }
    }, 1000); // Adding a delay to allow the table to render seems to be the only way to get this to work i.e. scrolling down to the highlighted row
  }, [currentSearchString, rows]);

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

export default Table;
