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

import React, {useEffect, useMemo, useState} from 'react';

import {Table} from 'apache-arrow';

import {Flamegraph} from '@parca/client';
import {Button, Select} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {divide, selectQueryParam, type NavigateFunction} from '@parca/utilities';

import DiffLegend from '../components/DiffLegend';
import IcicleGraph from './IcicleGraph';
import IcicleGraphArrow, {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_NAME,
} from './IcicleGraphArrow';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width?: number;
  graph?: Flamegraph;
  table?: Table<any>;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
}

const ProfileIcicleGraph = ({
  graph,
  table,
  total,
  filtered,
  curPath,
  setNewCurPath,
  sampleUnit,
  navigateTo,
  loading,
  setActionButtons,
}: ProfileIcicleGraphProps): JSX.Element => {
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';
  const {ref, dimensions} = useContainerDimensions();
  const [sortBy, setSortBy] = useState<string>(FIELD_FUNCTION_NAME);

  const [
    totalFormatted,
    totalUnfilteredFormatted,
    isTrimmed,
    trimmedFormatted,
    trimmedPercentage,
    isFiltered,
    filteredPercentage,
  ] = useMemo(() => {
    if (graph === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    // const trimmed = graph.trimmed;
    const trimmed = 0n;

    const totalUnfiltered = total + filtered;
    // safeguard against division by zero
    const totalUnfilteredDivisor = totalUnfiltered > 0 ? totalUnfiltered : 1n;

    return [
      numberFormatter.format(total),
      numberFormatter.format(totalUnfiltered),
      trimmed > 0,
      numberFormatter.format(trimmed),
      numberFormatter.format(divide(trimmed * 100n, totalUnfilteredDivisor)),
      filtered > 0,
      numberFormatter.format(divide(total * 100n, totalUnfilteredDivisor)),
    ];
  }, [graph, filtered, total]);

  useEffect(() => {
    if (setActionButtons === undefined) {
      return;
    }
    setActionButtons(
      <div className="flex w-full justify-end gap-2 pb-2">
        <div className="flex w-full items-center justify-between space-x-2">
          {table !== undefined && (
            <div>
              <label className="text-sm">SortBy</label>
              <Select
                items={[
                  {
                    key: FIELD_FUNCTION_NAME,
                    disabled: false,
                    element: {
                      active: <>Function</>,
                      expanded: (
                        <>
                          <span>Function</span>
                        </>
                      ),
                    },
                  },
                  {
                    key: FIELD_CUMULATIVE,
                    disabled: false,
                    element: {
                      active: <>Cumulative</>,
                      expanded: (
                        <>
                          <span>Cumulative</span>
                        </>
                      ),
                    },
                  },
                  {
                    key: FIELD_DIFF,
                    disabled: !compareMode,
                    element: {
                      active: <>Diff</>,
                      expanded: (
                        <>
                          <span>Diff</span>
                        </>
                      ),
                    },
                  },
                ]}
                selectedKey={sortBy}
                onSelection={key => setSortBy(key)}
                placeholder={'Sort By'}
                primary={false}
                disabled={false}
              />
            </div>
          )}
          <div>
            <label>&nbsp;</label>
            <Button
              color="neutral"
              onClick={() => setNewCurPath([])}
              disabled={curPath.length === 0}
              variant="neutral"
            >
              Reset View
            </Button>
          </div>
        </div>
      </div>
    );
  }, [setNewCurPath, curPath, setActionButtons, sortBy, table, compareMode]);

  if (graph === undefined && table === undefined) return <div>no data...</div>;

  if (total === 0n && !loading) return <>Profile has no samples</>;

  if (isTrimmed) {
    console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`);
  }

  return (
    <div className="relative">
      {compareMode && <DiffLegend />}
      <div ref={ref} className="min-h-48">
        {graph !== undefined && (
          <IcicleGraph
            width={dimensions?.width}
            graph={graph}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
          />
        )}
        {table !== undefined && (
          <IcicleGraphArrow
            width={dimensions?.width}
            table={table}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
            sortBy={sortBy}
          />
        )}
      </div>
      <p className="my-2 text-xs">
        Showing {totalFormatted}{' '}
        {isFiltered ? (
          <span>
            ({filteredPercentage}%) filtered of {totalUnfilteredFormatted}{' '}
          </span>
        ) : (
          <></>
        )}
        values.{' '}
      </p>
    </div>
  );
};

export default ProfileIcicleGraph;
