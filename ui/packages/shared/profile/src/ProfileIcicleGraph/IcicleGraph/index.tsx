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
import {scaleLinear} from 'd3-scale';
import {pointer} from 'd3-selection';
import {throttle} from 'lodash';

import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {Button, useURLState} from '@parca/components';
import {selectQueryParam, type NavigateFunction} from '@parca/functions';
import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';

import GraphTooltip, {type HoveringNode} from '../../GraphTooltip';
import ColorStackLegend from './ColorStackLegend';
import {IcicleNode, RowHeight} from './IcicleGraphNodes';
import useColoredGraph from './useColoredGraph';

interface IcicleGraphProps {
  graph: Flamegraph;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  isTrimmed?: boolean;
}

export const IcicleGraph = memo(function IcicleGraph({
  graph,
  width,
  setCurPath,
  curPath,
  sampleUnit,
  navigateTo,
  isTrimmed = false,
}: IcicleGraphProps): JSX.Element {
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
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const [colorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );

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
  const isColorStackLegendVisible = colorProfileName !== 'default';

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)}>
      <ColorStackLegend navigateTo={navigateTo} compareMode={compareMode} />
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
        className={cx('flex justify-start absolute', {
          'top-[-48px]': dashboardItems.length <= 1 && !isTrimmed && !isColorStackLegendVisible,
          'top-[-69px]': dashboardItems.length <= 1 && !isTrimmed,
          'top-[-54px]': dashboardItems.length <= 1 && isTrimmed && isColorStackLegendVisible,
          'top-[-54px] ': dashboardItems.length <= 1 && isTrimmed && !isColorStackLegendVisible,
          'top-[-54px] left-[25px]':
            dashboardItems.length > 1 && isTrimmed && isColorStackLegendVisible,
          'top-[-54px] left-[25px] ':
            dashboardItems.length > 1 && isTrimmed && !isColorStackLegendVisible,
          'top-[-70px] left-[25px]':
            dashboardItems.length > 1 && !isTrimmed && isColorStackLegendVisible,
          'top-[-46px] left-[25px]':
            dashboardItems.length > 1 && !isTrimmed && !isColorStackLegendVisible,
        })}
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
              compareMode={compareMode}
            />
          </g>
        </g>
      </svg>
    </div>
  );
});

export default IcicleGraph;
