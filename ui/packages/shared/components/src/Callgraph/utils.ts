interface Pos {
  x: number;
  y: number;
}

export const parsePos = (pos: string): Pos => {
  const [x, y] = pos.split(',');
  return {
    x: +x,
    y: +y,
  };
};

export const getPathDataFromPos = (linkPos: string, startPos: string, endPos: string) => {
  const posParts = linkPos.split(' ');

  // remove arrowhead end point
  posParts.shift();

  // remove first point (as this will be replaced by the source node's position)
  posParts.shift();

  // remove last point (as this will be replaced by the target node's position)
  posParts.pop();

  return `M${startPos}C${posParts.join(' ')} ${endPos}`;
};

// all edges (not just curves) are defined as cubic B-splines and are defined by 1 + n*3 points
// (n is integer >=1) (https://graphviz.org/docs/attr-types/splineType/) (i.e. 4 or 7 or 11 or ...)
// first point is the start
// next two are control points
// last point is the end

// points with e or s are arrowhead points
// for directed graphs we will just have 'e' points
// we can just filter these out for the sake of drawing our curves

// so show have M${p1}C${p2}${p3}${p4}...${p5}${p6}${p7}...
