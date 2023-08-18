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

const columnHelper = createColumnHelper<row>();

interface row {
  name: string;
  flat: bigint;
  cumulative: bigint;
  diff: bigint;
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

  const compareMode: boolean = rawcompareMode === undefined ? false : rawcompareMode === 'true';

  const dashboardItems = useMemo(() => {
    if (rawDashboardItems !== undefined) {
      return rawDashboardItems as string[];
    }
    return ['icicle'];
  }, [rawDashboardItems]);

  const columns = useMemo(() => {
    const cols: Array<ColumnDef<row, any>> = [
      columnHelper.accessor('name', {
        header: () => <span className="text-left">Name</span>,
        cell: info => info.getValue(),
        // sortingFn: (a, b) => {
        //   const aName = RowLabel(a.original.meta);
        //   const bName = RowLabel(b.original.meta);
        //   return aName.localeCompare(bName);
        // },
      }),
      columnHelper.accessor('flat', {
        header: () => 'Flat',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        header: () => 'Cumulative',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
    ];
    if (compareMode) {
      cols.push(
        columnHelper.accessor('diff', {
          header: () => 'Diff',
          cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
          size: 150,
          meta: {
            align: 'right',
          },
          invertSorting: true,
        })
      );
    }
    return cols;
  }, [unit, compareMode]);

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
          ...{search_string: ''},
        },
        {replace: true}
      );
    }
  }, [navigateTo, router]);

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
        id: compareMode ? 'diff' : 'cumulative',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

  if (loading) return <>Loading...</>;
  if (data === undefined) return <>Profile has no samples</>;

  const table = tableFromIPC(data);
  const flatColumn = table.getChild('flat');
  const cumulativeColumn = table.getChild('cumulative');
  const cumulativeDiffColumn = table.getChild('cumulative_diff');

  if (table.numRows === 0) return <>Profile has no samples</>;

  const rows: row[] = [];
  // TODO: Figure out how to only read the data of the columns we need for the virtualized table
  for (let i = 0; i < table.numRows; i++) {
    const flat: bigint = flatColumn?.get(i) ?? 0n;
    const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
    const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
    rows.push({
      name: RowName(table, i),
      flat,
      cumulative,
      diff: cumulativeDiff,
    });
  }

  return (
    <div className="relative">
      <div className="font-robotoMono h-[80vh] w-full">
        <TableComponent
          data={rows}
          columns={columns}
          initialSorting={initialSorting}
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
  const mappingFileColumn = table.getChild('mapping_file');
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
  const functionName: string | null = table.getChild('function_name')?.get(row) ?? '';
  if (functionName !== null) {
    return `${mapping} ${functionName}`;
  }

  const address: bigint = table.getChild('location_address')?.get(row) ?? 0;

  return hexifyAddress(address);
};

export default Table;
