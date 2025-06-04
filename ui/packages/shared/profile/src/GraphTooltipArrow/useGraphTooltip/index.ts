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

import {ProfileType} from '@parca/parser';
import {divide, valueFormatter} from '@parca/utilities';

import {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FLAT,
  FIELD_LOCATION_ADDRESS,
} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {getTextForCumulative, nodeLabel} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';

interface Props {
  table: Table<any>;
  profileType?: ProfileType;
  unit?: string;
  total: bigint;
  totalUnfiltered: bigint;
  compareAbsolute: boolean;
  row: number | null;
}

interface GraphTooltipData {
  name: string;
  locationAddress: bigint;
  cumulativeText: string;
  flatText: string;
  diffText: string;
  diff: bigint;
  row: number;
}

export const useGraphTooltip = ({
  table,
  profileType,
  unit,
  compareAbsolute,
  total,
  totalUnfiltered,
  row,
}: Props): GraphTooltipData | null => {
  if (row === null || profileType === undefined) {
    return null;
  }

  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;
  unit = unit ?? profileType.sampleUnit;

  const cumulative: bigint =
    table.getChild(FIELD_CUMULATIVE)?.get(row) !== null
      ? BigInt(table.getChild(FIELD_CUMULATIVE)?.get(row))
      : 0n;
  const flat: bigint =
    table.getChild(FIELD_FLAT)?.get(row) !== null
      ? BigInt(table.getChild(FIELD_FLAT)?.get(row))
      : 0n;
  const diff: bigint =
    table.getChild(FIELD_DIFF)?.get(row) !== null
      ? BigInt(table.getChild(FIELD_DIFF)?.get(row))
      : 0n;

  let diffText = '';
  const prevValue = cumulative - diff;
  const diffRatio = diff !== 0n ? divide(diff, prevValue) : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';

  diffText = compareAbsolute ? `${diffValueText} (${diffPercentageText})` : diffPercentageText;

  const name = nodeLabel(table, row, false);

  return {
    name,
    locationAddress,
    cumulativeText: getTextForCumulative(cumulative, totalUnfiltered, total, unit ?? ''),
    flatText: getTextForCumulative(flat, totalUnfiltered, total, unit ?? ''),
    diffText,
    diff,
    row,
  };
};
