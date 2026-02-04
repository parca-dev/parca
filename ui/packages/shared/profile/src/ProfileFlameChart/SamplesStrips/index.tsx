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
import cx from 'classnames';
import * as d3 from 'd3';
import isEqual from 'fast-deep-equal';
import {useIntersectionObserver} from 'usehooks-ts';

import {LabelSet} from '@parca/client';

import {TimelineGuide} from '../../TimelineGuide';
import {NumberDuo} from '../../utils';
import {DataPoint, SamplesGraph} from './SamplesGraph';

export type {DataPoint} from './SamplesGraph';

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
  stepMs: number;
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
    } else {
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

const stickyPx = 0;

const SamplesGraphContainer = ({
  isSelected,
  isCollapsed,
  cpu,
  width,
  onToggleCollapse,
  data,
  selectionBounds,
  setSelectionBounds,
  color,
  stepMs,
}: {
  isSelected: boolean;
  isCollapsed: boolean;
  cpu: LabelSet;
  width: number | undefined;
  onToggleCollapse: () => void;
  data: DataPoint[];
  selectionBounds: NumberDuo | undefined;
  setSelectionBounds: (bounds: NumberDuo | undefined) => void;
  color: (label: string) => string;
  stepMs: number;
}): JSX.Element => {
  const labelStr = labelSetToString(cpu);

  const {isIntersecting, ref} = useIntersectionObserver({
    rootMargin: `${stickyPx}px 0px 0px 0px`,
  });

  const isSticky = useMemo(() => {
    return isSelected && isIntersecting;
  }, [isSelected, isIntersecting]);

  return (
    <div
      className={cx('min-h-5', {
        relative: !isSelected,
        'sticky z-30 bg-white dark:bg-black bg-opacity-75': isSelected,
        '!bg-opacity-100': isSticky,
      })}
      style={{width: width ?? 1468, top: isSelected ? stickyPx : undefined}}
      key={labelStr}
      ref={ref}
    >
      <div
        className="text-xs absolute top-0 left-0 flex gap-[2px] items-center bg-white/50 dark:bg-black/50 px-1 rounded-sm cursor-pointer"
        style={{
          zIndex: 15,
        }}
        onClick={onToggleCollapse}
      >
        <Icon icon={isCollapsed ? 'bxs:right-arrow' : 'bxs:down-arrow'} />
        {labelStr}
      </div>
      {!isCollapsed ? (
        <SamplesGraph
          data={data}
          height={STRIP_HEIGHT}
          width={width ?? 1468}
          fill={color(labelStr)}
          selectionBounds={selectionBounds}
          setSelectionBounds={setSelectionBounds}
          stepMs={stepMs}
        />
      ) : null}
    </div>
  );
};

export const SamplesStrip = ({
  cpus,
  data,
  selectedTimeframe,
  onSelectedTimeframe,
  width,
  bounds,
  stepMs,
}: Props): JSX.Element => {
  const [collapsedIndices, setCollapsedIndices] = useState<number[]>([]);

  // Deterministic color: hash the label string so the same label always gets the same color
  // regardless of render order.
  const color = useMemo(() => {
    const palette = d3.schemeObservable10;
    const hashStr = (s: string): number => {
      let h = 0;
      for (let i = 0; i < s.length; i++) {
        h = (Math.imul(31, h) + s.charCodeAt(i)) | 0;
      }
      return Math.abs(h);
    };
    return (label: string): string => palette[hashStr(label) % palette.length];
  }, []);

  if (data.length === 0) {
    return (
      <span className="flex justify-center my-10">
        There is no data matching your filter criteria, please try changing the filter.
      </span>
    );
  }

  return (
    <div className="flex flex-col gap-1 relative my-0" style={{width: width ?? '100%'}}>
      <TimelineGuide
        bounds={[BigInt(0), BigInt(bounds[1] - bounds[0])]}
        width={width ?? 1468}
        height={getTimelineGuideHeight(cpus, collapsedIndices)}
        margin={1}
      />
      {cpus.map((cpu, i) => {
        const isCollapsed = collapsedIndices.includes(i);
        const isSelected = isEqual(cpu, selectedTimeframe?.labels);

        return (
          <SamplesGraphContainer
            isSelected={isSelected}
            isCollapsed={isCollapsed}
            cpu={cpu}
            width={width}
            data={data[i]}
            onToggleCollapse={() => {
              const newCollapsedIndices = [...collapsedIndices];
              if (collapsedIndices.includes(i)) {
                newCollapsedIndices.splice(newCollapsedIndices.indexOf(i), 1);
              } else {
                newCollapsedIndices.push(i);
              }
              setCollapsedIndices(newCollapsedIndices);
            }}
            selectionBounds={isSelected ? selectedTimeframe?.bounds : undefined}
            setSelectionBounds={bounds => {
              onSelectedTimeframe(cpu, bounds);
            }}
            color={color}
            stepMs={stepMs}
            key={labelSetToString(cpu)}
          />
        );
      })}
    </div>
  );
};
