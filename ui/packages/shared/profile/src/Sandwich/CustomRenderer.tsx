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

import React, {useEffect, useRef, useState} from 'react';

import {flexRender} from '@tanstack/react-table';
import cx from 'classnames';

import {type RowRendererProps} from '@parca/components/dist/Table';

import {Row, isDummyRow} from '../Table';
import {
  ROW_HEIGHT,
  isFirstSubRow,
  isLastSubRow,
  isSubRow,
  rowBgClassNames,
  sizeToBottomStyle,
  sizeToHeightStyle,
  sizeToWidthStyle,
} from '../Table/utils/functions';

const CustomRowRenderer = ({
  row,
  usePointerCursor,
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
      onClick={
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

export default CustomRowRenderer;
