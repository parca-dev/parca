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

import {useCallback, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from '@tanstack/react-table';
import cx from 'classnames';
import {useVirtual} from 'react-virtual';

declare module '@tanstack/table-core' {
  // @ts-expect-error
  interface ColumnMeta {
    align: 'left' | 'right';
  }
}

interface Props<TData> {
  data: TData[];
  columns: Array<ColumnDef<TData>>;
  initialSorting?: SortingState;
  onRowClick?: (row: TData) => void;
  enableHighlighting?: boolean;
  shouldHighlightRow?: (row: TData) => boolean;
  usePointerCursor?: boolean;
}

const Table = <T,>({
  data,
  columns,
  initialSorting = [],
  onRowClick,
  enableHighlighting = false,
  usePointerCursor = true,
  shouldHighlightRow,
}: Props<T>): JSX.Element => {
  const [sorting, setSorting] = useState<SortingState>(initialSorting);

  const table = useReactTable({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    state: {
      sorting,
    },
    onSortingChange: setSorting,
    enableColumnResizing: true,
    defaultColumn: {
      // @ts-expect-error
      size: 'auto',
    },
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
    <div ref={tableContainerRef} className="h-full overflow-scroll pr-2">
      <table className="w-full">
        <thead className="bg-gray-50 text-sm dark:bg-gray-800">
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
                        'justify-start': header.column.columnDef.meta?.align === 'left',
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
          {virtualRows.map(virtualRow => {
            const row = rows[virtualRow.index];
            return (
              <tr
                key={row.id}
                className={cx(
                  usePointerCursor ? 'cursor-pointer' : 'cursor-auto',
                  'hover:bg-[#62626212] dark:hover:bg-[#ffffff12]'
                )}
                onClick={onRowClick != null ? () => onRowClick(row.original) : undefined}
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
                      className={cx('p-1.5', {
                        'text-right': cell.column.columnDef.meta?.align === 'right',
                        'text-left': cell.column.columnDef.meta?.align === 'left',
                      })}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  );
                })}
              </tr>
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
