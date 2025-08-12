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

import {useCallback, useEffect, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import {
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type ExpandedState,
  type OnChangeFn,
  type Row,
  type RowModel,
  type SortingState,
  type Table as TableType,
  type VisibilityState,
} from '@tanstack/react-table';
import {useVirtualizer} from '@tanstack/react-virtual';
import {elementScroll} from '@tanstack/virtual-core';
import cx from 'classnames';

export interface RowRendererProps<TData> {
  row: Row<TData>;
  usePointerCursor?: boolean;
  onRowClick?: (row: TData) => void;
  onRowDoubleClick?: (row: Row<TData>, rows: Array<Row<TData>>) => void;
  onRowContextMenu?: (e: React.MouseEvent, row: TData) => void;
  enableHighlighting?: boolean;
  shouldHighlightRow?: (row: TData) => boolean;
  rows: Array<Row<TData>>;
}

const DefaultRowRenderer = ({
  row,
  usePointerCursor,
  onRowClick,
  onRowDoubleClick,
  onRowContextMenu,
  enableHighlighting,
  shouldHighlightRow,
  rows,
}: RowRendererProps<any>): JSX.Element => {
  return (
    <tr
      key={row.id}
      className={cx(
        usePointerCursor === true ? 'cursor-pointer' : 'cursor-auto',
        'hover:bg-[#62626212] dark:hover:bg-[#ffffff12]',
        {'bg-red-500': row.getIsExpanded()}
      )}
      onClick={onRowClick != null ? () => onRowClick(row.original) : undefined}
      onDoubleClick={onRowDoubleClick != null ? () => onRowDoubleClick(row, rows) : undefined}
      onContextMenu={onRowContextMenu != null ? e => onRowContextMenu(e, row.original) : undefined}
      style={
        enableHighlighting !== true || shouldHighlightRow === undefined
          ? undefined
          : {opacity: shouldHighlightRow(row.original) ? 1 : 0.5}
      }
    >
      {row.getVisibleCells().map(cell => {
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
            {flexRender(cell.column.columnDef.cell, cell.getContext())}
          </td>
        );
      })}
    </tr>
  );
};

interface Props<TData> {
  data: TData[];
  columns: Array<ColumnDef<TData>>;
  initialSorting?: SortingState;
  columnVisibility?: VisibilityState;
  onRowClick?: (row: TData) => void;
  onRowDoubleClick?: (row: Row<TData>, rows: Array<Row<TData>>) => void;
  onRowContextMenu?: (e: React.MouseEvent, row: TData) => void;
  enableHighlighting?: boolean;
  shouldHighlightRow?: (row: TData) => boolean;
  usePointerCursor?: boolean;
  className?: string;
  title?: string;
  emptyTableMessage?: string;
  getSubRows?: (originalRow: TData, index: number) => TData[];
  getCustomExpandedRowModel?: () => (table: TableType<TData>) => () => RowModel<TData>;
  expandedState?: ExpandedState;
  onExpandedChange?: OnChangeFn<ExpandedState> | undefined;
  CustomRowRenderer?: React.ComponentType<RowRendererProps<TData>> | null;
  scrollToIndex?: number;
  estimatedRowHeight?: number;
}

function easeInOutQuint(t: number): number {
  return t < 0.5 ? 16 * t * t * t * t * t : 1 + 16 * --t * t * t * t * t;
}

const Table = <T,>({
  data,
  columns,
  initialSorting = [],
  columnVisibility = {},
  onRowClick,
  onRowDoubleClick,
  onRowContextMenu,
  enableHighlighting = false,
  usePointerCursor = true,
  shouldHighlightRow,
  className = '',
  title = '',
  emptyTableMessage = '',
  getSubRows,
  getCustomExpandedRowModel,
  expandedState,
  onExpandedChange,
  CustomRowRenderer,
  scrollToIndex,
  estimatedRowHeight = 26,
}: Props<T>): JSX.Element => {
  const [sorting, setSorting] = useState<SortingState>(initialSorting);
  const tableContainerRef = useRef<HTMLDivElement>(null);
  const scrollingRef = useRef<number>();
  const isMounted = useRef(false);

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getExpandedRowModel: (getCustomExpandedRowModel ?? getExpandedRowModel)(),
    state: {
      sorting,
      columnVisibility,
      expanded: expandedState,
    },
    onSortingChange: setSorting,
    onExpandedChange,
    enableColumnResizing: true,
    defaultColumn: {
      // @ts-expect-error
      size: 'auto',
    },
    getSubRows,
  });

  const {rows} = table.getRowModel();

  const getActualRowIndex = useCallback(
    (targetIndex: number) => {
      const targetRow = data[targetIndex];
      if (targetRow === undefined) return -1;
      return rows.findIndex(row => row.original === targetRow);
    },
    [data, rows]
  );

  useEffect(() => {
    if (!isMounted.current) {
      isMounted.current = true;
      return;
    }

    if (scrollToIndex != null && scrollToIndex >= 0 && tableContainerRef.current !== null) {
      const actualIndex = getActualRowIndex(scrollToIndex);
      const container = tableContainerRef.current;

      // Always scroll, but if actualIndex is 0, scroll to top
      if (actualIndex === 0) {
        container.scrollTo({
          top: 0,
          behavior: 'smooth',
        });
      } else if (actualIndex > 0) {
        const targetPosition = actualIndex * estimatedRowHeight;
        const containerHeight = container.clientHeight;

        const scrollPosition = Math.max(
          0,
          targetPosition - containerHeight / 2 + estimatedRowHeight / 2
        );

        container.scrollTo({
          top: scrollPosition,
          behavior: 'smooth',
        });
      }
    }
  }, [scrollToIndex, getActualRowIndex, estimatedRowHeight]);

  const rowVirtualizer = useVirtualizer({
    count: rows.length,
    getScrollElement: () => tableContainerRef.current,
    overscan: 10,
    estimateSize: () => estimatedRowHeight ?? 26,
    scrollToFn: (offset, canSmooth, instance) => {
      const duration = 1000;
      const start = tableContainerRef.current?.scrollTop ?? 0;
      const startTime = (scrollingRef.current = Date.now());

      const run = (): void => {
        if (scrollingRef.current !== startTime) return;
        const now = Date.now();
        const elapsed = now - startTime;
        const progress = easeInOutQuint(Math.min(elapsed / duration, 1));
        const interpolated = start + (offset - start) * progress;

        if (elapsed < duration) {
          elementScroll(interpolated, canSmooth, instance);
          requestAnimationFrame(run);
        } else {
          elementScroll(interpolated, canSmooth, instance);
        }
      };

      requestAnimationFrame(run);
    },
  });
  const virtualRows = rowVirtualizer.getVirtualItems();

  const paddingTop: number = virtualRows.length > 0 ? virtualRows?.[0]?.start ?? 0 : 0;
  const paddingBottom: number =
    virtualRows.length > 0
      ? rowVirtualizer.getTotalSize() - (virtualRows?.[virtualRows.length - 1]?.end ?? 0)
      : 0;

  return (
    <div ref={tableContainerRef} className={cx('h-full overflow-auto pr-2', className)}>
      <table className="w-full">
        <thead className="sticky top-0 bg-gray-50 text-sm dark:bg-gray-800 z-[1]">
          {title.length > 0 ? (
            <tr>
              <th colSpan={columns.length} className="p-2 pl-4 text-left uppercase">
                {title}
              </th>
            </tr>
          ) : null}
          {table.getHeaderGroups().map(headerGroup => (
            <tr key={headerGroup.id}>
              {headerGroup.headers.map(header => {
                const isCurrentSortingColumn = sorting[0]?.id === header.id;
                const isDefaultSortDesc = header.column.getAutoSortDir() !== 'desc'; // It's weird why this is opposite but it only works correct this way.
                return (
                  <th
                    key={header.id}
                    className="cursor-pointer p-2"
                    onClick={() => {
                      setSorting([
                        {
                          id: header.id,
                          desc: isCurrentSortingColumn ? !sorting[0].desc : isDefaultSortDesc,
                        },
                      ]);
                    }}
                    style={{width: header.getSize()}}
                  >
                    <span
                      className={cx('flex items-center gap-2', {
                        /* @ts-expect-error */
                        'justify-start': header.column.columnDef.meta?.align === 'left',
                        /* @ts-expect-error */
                        'justify-end': header.column.columnDef.meta?.align === 'right',
                      })}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                      <Icon
                        icon={
                          isCurrentSortingColumn
                            ? sorting[0]?.desc
                              ? 'pepicons:triangle-up-filled'
                              : 'pepicons:triangle-down-filled'
                            : isDefaultSortDesc
                            ? 'pepicons:triangle-up-filled'
                            : 'pepicons:triangle-down-filled'
                        }
                        className={isCurrentSortingColumn ? undefined : 'opacity-50'}
                      />
                    </span>
                  </th>
                );
              })}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-gray-200 bg-white text-xs dark:divide-gray-700 dark:bg-gray-900">
          {paddingTop > 0 && (
            <tr>
              <td style={{height: `${paddingTop}px`}} />
            </tr>
          )}
          {rows.length === 0 && emptyTableMessage.length > 0 ? (
            <tr>
              <td colSpan={columns.length} className="p-2 text-center">
                {emptyTableMessage}
              </td>
            </tr>
          ) : null}
          {virtualRows.map(virtualRow => {
            const row = rows[virtualRow.index];
            if (CustomRowRenderer != null) {
              return (
                <CustomRowRenderer
                  key={row.id}
                  row={row}
                  enableHighlighting={enableHighlighting}
                  onRowDoubleClick={onRowDoubleClick}
                  onRowClick={onRowClick}
                  onRowContextMenu={onRowContextMenu}
                  shouldHighlightRow={shouldHighlightRow}
                  usePointerCursor={usePointerCursor}
                  rows={rows}
                />
              );
            }
            return (
              <DefaultRowRenderer
                key={row.id}
                row={row}
                enableHighlighting={enableHighlighting}
                onRowDoubleClick={onRowDoubleClick}
                onRowClick={onRowClick}
                onRowContextMenu={onRowContextMenu}
                shouldHighlightRow={shouldHighlightRow}
                usePointerCursor={usePointerCursor}
                rows={rows}
              />
            );
          })}
          {paddingBottom > 0 && (
            <tr>
              <td style={{height: `${paddingBottom}px`}} />
            </tr>
          )}
        </tbody>
        <tfoot>
          {table.getFooterGroups().map(footerGroup => (
            <tr key={footerGroup.id}>
              {footerGroup.headers.map(header => (
                <th key={header.id}>
                  {header.isPlaceholder
                    ? null
                    : flexRender(header.column.columnDef.footer, header.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </tfoot>
      </table>
    </div>
  );
};

export default Table;
