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

import React, {memo, useEffect, useMemo, useRef, useState} from 'react';

import cx from 'classnames';
import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';

import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import type {HoveringNode} from '../../GraphTooltip';
import GraphTooltip from '../../GraphTooltip';
import {Button, useURLState} from '@parca/components';
import {IcicleNode, RowHeight} from './IcicleGraphNodes';
import useColoredGraph from './useColoredGraph';
import {NavigateFunction, selectQueryParam} from '@parca/functions';
import ColorStackLegend from './ColorStackLegend';

interface IcicleGraphProps {
  graph: Flamegraph;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
}

export const IcicleGraph = memo(
  ({graph, width, setCurPath, curPath, sampleUnit, navigateTo}: IcicleGraphProps): JSX.Element => {
    const [hoveringNode, setHoveringNode] = useState<
      FlamegraphNode | FlamegraphRootNode | undefined
    >();
    const [pos, setPos] = useState([0, 0]);
    const [height, setHeight] = useState(0);
    const svg = useRef(null);
    const ref = useRef<SVGGElement>(null);
    const [rawDashboardItems] = useURLState({
      param: 'dashboard_items',
    });

    const dashboardItems = rawDashboardItems as string[];
    const coloredGraph = useColoredGraph(graph);
    const currentSearchString = (selectQueryParam('search_string') as string) ?? '';

    useEffect(() => {
      if (ref.current != null) {
        setHeight(ref?.current.getBoundingClientRect().height);
      }
    }, [width, coloredGraph]);

    const total = useMemo(() => parseFloat(coloredGraph.total), [coloredGraph.total]);
    const xScale = useMemo(() => {
      if (width === undefined) {
        return () => 0;
      }
      return scaleLinear().domain([0, total]).range([0, width]);
    }, [total, width]);

    if (coloredGraph.root === undefined || width === undefined) {
      return <></>;
    }

    const throttledSetPos = throttle(setPos, 20);
    const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement>): void => {
      // X/Y coordinate array relative to svg
      const rel = pointer(e);

      throttledSetPos([rel[0], rel[1]]);
    };

    return (
      <div onMouseLeave={() => setHoveringNode(undefined)} className="relative">
        <ColorStackLegend navigateTo={navigateTo} />
        <GraphTooltip
          unit={sampleUnit}
          total={total}
          x={pos[0]}
          y={pos[1]}
          hoveringNode={hoveringNode as HoveringNode}
          contextElement={svg.current}
          strings={coloredGraph.stringTable}
          mappings={coloredGraph.mapping}
          locations={coloredGraph.locations}
          functions={coloredGraph.function}
        />
        <div
          className={cx(
            dashboardItems.length > 1 ? 'top-[-46px] left-[25px]' : 'top-[-45px]',
            'flex justify-start absolute '
          )}
        >
          <Button
            color="neutral"
            onClick={() => setCurPath([])}
            disabled={curPath.length === 0}
            className="w-auto"
            variant="neutral"
          >
            Reset View
          </Button>
        </div>
        <svg
          className="font-robotoMono"
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
              />
            </g>
          </g>
        </svg>
      </div>
    );
  }
);

export default IcicleGraph;
