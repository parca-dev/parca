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

import {forwardRef, useImperativeHandle, useState} from 'react';

import {type Row} from '.';
import TableContextMenu from './TableContextMenu';
import {type ColumnName} from './utils/functions';

interface TableContextMenuWrapperProps {
  menuId: string;
  unit?: string;
  total?: bigint;
  totalUnfiltered?: bigint;
  columnVisibility?: Record<ColumnName, boolean>;
}

export interface TableContextMenuWrapperRef {
  setRow: (row: Row | null, callback?: () => void) => void;
}

const TableContextMenuWrapper = forwardRef<
  TableContextMenuWrapperRef,
  TableContextMenuWrapperProps
>(({menuId, unit, total, totalUnfiltered, columnVisibility}, ref) => {
  const [row, setRow] = useState<Row | null>(null);

  useImperativeHandle(ref, () => ({
    setRow: (newRow: Row | null, callback?: () => void) => {
      setRow(newRow);
      // Execute callback after state update using requestAnimationFrame
      if (callback != null) {
        requestAnimationFrame(callback);
      }
    },
  }));

  return (
    <TableContextMenu
      menuId={menuId}
      row={row}
      unit={unit}
      total={total}
      totalUnfiltered={totalUnfiltered}
      columnVisibility={columnVisibility}
    />
  );
});

TableContextMenuWrapper.displayName = 'TableContextMenuWrapper';

export default TableContextMenuWrapper;
