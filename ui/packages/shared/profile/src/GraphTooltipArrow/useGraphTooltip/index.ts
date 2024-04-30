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
  FIELD_CUMULATIVE_PER_SECOND,
  FIELD_DIFF,
  FIELD_DIFF_PER_SECOND,
  FIELD_LOCATION_ADDRESS,
} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {
  getTextForCumulative,
  getTextForCumulativePerSecond,
  nodeLabel,
} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';

interface Props {
  table: Table<any>;
  profileType?: ProfileType;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
}

interface GraphTooltipData {
  name: string;
  locationAddress: bigint;
  cumulativeText: string;
  cumulativePerSecondText: string;
  diffText: string;
  diff: bigint;
  row: number;
}

export const useGraphTooltip = ({
  table,
  profileType,
  total,
  totalUnfiltered,
  row,
  level,
}: Props): GraphTooltipData | null => {
  if (row === null || profileType === undefined) {
    return null;
  }

  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;

  const cumulative: bigint =
    table.getChild(FIELD_CUMULATIVE)?.get(row) !== null
      ? BigInt(table.getChild(FIELD_CUMULATIVE)?.get(row))
      : 0n;
  const cumulativePerSecond: number =
    table.getChild(FIELD_CUMULATIVE_PER_SECOND)?.get(row) !== null
      ? table.getChild(FIELD_CUMULATIVE_PER_SECOND)?.get(row)
      : 0;
  const diff: bigint =
    table.getChild(FIELD_DIFF)?.get(row) !== null
      ? BigInt(table.getChild(FIELD_DIFF)?.get(row))
      : 0n;
  const diffPerSecond: number =
    table.getChild(FIELD_DIFF_PER_SECOND)?.get(row) !== null
      ? table.getChild(FIELD_DIFF_PER_SECOND)?.get(row)
      : 0;

  let diffText = '';
  if (profileType?.delta ?? false) {
    const prevValue = cumulativePerSecond - diffPerSecond;
    const diffRatio = diffPerSecond !== 0 ? diffPerSecond / prevValue : 0;
    const diffSign = diffPerSecond > 0 ? '+' : '';
    const diffValueText = diffSign + valueFormatter(diffPerSecond, 'CPU Cores', 5);
    const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
    diffText = `${diffValueText} (${diffPercentageText})`;
  } else {
    const prevValue = cumulative - diff;
    const diffRatio = diff !== 0n ? divide(diff, prevValue) : 0;
    const diffSign = diff > 0 ? '+' : '';
    const diffValueText = diffSign + valueFormatter(diff, profileType?.sampleUnit ?? '', 1);
    const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
    diffText = `${diffValueText} (${diffPercentageText})`;
  }

  const name = nodeLabel(table, row, level, false);

  return {
    name,
    locationAddress,
    cumulativeText: getTextForCumulative(
      cumulative,
      totalUnfiltered,
      total,
      profileType?.periodUnit ?? ''
    ),
    cumulativePerSecondText: getTextForCumulativePerSecond(
      cumulativePerSecond,
      profileType?.periodUnit ?? 'CPU Cores'
    ),
    diffText,
    diff,
    row,
  };
};
