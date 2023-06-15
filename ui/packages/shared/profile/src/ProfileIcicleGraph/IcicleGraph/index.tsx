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

import {memo, useEffect, useMemo, useRef, useState} from 'react';

import {Flamegraph} from '@parca/client';
import {setHoveringNode, useAppDispatch} from '@parca/store';
import {scaleLinear, selectQueryParam, type NavigateFunction} from '@parca/utilities';

import GraphTooltip from '../../GraphTooltip';
import ColorStackLegend from './ColorStackLegend';
import {IcicleNode, RowHeight} from './IcicleGraphNodes';
import useColoredGraph from './useColoredGraph';

interface IcicleGraphProps {
  graph: Flamegraph;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
}

export const IcicleGraph = memo(function IcicleGraph({
  graph,
  total,
  filtered,
  width,
  setCurPath,
  curPath,
  sampleUnit,
  navigateTo,
}: IcicleGraphProps): JSX.Element {
  const dispatch = useAppDispatch();
  const [height, setHeight] = useState(0);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  const coloredGraph = useColoredGraph(graph);
  const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width, coloredGraph]);

  const xScale = useMemo(() => {
    if (width === undefined) {
      return () => 0;
    }
    return scaleLinear([0n, total], [0, width]);
  }, [total, width]);

  if (coloredGraph.root === undefined || width === undefined) {
    return <></>;
  }

  return (
    <div onMouseLeave={() => dispatch(setHoveringNode(undefined))}>
      <ColorStackLegend navigateTo={navigateTo} compareMode={compareMode} />
      <GraphTooltip
        unit={sampleUnit}
        total={total}
        totalUnfiltered={total + filtered}
        contextElement={svg.current}
        strings={coloredGraph.stringTable}
        mappings={coloredGraph.mapping}
        locations={coloredGraph.locations}
        functions={coloredGraph.function}
      />
      <svg
        className="font-robotoMono"
        width={width}
        height={height}
        preserveAspectRatio="xMinYMid"
        ref={svg}
      >
        <g ref={ref}>
          <g transform={'translate(0, 0)'}>
            <IcicleNode
              x={0}
              y={0}
              totalWidth={width}
              height={RowHeight}
              setCurPath={setCurPath}
              curPath={curPath}
              data={coloredGraph.root}
              strings={coloredGraph.stringTable}
              mappings={coloredGraph.mapping}
              locations={coloredGraph.locations}
              functions={coloredGraph.function}
              total={total}
              xScale={xScale}
              path={[]}
              level={0}
              isRoot={true}
              searchString={currentSearchString}
              compareMode={compareMode}
            />
          </g>
        </g>
      </svg>
    </div>
  );
});

export default IcicleGraph;
