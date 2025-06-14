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

import {type ColumnDef, type Row as RowType} from '@tanstack/table-core';

import {Table as TableComponent} from '@parca/components';

import {type Row} from '../../Table';
import {ColumnName, DataRow, ROW_HEIGHT} from '../../Table/utils/functions';

interface TableSectionProps {
  rows: Row[];
  columns: Array<ColumnDef<Row>>;
  initialSorting: Array<{id: string; desc: boolean}>;
  selectedRow: RowType<Row> | null;
  onRowClick: (row: DataRow) => void;
  shouldHighlightRow: (row: Row) => boolean;
  enableHighlighting: boolean;
  columnVisibility: Record<ColumnName, boolean>;
  height?: number;
  sandwichFunctionName?: string;
}

export function TableSection({
  rows,
  columns,
  initialSorting,
  selectedRow,
  onRowClick,
  shouldHighlightRow,
  enableHighlighting,
  columnVisibility,
  height,
  sandwichFunctionName,
}: TableSectionProps): JSX.Element {
  console.log(height);
  return (
    <div
      style={{height: height !== undefined ? `${height}px` : '80vh'}}
      className={`font-robotoMono w-full cursor-pointer ${
        selectedRow != null && sandwichFunctionName !== undefined ? 'w-[50%]' : ''
      }`}
    >
      <TableComponent
        data={rows}
        columns={columns}
        initialSorting={initialSorting}
        usePointerCursor={true}
        onRowClick={onRowClick}
        shouldHighlightRow={shouldHighlightRow}
        enableHighlighting={enableHighlighting}
        estimatedRowHeight={ROW_HEIGHT}
        columnVisibility={columnVisibility}
      />
    </div>
  );
}
