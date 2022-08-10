import React, {useEffect, useRef, useState} from 'react';
import CytoscapeCallgraph from './CytoscapeCallgraph';
import D3DagCallgraph from './D3DagCallgraph';
import DotLayoutCallgraph from './DotLayoutCallgraph';
import mockData from './CytoscapeCallgraph/mockData';
import {useContainerDimensions} from '@parca/dynamicsize';
import {CallgraphData} from './types';
import {jsonGraphWithMetaData} from './DotLayoutCallgraph/mockData';

interface Props {
  graph: CallgraphData;
  sampleUnit: string;
  width?: number;
  height?: number;
}

const Callgraph = ({
  graph,
  sampleUnit,
  width: customWidth,
  height: customHeight,
}: Props): JSX.Element => {
  const {ref: containerRef, dimensions: originalDimensions} = useContainerDimensions();
  const fullWidth = customWidth ?? originalDimensions?.width;
  const fullHeight = customHeight ?? 600;

  return (
    <div ref={containerRef}>
      {/* <D3DagCallgraph graph={{graph: {data: mockData}}} width={fullWidth} height={fullHeight} /> */}
      {/* <CytoscapeCallgraph data={mockData} width={fullWidth} height={fullHeight} /> */}
      <DotLayoutCallgraph
        graph={graph}
        sampleUnit={sampleUnit}
        width={fullWidth}
        height={fullHeight}
      />
    </div>
  );
};

export default Callgraph;
