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

import {MouseEventHandler, useCallback, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import {
  ExpandedState,
  Row,
  RowModel,
  flexRender,
  getCoreRowModel,
  getExpandedRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
  type Table as TableType,
  type VisibilityState,
} from '@tanstack/react-table';
import cx from 'classnames';
import {useVirtual} from 'react-virtual';

export interface RowRendererProps<TData> {
  row: Row<TData>;
  usePointerCursor?: boolean;
  onRowClick?: (row: TData) => void;
  getOnRowDoubleClick?: (row: Row<TData>) => MouseEventHandler<HTMLTableRowElement> | undefined;
  enableHighlighting?: boolean;
  shouldHighlightRow?: (row: TData) => boolean;
}

const DefaultRowRenderer = ({
  row,
  usePointerCursor,
  onRowClick,
  getOnRowDoubleClick,
  enableHighlighting,
  shouldHighlightRow,
}: RowRendererProps<any>): JSX.Element => {
  return (
    <tr
      key={row.id}
      className={cx(
        usePointerCursor ? 'cursor-pointer' : 'cursor-auto',
        'hover:bg-[#62626212] dark:hover:bg-[#ffffff12]',
        {'bg-red-500': row.getIsExpanded()}
      )}
      onClick={onRowClick != null ? () => onRowClick(row.original) : undefined}
      onDoubleClick={getOnRowDoubleClick != null ? getOnRowDoubleClick(row) : undefined}
      style={
        !enableHighlighting || shouldHighlightRow === undefined
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
  getOnRowDoubleClick?: (row: Row<TData>) => MouseEventHandler<HTMLTableRowElement> | undefined;
  enableHighlighting?: boolean;
  shouldHighlightRow?: (row: TData) => boolean;
  usePointerCursor?: boolean;
  className?: string;
  title?: string;
  emptyTableMessage?: string;
  getSubRows?: (originalRow: TData, index: number) => TData[];
  getCustomExpandedRowModel?: () => (table: TableType<TData>) => () => RowModel<TData>;
  expandedState?: ExpandedState;
  onExpandedChange?: (expanded: ExpandedState) => void;
  CustomRowRenderer?: React.ComponentType<RowRendererProps<TData>> | null;
}

const Table = <T,>({
  data,
  columns,
  initialSorting = [],
  columnVisibility = {},
  onRowClick,
  getOnRowDoubleClick,
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
}: Props<T>): JSX.Element => {
  const [sorting, setSorting] = useState<SortingState>(initialSorting);

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
    onExpandedChange: onExpandedChange,
    enableColumnResizing: true,
    defaultColumn: {
      // @ts-expect-error
      size: 'auto',
    },
    getSubRows,
  });

  const tableContainerRef = useRef<HTMLDivElement>(null);
  const {rows} = table.getRowModel();
  const rowVirtualizer = useVirtual({
    parentRef: tableContainerRef,
    size: rows.length,
    overscan: 10,
    estimateSize: useCallback(() => 26, []),
  });
  const {virtualItems: virtualRows, totalSize} = rowVirtualizer;

  const paddingTop: number = virtualRows.length > 0 ? virtualRows?.[0]?.start ?? 0 : 0;
  const paddingBottom: number =
    virtualRows.length > 0 ? totalSize - (virtualRows?.[virtualRows.length - 1]?.end ?? 0) : 0;

  return (
    <div ref={tableContainerRef} className={cx('h-full overflow-scroll pr-2', className)}>
      <table className="w-full">
        <thead className="sticky top-0 bg-gray-50 text-sm dark:bg-gray-800 z-10">
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
                              ? 'pepicons:triangle-down-filled'
                              : 'pepicons:triangle-up-filled'
                            : isDefaultSortDesc
                            ? 'pepicons:triangle-down-filled'
                            : 'pepicons:triangle-up-filled'
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
                  getOnRowDoubleClick={getOnRowDoubleClick}
                  onRowClick={onRowClick}
                  shouldHighlightRow={shouldHighlightRow}
                  usePointerCursor={usePointerCursor}
                />
              );
            }
            return (
              <DefaultRowRenderer
                key={row.id}
                row={row}
                enableHighlighting={enableHighlighting}
                getOnRowDoubleClick={getOnRowDoubleClick}
                onRowClick={onRowClick}
                shouldHighlightRow={shouldHighlightRow}
                usePointerCursor={usePointerCursor}
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
