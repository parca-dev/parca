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

import {Table} from 'apache-arrow';

import {
  BINARY_FEATURE_TYPES,
  EVERYTHING_ELSE,
  FILENAMES_FEATURE_TYPES,
  type BinaryFeature,
  type FilenameFeature,
} from '@parca/store';
import {divide, getLastItem, valueFormatter} from '@parca/utilities';

import {MergedProfileSource, ProfileSource} from '../../ProfileSource';
import {BigIntDuo, hexifyAddress} from '../../utils';
import {
  FIELD_FUNCTION_NAME,
  FIELD_LABELS_ONLY,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from './index';

export function nodeLabel(
  table: Table<any>,
  row: number,
  level: number,
  showBinaryName: boolean
): string {
  const labelsOnly: boolean | null = table.getChild(FIELD_LABELS_ONLY)?.get(row);
  if (level === 1 && labelsOnly !== null && labelsOnly) {
    const labelPrefix = 'labels.';
    const labelColumnNames = table.schema.fields.filter(field =>
      field.name.startsWith(labelPrefix)
    );

    return labelColumnNames
      .map((field, i) => [
        labelColumnNames[i].name.slice(labelPrefix.length),
        arrowToString(table.getChild(field.name)?.get(row)) ?? '',
      ])
      .filter(value => value[1] !== '')
      .map(([k, v]) => `${k}="${v}"`)
      .join(', ');
  }

  const functionName: string | null = arrowToString(table.getChild(FIELD_FUNCTION_NAME)?.get(row));
  if (functionName !== null && functionName !== '') {
    return functionName;
  }

  let mappingString = '';
  if (showBinaryName) {
    const mappingFile: string | null = table.getChild(FIELD_MAPPING_FILE)?.get(row) ?? '';
    const binary: string | undefined = getLastItem(mappingFile ?? undefined);
    if (binary != null) mappingString = `[${binary}]`;
  }

  const addressBigInt: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row);
  const address = hexifyAddress(addressBigInt);
  const fallback = `${mappingString}${address}`;
  return fallback === '' ? '<unknown>' : fallback;
}

export const extractFeature = (mapping: string): BinaryFeature => {
  if (mapping != null && mapping !== '') {
    return {name: mapping, type: BINARY_FEATURE_TYPES.Binary};
  }

  return {name: EVERYTHING_ELSE, type: BINARY_FEATURE_TYPES.Misc};
};

export const extractFilenameFeature = (filename: string): FilenameFeature => {
  if (filename != null && filename !== '') {
    return {name: filename, type: FILENAMES_FEATURE_TYPES.Filename};
  }

  return {name: EVERYTHING_ELSE, type: FILENAMES_FEATURE_TYPES.Misc};
};

export const getTextForCumulative = (
  hoveringNodeCumulative: bigint,
  totalUnfiltered: bigint,
  total: bigint,
  unit: string
): string => {
  const filtered =
    totalUnfiltered > total
      ? ` / ${(100 * divide(hoveringNodeCumulative, total)).toFixed(2)}% of filtered`
      : '';
  return `${valueFormatter(hoveringNodeCumulative, unit, 2)}
    (${(100 * divide(hoveringNodeCumulative, totalUnfiltered)).toFixed(2)}%${filtered})`;
};

export const getTextForCumulativePerSecond = (
  hoveringNodeCumulative: number,
  unit: string
): string => {
  return `${valueFormatter(
    hoveringNodeCumulative,
    unit === 'nanoseconds' ? 'CPU Cores' : unit,
    5
  )}/s`;
};

export const arrowToString = (buffer: any): string | null => {
  if (buffer == null || typeof buffer === 'string') {
    return buffer;
  }
  if (ArrayBuffer.isView(buffer)) {
    return new TextDecoder().decode(buffer);
  }
  return '';
};

export const boundsFromProfileSource = (profileSource?: ProfileSource): BigIntDuo => {
  if (profileSource === undefined) {
    return [0n, 1n];
  }

  if (!(profileSource instanceof MergedProfileSource)) {
    return [0n, 1n];
  }

  const request = profileSource.QueryRequest();

  if (
    request.options.oneofKind !== 'merge' ||
    request.options.merge.start === undefined ||
    request.options.merge.end === undefined
  ) {
    return [0n, 1n];
  }

  const start =
    request.options.merge.start.seconds * 1000000000n + BigInt(request.options.merge.start.nanos);
  const end =
    request.options.merge.end.seconds * 1000000000n + BigInt(request.options.merge.end.nanos);

  return [start, end];
};
