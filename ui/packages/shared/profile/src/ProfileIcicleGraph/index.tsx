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
    filteredFormatted,
    filteredPercentage,
  ] = useMemo(() => {
    if (graph === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    const total = BigInt(graph.total);
    const totalFormatted = numberFormatter.format(total);

    if (graph.untrimmedTotal === '0' && graph.unfilteredTotal === '0') {
      return [totalFormatted, '', false, '0', '0', false, '0', '0'];
    }

    const unfilteredTotal = BigInt(graph.unfilteredTotal);
    const untrimmedTotal = BigInt(graph.untrimmedTotal);

    let raw = total;
    if (untrimmedTotal > raw) {
      raw = untrimmedTotal;
    }
    if (unfilteredTotal > raw) {
      raw = unfilteredTotal;
    }

    const trimmed = untrimmedTotal - total;
    let trimmedPercentage = BigInt(0);
    if (trimmed > 0) {
      trimmedPercentage = (trimmed * BigInt(100)) / untrimmedTotal;
    }

    const trimmedOrTotal = trimmed > 0 ? trimmed : total;
    const filtered = unfilteredTotal - trimmedOrTotal;
    let filteredPercentage = BigInt(0);
    if (filtered > 0) {
      filteredPercentage = (filtered * BigInt(100)) / unfilteredTotal;
    }

    return [
      totalFormatted,
      numberFormatter.format(raw),
      trimmed > 0,
      numberFormatter.format(trimmed),
      trimmedPercentage.toString(),
      filtered > 0,
      numberFormatter.format(filtered),
      filteredPercentage.toString(),
    ];
  }, [graph]);

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

  const total = graph.total;

  if (parseFloat(total) === 0 && !loading) return <>Profile has no samples</>;

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
        Showing {totalFormatted} {isFiltered || isTrimmed ? <span>of {rawFormatted} </span> : <></>}
        samples.{' '}
        {isFiltered ? (
          <span>
            Filtered {filteredFormatted} ({filteredPercentage}%) samples.&nbsp;
          </span>
        ) : (
          <></>
        )}
        {isTrimmed ? (
          <span>
            Trimmed {trimmedFormatted} ({trimmedPercentage}%) too small samples.
          </span>
        ) : (
          <></>
        )}
      </p>
    </div>
  );
};

export default ProfileIcicleGraph;
