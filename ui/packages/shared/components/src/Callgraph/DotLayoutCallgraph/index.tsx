// 1. Get callgraph data from the backend as nodes+links (https://github.com/parca-dev/parca/issues/1470)
// 2. Translate that on the frontend to a 'dot' graph string (example ‘dot’ graph: https://graphviz.org/Gallery/directed/pprof.gv.txt) https://github.com/Risto-Stevcev/json-to-dot/blob/master/index.js
// 3. Use Graphviz-WASM (which requires a 'dot' string to calculate layout) to translate the 'dot' graph to a 'JSON' graph (https://github.com/fabiospampinato/graphviz-wasm#readme)
// 4. Render that JSON graph (which has info on node positions and edge control points) in a Canvas container
// 5. Add interaction / tooltip with information

const DotLayoutCallgraph = ({data, height, width}) => {
  return <div />;
};

export default DotLayoutCallgraph;
