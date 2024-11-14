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

import {AreaGraph, DataPoint, NumberDuo} from './AreaGraph';
import {TimelineGuide} from './TimelineGuide';

interface Props {
  cpus: string[];
  data: DataPoint[][];
  selectedTimeline?: {
    index: number;
    bounds: NumberDuo;
  };
  onSelectedTimeline: (index: number, bounds: NumberDuo | undefined) => void;
}

const COLORS = [];

const getTimelineGuideHeight = (cpus: string[], collapsedIndices: number[]) => {
  return 56 * (cpus.length - collapsedIndices.length) + 20 * collapsedIndices.length + 24;
};

export const MetricsGraphStrips = ({cpus, data, selectedTimeline, onSelectedTimeline}: Props) => {
  const [collapsedIndices, setCollapsedIndices] = useState<number[]>([]);

  // @ts-expect-error
  const color = d3.scaleOrdinal(d3.schemeObservable10);

  return (
    <div className="flex flex-col gap-1 relative">
      <TimelineGuide
        data={data}
        width={1468}
        height={getTimelineGuideHeight(cpus, collapsedIndices)}
        margin={1}
      />
      {cpus.map((cpu, i) => {
        const isCollapsed = collapsedIndices.includes(i);
        return (
          <div className="relative min-h-5" key={cpu}>
            <div
              className="text-xs absolute top-0 left-0 flex gap-[2px] items-center bg-white/50 px-1 rounded-sm cursor-pointer z-30"
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
              {cpu}
            </div>
            {!isCollapsed ? (
              <AreaGraph
                data={data[i]}
                height={56}
                width={1468}
                fill={color(i.toString()) as string}
                selectionBounds={
                  selectedTimeline?.index === i ? selectedTimeline.bounds : undefined
                }
                setSelectionBounds={bounds => {
                  onSelectedTimeline(i, bounds);
                }}
              />
            ) : null}
          </div>
        );
      })}
    </div>
  );
};
