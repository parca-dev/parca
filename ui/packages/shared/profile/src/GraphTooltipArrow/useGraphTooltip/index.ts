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

import {divide, valueFormatter} from '@parca/utilities';

import {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_LOCATION_ADDRESS,
} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {getTextForCumulative, nodeLabel} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';

interface Props {
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
}

interface GraphTooltipData {
  name: string;
  locationAddress: bigint;
  cumulativeText: string;
  diffText: string;
  diff: bigint;
  row: number;
}

export const useGraphTooltip = ({
  table,
  unit,
  total,
  totalUnfiltered,
  row,
  level,
}: Props): GraphTooltipData | null => {
  if (row === null) {
    return null;
  }

  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;
  const cumulative: bigint = BigInt(table.getChild(FIELD_CUMULATIVE)?.get(row)) ?? 0n;
  const diff: bigint = BigInt(table.getChild(FIELD_DIFF)?.get(row)) ?? 0n;

  const prevValue = cumulative - diff;
  const diffRatio = diff !== 0n ? divide(diff, prevValue) : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;

  const name = nodeLabel(table, row, level, false);

  return {
    name,
    locationAddress,
    cumulativeText: getTextForCumulative(cumulative, totalUnfiltered, total, unit),
    diffText,
    diff,
    row,
  };
};
