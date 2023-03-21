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

import {useEffect, useMemo} from 'react';

import {Flamegraph} from '@parca/client';
import {Button} from '@parca/components';
import {useContainerDimensions} from '@parca/dynamicsize';
import {selectQueryParam, type NavigateFunction} from '@parca/functions';

import DiffLegend from '../components/DiffLegend';
import {IcicleGraph} from './IcicleGraph';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width?: number;
  graph: Flamegraph | undefined;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  onContainerResize?: ResizeHandler;
  navigateTo?: NavigateFunction;
  loading: boolean;
  setActionButtons?: (buttons: JSX.Element) => void;
}

const ProfileIcicleGraph = ({
  graph,
  total,
  filtered,
  curPath,
  setNewCurPath,
  sampleUnit,
  onContainerResize,
  navigateTo,
  loading,
  setActionButtons,
}: ProfileIcicleGraphProps): JSX.Element => {
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';
  const {ref, dimensions} = useContainerDimensions();

  useEffect(() => {
    if (dimensions === undefined) return;
    if (onContainerResize === undefined) return;

    onContainerResize(dimensions.width, dimensions.height);
  }, [dimensions, onContainerResize]);

  const [
    totalFormatted,
    rawFormatted,
    isTrimmed,
    trimmedFormatted,
    trimmedPercentage,
    isFiltered,
    filteredPercentage,
  ] = useMemo(() => {
    if (graph === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    const trimmed = BigInt(graph.trimmed);

    const rawTotal = total + filtered + trimmed;

    // safeguard against division by zero
    const rawTotalDivisor = rawTotal > 0 ? rawTotal : BigInt(1);

    return [
      numberFormatter.format(total),
      numberFormatter.format(rawTotal),
      trimmed > 0,
      numberFormatter.format(trimmed),
      numberFormatter.format((trimmed * BigInt(100)) / rawTotalDivisor),
      filtered > 0,
      numberFormatter.format((total * BigInt(100)) / rawTotalDivisor),
    ];
  }, [filtered, graph, total]);

  useEffect(() => {
    if (setActionButtons === undefined) {
      return;
    }
    setActionButtons(
      <>
        <Button
          color="neutral"
          onClick={() => setNewCurPath([])}
          disabled={curPath.length === 0}
          className="w-auto !text-gray-800 dark:!text-gray-200"
          variant="neutral"
        >
          Reset View
        </Button>
      </>
    );
  }, [setNewCurPath, curPath, setActionButtons]);

  if (graph === undefined) return <div>no data...</div>;

  if (total === BigInt(0) && !loading) return <>Profile has no samples</>;

 if (isTrimmed) {
   console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`)
 }

  return (
    <div className="relative">
      {compareMode && <DiffLegend />}
      <div ref={ref}>
        <IcicleGraph
          width={dimensions?.width}
          graph={graph}
          curPath={curPath}
          setCurPath={setNewCurPath}
          sampleUnit={sampleUnit}
          navigateTo={navigateTo}
        />
      </div>
      <p className="my-2 text-xs">
        Showing {totalFormatted}{' '}
        {isFiltered ? (
          <span>
            ({filteredPercentage}%) filtered of {rawFormatted}{' '}
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
