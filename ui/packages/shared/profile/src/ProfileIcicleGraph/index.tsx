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

import {Flamegraph} from '@parca/client';
import {useContainerDimensions} from '@parca/dynamicsize';

import DiffLegend from '../components/DiffLegend';
import IcicleGraph from '../IcicleGraph';
import {selectQueryParam} from '@parca/functions';
import {useEffect, useMemo} from 'react';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width?: number;
  graph: Flamegraph | undefined;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  onContainerResize?: ResizeHandler;
  loading: boolean;
}

const ProfileIcicleGraph = ({
  graph,
  curPath,
  setNewCurPath,
  sampleUnit,
  onContainerResize,
  loading,
}: ProfileIcicleGraphProps): JSX.Element => {
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';
  const {ref, dimensions} = useContainerDimensions();

  useEffect(() => {
    if (dimensions === undefined) return;
    if (onContainerResize === undefined) return;

    onContainerResize(dimensions.width, dimensions.height);
  }, [dimensions, onContainerResize]);

  const [trimDifference, trimmedPercentage, formattedTotal, formattedUntrimmedTotal] =
    useMemo(() => {
      if (graph === undefined || graph.untrimmedTotal === '0') {
        return [BigInt(0), '0'];
      }

      const untrimmedTotal = BigInt(graph.untrimmedTotal);
      const total = BigInt(graph.total);

      const trimDifference = untrimmedTotal - total;
      const trimmedPercentage = (total * BigInt(100)) / untrimmedTotal;

      return [
        trimDifference,
        trimmedPercentage.toString(),
        numberFormatter.format(total),
        numberFormatter.format(untrimmedTotal),
      ];
    }, [graph]);

  if (graph === undefined) return <div>no data...</div>;

  const total = graph.total;

  if (parseFloat(total) === 0 && !loading) return <>Profile has no samples</>;

  return (
    <>
      {compareMode && <DiffLegend />}
      {trimDifference > BigInt(0) ? (
        <p className="my-2 text-sm">
          Showing {formattedTotal}({trimmedPercentage}%) out of {formattedUntrimmedTotal} samples
        </p>
      ) : null}
      <div ref={ref}>
        <IcicleGraph
          width={dimensions?.width}
          graph={graph}
          curPath={curPath}
          setCurPath={setNewCurPath}
          sampleUnit={sampleUnit}
        />
      </div>
    </>
  );
};

export default ProfileIcicleGraph;
