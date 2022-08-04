import React from 'react';
import {Core as CSCore} from 'cytoscape';
import CytoscapeComponent from 'react-cytoscapejs';
import stylesheet from './stylesheet';
import {CallgraphData} from '../types';

interface Props {
  data: CallgraphData;
  width?: number;
  height?: number;
}

const layout = {
  name: 'breadthfirst',
  fit: true, // whether to fit the viewport to the graph
  directed: false, // whether the tree is directed downwards (or edges can point in any direction if false)
  padding: 50, // padding on fit
  circle: false, // put depths in concentric circles if true, put depths top down if false
  grid: false, // whether to create an even grid into which the DAG is placed (circle:false only)
  spacingFactor: 1.75, // positive spacing factor, larger => more space between nodes (N.B. n/a if causes overlap)
  boundingBox: undefined, // constrain layout bounds; { x1, y1, x2, y2 } or { x1, y1, w, h }
  avoidOverlap: true, // prevents node overlap, may overflow boundingBox if not enough space
  nodeDimensionsIncludeLabels: false, // Excludes the label when calculating node bounding boxes for the layout algorithm
  roots: '#root', // the roots of the trees
  maximal: false, // whether to shift nodes down their natural BFS depths in order to avoid upwards edges (DAGS only)
  depthSort: undefined, // a sorting function to order nodes at equal depth. e.g. function(a, b){ return a.data('weight') - b.data('weight') }
  animate: false, // whether to transition the node positions
  animationDuration: 500, // duration of animation in ms if enabled
  animationEasing: undefined, // easing of animation if enabled,
  // animateFilter: function ( node, i ){ return true; }, // a function that determines whether the node should be animated.  All nodes animated by default on animate enabled.  Non-animated nodes are positioned immediately when the layout starts
  ready: undefined, // callback on layoutready
  stop: undefined, // callback on layoutstop
  // transform: function (node, position ){ return position; } // transform a given node position. Useful for changing flow direction in discrete layouts
};

export default React.memo(({data, width, height}: Props) => {
  const {nodes, edges} = data;
  // Cytoscape instance
  const cyRef = React.useRef<CSCore | null>(null);

  const init = React.useCallback((cy: CSCore) => {
    if (!cyRef.current) {
      cyRef.current = cy;
    }
  }, []);

  return (
    <CytoscapeComponent
      elements={CytoscapeComponent.normalizeElements({
        nodes,
        edges,
      })}
      layout={layout}
      // @ts-ignore
      cy={cy => init(cy)}
      // @ts-ignore
      stylesheet={stylesheet}
      style={{
        width: width ?? '90vw',
        height: height ?? '100vh',
      }}
      panningEnabled={true}
      userPanningEnabled={false}
      zoomingEnabled={true}
      userZoomingEnabled={false}
      // autolock={true}
    />
  );
});
