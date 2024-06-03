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

import {Row, RowData, RowModel, Table, getMemoOptions, memo} from '@tanstack/table-core';

export function getTopAndBottomExpandedRowModel<TData extends RowData>(): (
  table: Table<TData>
) => () => RowModel<TData> {
  return table =>
    memo(
      () => [
        table.getState().expanded,
        table.getPreExpandedRowModel(),
        table.options.paginateExpandedRows,
      ],
      (expanded, rowModel, paginateExpandedRows) => {
        if (!rowModel.rows.length || (expanded !== true && !Object.keys(expanded ?? {}).length)) {
          return rowModel;
        }

        if (!paginateExpandedRows) {
          // Only expand rows at this point if they are being paginated
          return rowModel;
        }

        return expandRows(rowModel);
      },
      getMemoOptions(table.options, 'debugTable', 'getExpandedRowModel')
    );
}

export function expandRows<TData extends RowData>(rowModel: RowModel<TData>) {
  const expandedRows: Row<TData>[] = [];

  const handleRow = (row: Row<TData>) => {
    if (!row.getIsExpanded()) {
      expandedRows.push(row);
      return;
    }

    // @ts-expect-error
    const topSubRows = (row.subRows ?? []).filter(subRow => subRow.original?.isTopSubRow);
    if (topSubRows.length) {
      topSubRows.forEach(handleRow);
    }
    expandedRows.push(row);

    // @ts-expect-error
    const bottomSubRows = (row.subRows ?? []).filter(subRow => subRow.original?.isBottomSubRow);

    if (bottomSubRows?.length) {
      // Needs to be split into dummy and non-dummy rows to ensure that the dummy rows are rendered at the bottom.
      // @ts-expect-error
      const dummyRows = bottomSubRows.filter(subRow => 'size' in subRow.original);
      // @ts-expect-error
      const nonDummyRows = bottomSubRows.filter(subRow => !('size' in subRow.original));
      nonDummyRows.forEach(handleRow);
      dummyRows.forEach(handleRow);
    }
  };

  rowModel.rows.forEach(handleRow);

  return {
    rows: expandedRows,
    flatRows: rowModel.flatRows,
    rowsById: rowModel.rowsById,
  };
}
