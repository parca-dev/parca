import {min} from 'd3-array';

// For a given link, return the target node's depth
function targetDepth(d) {
  return d.target.depth;
}

// The depth of a node when the nodeAlign (align) is set to 'left'
export function sankeyLeft(node) {
  return node.depth;
}

// The depth of a node when the nodeAlign (align) is set to 'right'
export function sankeyRight(node, n) {
  return n - 1 - node.height;
}

// The depth of a node when the nodeAlign (align) is set to 'justify'
export function sankeyJustify(node, n) {
  return node.sourceLinks.length ? node.depth : n - 1;
}

// The depth of a node when the nodeAlign (align) is set to 'center'
export function sankeyCenter(node) {
  return node.targetLinks.length
    ? node.depth
    : node.sourceLinks.length
    ? min(node.sourceLinks, targetDepth) - 1
    : 0;
}
