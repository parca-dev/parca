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

import {useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';
import * as d3 from 'd3';
import isEqual from 'fast-deep-equal';

import {LabelSet} from '@parca/client';

import {TimelineGuide} from '../TimelineGuide';
import {NumberDuo} from '../utils';
import {AreaGraph, DataPoint} from './AreaGraph';

interface Props {
  cpus: LabelSet[];
  data: DataPoint[][];
  selectedTimeframe?: {
    labels: LabelSet;
    bounds: NumberDuo;
  };
  onSelectedTimeframe: (labels: LabelSet, bounds: NumberDuo | undefined) => void;
  width?: number;
  bounds: NumberDuo;
}

export const labelSetToString = (labelSet?: LabelSet): string => {
  if (labelSet === undefined) {
    return '{}';
  }

  let str = '{';

  let isFirst = true;
  for (const label of labelSet.labels) {
    if (!isFirst) {
      str += ', ';
      isFirst = false;
    }
    str += `${label.name}: ${label.value}`;
  }

  str += '}';

  return str;
};

const STRIP_HEIGHT = 24;

const getTimelineGuideHeight = (cpus: LabelSet[], collapsedIndices: number[]): number => {
  return (
    (STRIP_HEIGHT + 4) * (cpus.length - collapsedIndices.length) +
    20 * collapsedIndices.length +
    24 -
    6
  );
};

export const MetricsGraphStrips = ({
  cpus,
  data,
  selectedTimeframe,
  onSelectedTimeframe,
  width,
  bounds,
}: Props): JSX.Element => {
  const [collapsedIndices, setCollapsedIndices] = useState<number[]>([]);

  const color = d3.scaleOrdinal(d3.schemeObservable10);

  const valueBounds = d3.extent(data.flatMap(d => d.map(p => p.value))) as [number, number];

  return (
    <div className="flex flex-col gap-1 relative my-0 ml-[70px]" style={{width: width ?? '100%'}}>
      <TimelineGuide
        bounds={[BigInt(0), BigInt(bounds[1] - bounds[0])]}
        width={width ?? 1468}
        height={getTimelineGuideHeight(cpus, collapsedIndices)}
        margin={1}
      />
      {cpus.map((cpu, i) => {
        const isCollapsed = collapsedIndices.includes(i);
        const isSelected = isEqual(cpu, selectedTimeframe?.labels);
        const labelStr = labelSetToString(cpu);
        return (
          <div
            className={cx('min-h-5', {
              relative: !isSelected,
              'sticky z-30 bg-white dark:bg-black bg-opacity-75': isSelected,
            })}
            style={{width: width ?? 1468, top: isSelected ? 302 : undefined}}
            key={labelStr}
          >
            <div
              className="text-xs absolute top-0 left-0 flex gap-[2px] items-center bg-white/50 dark:bg-black/50 px-1 rounded-sm cursor-pointer"
              style={{
                zIndex: 15,
              }}
              onClick={() => {
                const newCollapsedIndices = [...collapsedIndices];
                if (collapsedIndices.includes(i)) {
                  newCollapsedIndices.splice(newCollapsedIndices.indexOf(i), 1);
                } else {
                  newCollapsedIndices.push(i);
                }
                setCollapsedIndices(newCollapsedIndices);
              }}
            >
              <Icon icon={isCollapsed ? 'bxs:right-arrow' : 'bxs:down-arrow'} />
              {labelStr}
            </div>
            {!isCollapsed ? (
              <AreaGraph
                data={data[i]}
                height={STRIP_HEIGHT}
                width={width ?? 1468}
                fill={color(labelStr)}
                selectionBounds={isSelected ? selectedTimeframe?.bounds : undefined}
                setSelectionBounds={bounds => {
                  onSelectedTimeframe(cpu, bounds);
                }}
                valueBounds={valueBounds}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
};
