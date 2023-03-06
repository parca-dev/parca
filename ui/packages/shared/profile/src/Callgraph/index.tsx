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

import {memo, useEffect, useRef, useState} from 'react';

import * as d3 from 'd3';
import SVG from 'react-inlinesvg';
import {MapInteractionCSS} from 'react-map-interaction';

import {CallgraphEdge, Callgraph as CallgraphType} from '@parca/client';
import {getNewSpanColor} from '@parca/functions';
import {selectDarkMode, useAppSelector} from '@parca/store';

import {GraphTooltipContent as TooltipContent, type HoveringNode} from '../GraphTooltip';

export interface Props {
  data: CallgraphType;
  svgString: string;
  sampleUnit: string;
  width: number;
}

const Callgraph = ({data, svgString, sampleUnit, width}: Props): JSX.Element => {
  const svgRef = useRef(null);
  const svgWrapper = useRef(null);
  const [svgWrapperLoaded, setSvgWrapperLoaded] = useState(false);
  const [hoveredNode, setHoveredNode] = useState<any | null>(null);
  // const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  // const isSearchEmpty = currentSearchString === undefined || currentSearchString === '';
  // const [rawDashboardItems] = useURLState({param: 'dashboard_items'});
  // const dashboardItems = rawDashboardItems as string[];
  // const isCurrentSearchMatch = isSearchEmpty
  //               ? true
  //               : isSearchMatch(currentSearchString, sourceNode.functionName) &&
  //                 isSearchMatch(currentSearchString, targetNode.functionName);

  const isDarkMode = useAppSelector(selectDarkMode);
  const maxColor: string = getNewSpanColor(isDarkMode);
  const minColor: string = d3.scaleLinear([isDarkMode ? 'black' : 'white', maxColor])(0.3);
  const colorRange: [string, string] = [minColor, maxColor];
  const cumulatives = data.edges.map((edge: CallgraphEdge) => parseInt(edge.cumulative));
  const cumulativesRange = d3.extent(cumulatives);
  const colorScale = d3
    .scaleSequentialLog(d3.interpolateBlues)
    .domain([Number(cumulativesRange[0]), Number(cumulativesRange[1])])
    .range(colorRange);

  useEffect(() => {
    setSvgWrapperLoaded(true);
  }, []);

  // const resetView = (): void => {
  //   set scale and translate to default values
  // };

  useEffect(() => {
    if (svgWrapperLoaded && svgRef.current) {
      const addInteraction = () => {
        const svg = d3.select(svgRef.current);
        const nodes = svg.selectAll('.node');

        nodes.each(function () {
          const nodeData = data.nodes.find(n => {
            // @ts-ignore
            return n.id === this.id;
          });

          const defaultColor = colorScale(Number(nodeData?.cumulative));
          // const hexColor = d3.color(rgbColor)?.formatHex() ?? 'red';

          const node = d3.select(this);
          const path = node.select('path');

          node
            .style('cursor', 'pointer')
            .on('mouseenter', function (e) {
              d3.select(this).select('path').style('fill', 'white');
              setHoveredNode({...nodeData, mouseX: e.clientX, mouseY: e.clientY});
            })
            .on('mouseleave', function (e) {
              d3.select(this).select('path').style('fill', defaultColor);
              setHoveredNode(null);
            });
          path.style('fill', defaultColor);
        });
      };

      setTimeout(addInteraction, 1000);
    }
  }, [svgWrapper.current, svgWrapperLoaded]);

  if (data.nodes.length < 1) return <>Profile has no samples</>;

  return (
    <div className="w-full overflow-hidden relative">
      <MapInteractionCSS showControls minScale={1} maxScale={5}>
        <SVG
          ref={svgWrapper}
          src={svgString}
          width={width}
          height="auto"
          title="Callgraph"
          innerRef={svgRef}
        />
      </MapInteractionCSS>
      {hoveredNode && (
        // <div className={`absolute top-${hoveredNode.mouseY} left-${hoveredNode.mouseX}`}>
        <div className={`absolute top-0 left-0`}>
          <TooltipContent
            hoveringNode={hoveredNode as HoveringNode}
            unit={sampleUnit}
            total={parseInt(data.cumulative)}
            isFixed={false}
            strings={hoveredNode.meta.line}
            locations={hoveredNode.meta.location}
            functions={hoveredNode.meta.function}
            mappings={hoveredNode.meta.mapping}
          />
        </div>
      )}
      {/* {stage.scale.x !== 1 && (
          <div
            className={cx(
              dashboardItems.length > 1 ? 'left-[25px]' : 'left-0',
              'w-auto absolute top-[-46px]'
            )}
          >
            <Button variant="neutral" onClick={resetZoom}>
              Reset Zoom
            </Button>
          </div>
        )} */}
    </div>
  );
};

export default memo(Callgraph);
