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

import {type Row as RowType} from '@tanstack/table-core';
import {Column} from '@uwdata/flechette';

import {getLastItem} from '@parca/utilities';

import {colorByColors} from '../../ProfileFlameGraph/FlameGraphArrow/FlameGraphNodes';
import {hexifyAddress} from '../../utils';

export interface DataRow {
  id: number;
  moreActions?: string;
  colorProperty: {
    color: string;
    mappingFile: string;
  };
  name: string;
  flat: bigint;
  flatDiff: bigint;
  cumulative: bigint;
  cumulativeDiff: bigint;
  mappingFile: string;
  functionSystemName: string;
  functionFileName: string;
  callers?: DataRow[];
  callees?: DataRow[];
  subRows?: Row[];
  isTopSubRow?: boolean;
  isBottomSubRow?: boolean;
}

export interface DummyRow {
  size: number;
  message?: string;
  isTopSubRow?: boolean;
  isBottomSubRow?: boolean;
}

export type Row = DataRow | DummyRow;

export const addPlusSign = (num: string): string => {
  if (num.charAt(0) === '0' || num.charAt(0) === '-') {
    return num;
  }

  return `+${num}`;
};

export const getRowColor = (
  colorByColors: colorByColors,
  mappingFileColumn: Column<string> | null,
  row: number,
  functionFileNameColumn: Column<string> | null,
  colorBy: string
): string => {
  if (mappingFileColumn === null) {
    console.error('mapping_file column not found');
    return '';
  }

  const mappingFile: string | null = mappingFileColumn?.get(row) ?? null;
  const functionFileName: string | null = functionFileNameColumn?.get(row) ?? null;
  let mapping = '';
  let filename = '';

  if (mappingFile != null) {
    mapping = `${getLastItem(mappingFile) ?? ''}`;
  }

  if (functionFileName != null) {
    filename = `${getLastItem(functionFileName) ?? ''}`;
  }

  if (colorBy === 'filename') {
    return colorByColors[filename === '' ? 'Everything else' : filename];
  }

  return colorByColors[mapping];
};

export const RowName = (
  mappingFileColumn: Column<string> | null,
  locationAddressColumn: Column<bigint> | null,
  functionNameColumn: Column<string> | null,
  row: number
): string => {
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
  const functionName: string | null = functionNameColumn?.get(row) ?? '';
  if (functionName !== null && functionName !== '') {
    return `${mapping} ${functionName}`;
  }

  const address: bigint = locationAddressColumn?.get(row) ?? 0n;

  return hexifyAddress(address);
};

export const getRowsCount = (rows: Array<RowType<Row>>): number => {
  if (rows.length < 6) {
    return 6;
  }

  return rows.length;
};

export function getScrollTargetIndex(
  rows: Array<RowType<Row>>,
  parentRow: RowType<Row>,
  newRow: RowType<Row>
): number {
  const parentIndex = rows.indexOf(parentRow);
  const newRowIndex = rows.indexOf(newRow);
  let targetIndex = newRowIndex;
  if (parentIndex > newRowIndex) {
    // Adjusting the number of subs rows to scroll to the main row after expansion.
    targetIndex -= getRowsCount(newRow.subRows);
  }
  if (parentIndex < newRowIndex) {
    // If the parent row is above the new row, we need to adjust the number of subrows of the parent.
    targetIndex += getRowsCount(parentRow.subRows);
  }
  if (targetIndex < 0) {
    targetIndex = 0;
  }
  return targetIndex;
}

export function isSubRow(row: Row): boolean {
  return row.isTopSubRow === true || row.isBottomSubRow === true;
}

export function isLastSubRow(row: RowType<Row>, rows: Array<RowType<Row>>): boolean {
  const index = rows.indexOf(row);
  const nextRow = rows[index + 1];
  return nextRow == null || (!isSubRow(nextRow.original) && !nextRow.getIsExpanded());
}

export function isFirstSubRow(row: RowType<Row>, rows: Array<RowType<Row>>): boolean {
  const index = rows.indexOf(row);
  const prevRow = rows[index - 1];
  return prevRow == null || (!isSubRow(prevRow.original) && !prevRow.getIsExpanded());
}

export type ColumnName =
  | 'flat'
  | 'flatPercentage'
  | 'flatDiff'
  | 'flatDiffPercentage'
  | 'cumulative'
  | 'cumulativePercentage'
  | 'cumulativeDiff'
  | 'cumulativeDiffPercentage'
  | 'name'
  | 'functionSystemName'
  | 'functionFileName'
  | 'mappingFile';

export const rowBgClassNames = (
  isExpanded: boolean,
  isSubRow: boolean
): Record<string, boolean> => {
  return {
    relative: true,
    'bg-indigo-100 dark:bg-gray-600': isSubRow,
    'bg-indigo-50 dark:bg-gray-700': isExpanded,
  };
};

export const ROW_HEIGHT = 29;

export const sizeToHeightStyle = (size: number): Record<string, string> => {
  return {
    height: `${size * ROW_HEIGHT}px`,
  };
};

export const sizeToWidthStyle = (size: number): Record<string, string> => {
  return {
    width: `${size * ROW_HEIGHT}px`,
  };
};

export const sizeToBottomStyle = (size: number): Record<string, string> => {
  return {
    bottom: `-${size * ROW_HEIGHT}px`,
  };
};

export const getCallerRows = (callers: DataRow[]): Row[] => {
  if (callers.length === 0) {
    return [{size: 3, message: 'No callers.', isTopSubRow: true}];
  }

  const rows = callers.map(row => {
    return {...row, isTopSubRow: true};
  });
  if (rows.length >= 3) {
    return rows;
  }

  return [...rows, {size: 3 - rows.length, message: '', isTopSubRow: true}];
};

export const getCalleeRows = (callees: DataRow[]): Row[] => {
  if (callees.length === 0) {
    return [{size: 3, message: 'No callees.', isBottomSubRow: true}];
  }

  const rows = callees.map(row => {
    return {...row, isBottomSubRow: true};
  });
  if (rows.length >= 3) {
    return rows;
  }

  return [{size: 3 - rows.length, message: '', isBottomSubRow: true}, ...rows];
};

export const getPercentageString = (value: bigint | number, total: bigint | number): string => {
  if (total === 0n) {
    return '0%';
  }

  const percentage = (Number(value) / Number(total)) * 100;
  return `${percentage.toFixed(2)}%`;
};

export const getRatioString = (value: bigint | number, total: bigint, filtered: bigint): string => {
  if (filtered === 0n) {
    return ` ${getPercentageString(value, total)}`;
  }

  return `${getPercentageString(value, total)} / ${getPercentageString(value, filtered)}`;
};

export const possibleColumns = [
  'flat',
  'flatPercentage',
  'flatDiff',
  'flatDiffPercentage',
  'cumulative',
  'cumulativePercentage',
  'cumulativeDiff',
  'cumulativeDiffPercentage',
  'name',
  'functionSystemName',
  'functionFileName',
  'mappingFile',
];

export const percentageString = (value: bigint | number, total: bigint | number): string => {
  if (total === 0n) {
    return '0%';
  }

  const percentage = (Number(value) / Number(total)) * 100;
  return `${percentage.toFixed(2)}%`;
};

export const ratioString = (value: bigint | number, total: bigint, filtered: bigint): string => {
  if (filtered === 0n) {
    return ` ${percentageString(value, total)}`;
  }

  return `${percentageString(value, total)} / ${percentageString(value, filtered)}`;
};
