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

import React, {useEffect, useMemo, useRef, useState} from 'react';

import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';

import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import type {HoveringNode} from '../../GraphTooltip';
import GraphTooltip from '../../GraphTooltip';
import {FeatureColor} from '@parca/functions';
import {Button} from '@parca/components';
import {featureColors, IcicleNode, RowHeight} from './IcicleGraphNodes';

interface IcicleGraphProps {
  graph: Flamegraph;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
}

export default function IcicleGraph({
  graph,
  width,
  setCurPath,
  curPath,
  sampleUnit,
}: IcicleGraphProps): JSX.Element {
  const [hoveringNode, setHoveringNode] = useState<
    FlamegraphNode | FlamegraphRootNode | undefined
  >();
  const [pos, setPos] = useState([0, 0]);
  const [height, setHeight] = useState(0);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);
  const [featureColorsState, setFeatureColorsState] =
    useState<Record<string, FeatureColor>>(featureColors);

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width, graph]);

  useEffect(() => {
    if (
      Object.values(
        Object.values(featureColors).reduce((acc, val) => {
          acc[val.color] = true;
          return acc;
        }, {})
      ).length < 2
    ) {
      if (Object.values(featureColorsState).length > 0) {
        setFeatureColorsState({});
      }
      return;
    }

    if (Object.values(featureColorsState).length !== Object.values(featureColors).length) {
      console.log('setting featurColors', featureColors);
      setFeatureColorsState(featureColors);
    }
  });

  const total = useMemo(() => parseFloat(graph.total), [graph.total]);
  const xScale = useMemo(() => {
    if (width === undefined) {
      return () => 0;
    }
    return scaleLinear().domain([0, total]).range([0, width]);
  }, [total, width]);

  if (graph.root === undefined || width === undefined) {
    return <></>;
  }

  const throttledSetPos = throttle(setPos, 20);
  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    throttledSetPos([rel[0], rel[1]]);
  };

  console.log('featureColorsState', featureColorsState);

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)}>
      <div className="flex flex-wrap gap-4 px-10 my-6">
        {Object.values(featureColorsState)
          .sort((a, b) => {
            if (a.feature === 'Everything else') {
              return 1;
            }
            if (b.feature === 'Everything else') {
              return -1;
            }
            return a.feature?.localeCompare(b.feature ?? '') ?? 0;
          })
          .map(({feature, color}) => {
            return (
              <div key={feature} className="flex gap-1 items-center">
                <div className="w-4 h-4 mr-1 inline-block" style={{backgroundColor: color}} />
                <span className="text-sm">{feature}</span>
              </div>
            );
          })}
      </div>
      <GraphTooltip
        unit={sampleUnit}
        total={total}
        x={pos[0]}
        y={pos[1]}
        hoveringNode={hoveringNode as HoveringNode}
        contextElement={svg.current}
        strings={graph.stringTable}
        mappings={graph.mapping}
        locations={graph.locations}
        functions={graph.function}
      />
      <div className="w-full flex justify-start">
        <Button
          color="neutral"
          onClick={() => setCurPath([])}
          disabled={curPath.length === 0}
          className="w-auto"
          variant="neutral"
        >
          Reset zoom
        </Button>
      </div>
      <svg
        className="font-robotoMono "
        width={width}
        height={height}
        onMouseMove={onMouseMove}
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
              setHoveringNode={setHoveringNode}
              curPath={curPath}
              data={graph.root}
              strings={graph.stringTable}
              mappings={graph.mapping}
              locations={graph.locations}
              functions={graph.function}
              total={total}
              xScale={xScale}
              path={[]}
              level={0}
              isRoot={true}
            />
          </g>
        </g>
      </svg>
    </div>
  );
}
