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

import {useMemo, useState} from 'react';

import {Icon} from '@iconify/react';
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

const getTimelineGuideHeight = (cpus: LabelSet[], collapsedIndices: number[]): number => {
  return 56 * (cpus.length - collapsedIndices.length) + 20 * collapsedIndices.length + 24;
};

export const MetricsGraphStrips = ({
  cpus,
  data,
  selectedTimeframe,
  onSelectedTimeframe,
  width,
}: Props): JSX.Element => {
  const [collapsedIndices, setCollapsedIndices] = useState<number[]>([]);

  // @ts-expect-error
  const color = d3.scaleOrdinal(d3.schemeObservable10);

  const bounds = useMemo(() => {
    const bounds: NumberDuo = data.length > 0 ? [Infinity, -Infinity] : [0, 1];
    data.forEach(cpuData => {
      cpuData.forEach(dataPoint => {
        bounds[0] = Math.min(bounds[0], dataPoint.timestamp);
        bounds[1] = Math.max(bounds[1], dataPoint.timestamp);
      });
    });
    return [0, bounds[1] - bounds[0]] as NumberDuo;
  }, [data]);

  return (
    <div className="flex flex-col gap-1 relative my-0 ml-[70px]" style={{width: width ?? '100%'}}>
      <TimelineGuide
        bounds={[BigInt(bounds[0]), BigInt(bounds[1])]}
        width={width ?? 1468}
        height={getTimelineGuideHeight(cpus, collapsedIndices)}
        margin={1}
      />
      {cpus.map((cpu, i) => {
        const isCollapsed = collapsedIndices.includes(i);
        const labelStr = labelSetToString(cpu);
        return (
          <div className="relative min-h-5" style={{width: width ?? 1468}} key={labelStr}>
            <div
              className="text-xs absolute top-0 left-0 flex gap-[2px] items-center bg-white/50 px-1 rounded-sm cursor-pointer"
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
                height={56}
                width={width ?? 1468}
                fill={color(labelStr) as string}
                selectionBounds={
                  isEqual(cpu, selectedTimeframe?.labels) ? selectedTimeframe?.bounds : undefined
                }
                setSelectionBounds={bounds => {
                  onSelectedTimeframe(cpu, bounds);
                }}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
};
