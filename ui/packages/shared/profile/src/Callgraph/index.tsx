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

import {useEffect, useRef, useState} from 'react';

import cx from 'classnames';
import * as d3 from 'd3';
import SVG from 'react-inlinesvg';
import {MapInteractionCSS} from 'react-map-interaction';

import {CallgraphEdge, Callgraph as CallgraphType} from '@parca/client';
import {Button, useKeyDown, useURLState} from '@parca/components';
import {selectDarkMode, setHoveringNode, useAppDispatch, useAppSelector} from '@parca/store';
import {getNewSpanColor} from '@parca/utilities';

import GraphTooltip from '../GraphTooltip';

export interface Props {
  data: CallgraphType;
  svgString: string;
  sampleUnit: string;
  width: number;
}

interface View {
  scale: number;
  translation: {x: number; y: number};
}

const Callgraph = ({data, svgString, sampleUnit, width}: Props): JSX.Element => {
  const originalView = {
    scale: 1,
    translation: {x: 0, y: 0},
  };
  const [view, setView] = useState<View>(originalView);
  const containerRef = useRef(null);
  const svgRef = useRef(null);
  const svgWrapper = useRef(null);
  const [svgWrapperLoaded, setSvgWrapperLoaded] = useState(false);
  const dispatch = useAppDispatch();
  const {isShiftDown} = useKeyDown();
  // TODO: implement highlighting nodes on user search
  // const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  // const isSearchEmpty = currentSearchString === undefined || currentSearchString === '';
  // const isCurrentSearchMatch = isSearchEmpty
  //   ? true
  //   : isSearchMatch(currentSearchString, sourceNode.functionName) &&
  //     isSearchMatch(currentSearchString, targetNode.functionName);
  const [rawDashboardItems] = useURLState({param: 'dashboard_items'});
  const dashboardItems =
    rawDashboardItems !== undefined ? (rawDashboardItems as string[]) : ['icicle'];

  const isDarkMode = useAppSelector(selectDarkMode);
  const maxColor: string = getNewSpanColor(isDarkMode);
  const minColor: string = d3.scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  const colorRange: [string, string] = [minColor, maxColor];
  const cumulatives = data.edges.map((edge: CallgraphEdge) => edge.cumulative.toString());
  const cumulativesRange = d3.extent(cumulatives);
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain([Number(cumulativesRange[0]), Number(cumulativesRange[1])])
    .range(colorRange);

  useEffect(() => {
    setSvgWrapperLoaded(true);
  }, []);

  useEffect(() => {
    if (svgWrapperLoaded && svgRef.current !== null) {
      const addInteraction = (): void => {
        const svg = d3.select(svgRef.current);
        const nodes = svg.selectAll('.node');

        nodes.each(function () {
          const nodeData = data.nodes.find((n): boolean => {
            return n.id === (this as Element).id;
          });
          const defaultColor = colorScale(Number(nodeData?.cumulative));
          const node = d3.select(this);
          const path = node.select('path');

          node
            .style('cursor', 'pointer')
            .on('mouseenter', function () {
              if (isShiftDown) return;
              d3.select(this).select('path').style('fill', 'white');
              const hoveringNode = {
                ...nodeData,
                meta: {...nodeData?.meta, lineIndex: 0, locationIndex: 0},
              };
              // @ts-expect-error
              dispatch(setHoveringNode(hoveringNode));
            })
            .on('mouseleave', function () {
              if (isShiftDown) return;
              d3.select(this).select('path').style('fill', defaultColor);
              dispatch(setHoveringNode(undefined));
            });
          path.style('fill', defaultColor);
        });
      };

      setTimeout(addInteraction, 1000);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [svgWrapper.current, svgWrapperLoaded]);

  if (data.nodes.length < 1) return <>Profile has no samples</>;

  const resetView = (): void => setView(originalView);

  const isResetViewButtonEnabled =
    view.scale !== originalView.scale ||
    view.translation.x !== originalView.translation.x ||
    view.translation.y !== originalView.translation.y;

  return (
    <div className="relative w-full">
      <div ref={containerRef} className="w-full overflow-hidden">
        <MapInteractionCSS
          showControls
          minScale={1}
          maxScale={5}
          value={view}
          onChange={(value: View) => setView(value)}
        >
          <SVG
            ref={svgWrapper}
            src={svgString}
            width={width}
            height="auto"
            title="Callgraph"
            innerRef={svgRef}
          />
        </MapInteractionCSS>
        {svgRef.current !== null && (
          <GraphTooltip
            type="callgraph"
            unit={sampleUnit}
            total={data.cumulative}
            totalUnfiltered={data.cumulative}
            contextElement={containerRef.current}
          />
        )}
      </div>
      <div
        className={cx(
          dashboardItems.length > 1 ? 'left-[25px]' : 'left-0',
          'absolute top-[-46px] w-auto'
        )}
      >
        <Button
          variant="neutral"
          onClick={resetView}
          className="z-50"
          disabled={!isResetViewButtonEnabled}
        >
          Reset View
        </Button>
      </div>
    </div>
  );
};

export default Callgraph;
