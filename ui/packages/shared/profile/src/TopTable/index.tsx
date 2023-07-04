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

import {Top, TopNode, TopNodeMeta} from '@parca/client';
import {Button, Table, useURLState} from '@parca/components';
import {
  getLastItem,
  isSearchMatch,
  parseParams,
  valueFormatter,
  type NavigateFunction,
} from '@parca/utilities';

import {hexifyAddress} from '../utils';

interface TopTableProps {
  loading: boolean;
  data?: Top;
  sampleUnit: string;
  navigateTo?: NavigateFunction;
  currentSearchString?: string;
  setActionButtons?: (buttons: JSX.Element) => void;
}

export const RowLabel = (meta: TopNodeMeta | undefined): string => {
  if (meta === undefined) return '<unknown>';
  const mapping = `${
    meta?.mapping?.file !== undefined && meta?.mapping?.file !== ''
      ? `[${getLastItem(meta.mapping.file) ?? ''}]`
      : ''
  }`;
  if (meta.function?.name !== undefined && meta.function?.name !== '')
    return `${mapping} ${meta.function.name}`;

  const address = hexifyAddress(meta.location?.address);
  const fallback = `${mapping} ${address}`;

  return fallback === '' ? '<unknown>' : fallback;
};

const columnHelper = createColumnHelper<TopNode>();

const addPlusSign = (num: string): string => {
  if (num.charAt(0) === '0' || num.charAt(0) === '-') {
    return num;
  }

  return `+${num}`;
};

export const TopTable = React.memo(function TopTable({
  data: top,
  sampleUnit: unit,
  navigateTo,
  loading,
  currentSearchString,
  setActionButtons,
}: TopTableProps): JSX.Element {
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
    const cols: Array<ColumnDef<TopNode, any>> = [
      columnHelper.accessor('meta', {
        header: () => <span className="text-left">Name</span>,
        cell: info => {
          const meta = info.row.original.meta;
          const name = RowLabel(meta);
          return name;
        },
        sortingFn: (a, b) => {
          const aName = RowLabel(a.original.meta);
          const bName = RowLabel(b.original.meta);
          return aName.localeCompare(bName);
        },
      }),
      columnHelper.accessor('flat', {
        header: () => 'Flat',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        sortDescFirst: true,
      }),
      columnHelper.accessor('cumulative', {
        header: () => 'Cumulative',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        sortDescFirst: true,
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
          sortDescFirst: true,
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
    (row: TopNode) => {
      // If there is only one dashboard item, we don't want to select a span
      if (dashboardItems.length <= 1) {
        return;
      }

      const meta = row.meta;
      if (meta === undefined) {
        return;
      }
      const name = RowLabel(meta);
      selectSpan(name);
    },
    [selectSpan, dashboardItems.length]
  );

  const shouldHighlightRow = useCallback(
    (row: TopNode) => {
      const meta = row.meta;
      if (meta === undefined) return false;
      const name = RowLabel(meta);
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
    return [{id: compareMode ? 'diff' : 'cumulative', desc: true}];
  }, [compareMode]);

  const total = top != null ? top.list.length : 0;

  if (total === 0 && !loading) return <>Profile has no samples</>;

  return (
    <div className="relative">
      <div className="font-robotoMono h-[80vh] w-full overflow-scroll">
        <Table
          data={top?.list ?? []}
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

export default TopTable;
