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

import {type Table} from '@uwdata/flechette';

import {type colorByColors} from '../../ProfileFlameGraph/FlameGraphArrow/FlameGraphNodes';
import {
  FIELD_CUMULATIVE,
  FIELD_CUMULATIVE_DIFF,
  FIELD_FLAT,
  FIELD_FLAT_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_FUNCTION_SYSTEM_NAME,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../Table';
import {RowName, getRowColor, type DataRow} from '../../Table/utils/functions';

interface ProcessRowDataProps {
  table: Table | null;
  colorByColors: colorByColors;
  colorBy: string;
}

export function processRowData({table, colorByColors, colorBy}: ProcessRowDataProps): DataRow[] {
  if (table == null || table.numRows === 0) {
    return [];
  }

  const flatColumn = table.getChild(FIELD_FLAT);
  const flatDiffColumn = table.getChild(FIELD_FLAT_DIFF);
  const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
  const cumulativeDiffColumn = table.getChild(FIELD_CUMULATIVE_DIFF);
  const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
  const functionSystemNameColumn = table.getChild(FIELD_FUNCTION_SYSTEM_NAME);
  const functionFileNameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
  const mappingFileColumn = table.getChild(FIELD_MAPPING_FILE);
  const locationAddressColumn = table.getChild(FIELD_LOCATION_ADDRESS);

  const getRow = (i: number): DataRow => {
    const flat: bigint = flatColumn?.get(i) ?? 0n;
    const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
    const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
    const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
    const functionSystemName: string = functionSystemNameColumn?.get(i) ?? '';
    const functionFileName: string = functionFileNameColumn?.get(i) ?? '';
    const mappingFile: string = mappingFileColumn?.get(i) ?? '';

    return {
      id: i,
      colorProperty: {
        color: getRowColor(colorByColors, mappingFileColumn, i, functionFileNameColumn, colorBy),
        mappingFile,
      },
      name: RowName(mappingFileColumn, locationAddressColumn, functionNameColumn, i),
      flat,
      flatDiff,
      cumulative,
      cumulativeDiff,
      functionSystemName,
      functionFileName,
      mappingFile,
    };
  };

  return Array.from({length: table.numRows}, (_, i) => getRow(i));
}
