/* eslint-disable */
/// https://github.com/tomshanley/d3-sankeyCircular-circular
// fork of https://github.com/d3/d3-sankeyCircular copyright Mike Bostock
import {ascending, min, max, mean, sum} from 'd3-array';
import {map, nest} from 'd3-collection';
import {sankeyJustify} from './align';
import {linkHorizontal} from 'd3-shape';
import findCircuits from 'elementary-circuits-directed-graph';

// returns a function, using the parameter given to the sankey setting
function constant(x) {
  return function () {
    return x;
  };
}

// sort links' breadth (ie top to bottom in a column), based on their source nodes' breadths
function ascendingSourceBreadth(a, b) {
  return ascendingBreadth(a.source, b.source) || a.index - b.index;
}

// sort links' breadth (ie top to bottom in a column), based on their target nodes' breadths
function ascendingTargetBreadth(a, b) {
  return ascendingBreadth(a.target, b.target) || a.index - b.index;
}

// sort nodes' breadth (ie top to bottom in a column)
// if both nodes have circular links, or both don't have circular links, then sort by the top (y0) of the node
// else push nodes that have top circular links to the top, and nodes that have bottom circular links to the bottom
function ascendingBreadth(a, b) {
  if (a.partOfCycle === b.partOfCycle) {
    return a.y0 - b.y0;
  } else {
    if (a.circularLinkType === 'top' || b.circularLinkType === 'bottom') {
      return -1;
    } else {
      return 1;
    }
  }
}

// return the value of a node or link
function value(d) {
  return d.value;
}

// return the vertical center of a node
function nodeCenter(node) {
  return (node.y0 + node.y1) / 2;
}

// return the vertical center of a link's source node
function linkSourceCenter(link) {
  return nodeCenter(link.source);
}

// return the vertical center of a link's target node
function linkTargetCenter(link) {
  return nodeCenter(link.target);
}

// Return the default value for ID for node, d.index
function defaultId(d) {
  return d.index;
}

// Return the default object the graph's nodes, graph.nodes
function defaultNodes(graph) {
  return graph.nodes;
}

// Return the default object the graph's nodes, graph.links
function defaultLinks(graph) {
  return graph.links;
}

// Return the node from the collection that matches the provided ID, or throw an error if no match
function find(nodeById, id) {
  var node = nodeById.get(id);
  if (!node) throw new Error('missing: ' + id);
  return node;
}

function getNodeID(node, id) {
  return id(node);
}

// The main sankeyCircular functions

// Some constants for circular link calculations
var verticalMargin = 25;
var baseRadius = 10;
var scale = 0.3; //Possibly let user control this, although anything over 0.5 starts to get too cramped

export default function (NODE_WIDTH = 24) {
  // Set the default values
  var x0 = 0,
    y0 = 0,
    x1 = 1,
    y1 = 1, // extent
    dx = NODE_WIDTH, // nodeWidth
    py, // nodePadding, for vertical postioning
    id = defaultId,
    align = sankeyJustify,
    nodes = defaultNodes,
    links = defaultLinks,
    iterations = 32,
    circularLinkGap = 2,
    paddingRatio,
    sortNodes = null;

  function sankeyCircular() {
    var graph = {
      nodes: nodes.apply(null, arguments),
      links: links.apply(null, arguments),
    };

    // Process the graph's nodes and links, setting their positions

    // 1.  Associate the nodes with their respective links, and vice versa
    computeNodeLinks(graph);

    // 2.  Determine which links result in a circular path in the graph
    identifyCircles(graph, id, sortNodes);

    // 4. Calculate the nodes' values, based on the values of the incoming and outgoing links
    computeNodeValues(graph);

    // 5.  Calculate the nodes' depth based on the incoming and outgoing links
    //     Sets the nodes':
    //     - depth:  the depth in the graph
    //     - column: the depth (0, 1, 2, etc), as is relates to visual position from left to right
    //     - x0, x1: the x coordinates, as is relates to visual position from left to right
    computeNodeDepths(graph);

    // 3.  Determine how the circular links will be drawn,
    //     either travelling back above the main chart ("top")
    //     or below the main chart ("bottom")
    selectCircularLinkTypes(graph, id);

    // 6.  Calculate the nodes' and links' vertical position within their respective column
    //     Also readjusts sankeyCircular size if circular links are needed, and node x's
    computeNodeBreadths(graph, iterations, id);
    computeLinkBreadths(graph);

    // 7.  Sort links per node, based on the links' source/target nodes' breadths
    // 8.  Adjust nodes that overlap links that span 2+ columns
    var linkSortingIterations = 4; //Possibly let user control this number, like the iterations over node placement
    for (var iteration = 0; iteration < linkSortingIterations; iteration++) {
      sortSourceLinks(graph, y1, id);
      sortTargetLinks(graph, y1, id);
      resolveNodeLinkOverlaps(graph, y0, y1, id);
      sortSourceLinks(graph, y1, id);
      sortTargetLinks(graph, y1, id);
    }

    // 8.1  Fix nodes overlapping after sortNodes
    resolveNodesOverlap(graph, y0, py);

    // 8.2  Adjust node and link positions back to fill height of chart area if compressed
    fillHeight(graph, y0, y1);

    // 9. Calculate visually appealling path for the circular paths, and create the "d" string
    addCircularPathData(graph, circularLinkGap, y1, id);

    return graph;
  } // end of sankeyCircular function

  // Set the sankeyCircular parameters
  // nodeID, nodeAlign, nodeWidth, nodePadding, nodes, links, size, extent, iterations, nodePaddingRatio, circularLinkGap
  sankeyCircular.nodeId = function (_) {
    return arguments.length
      ? ((id = typeof _ === 'function' ? _ : constant(_)), sankeyCircular)
      : id;
  };

  sankeyCircular.nodeAlign = function (_) {
    return arguments.length
      ? ((align = typeof _ === 'function' ? _ : constant(_)), sankeyCircular)
      : align;
  };

  sankeyCircular.nodeWidth = function (_) {
    return arguments.length ? ((dx = +_), sankeyCircular) : dx;
  };

  sankeyCircular.nodePadding = function (_) {
    return arguments.length ? ((py = +_), sankeyCircular) : py;
  };

  sankeyCircular.nodes = function (_) {
    return arguments.length
      ? ((nodes = typeof _ === 'function' ? _ : constant(_)), sankeyCircular)
      : nodes;
  };

  sankeyCircular.links = function (_) {
    return arguments.length
      ? ((links = typeof _ === 'function' ? _ : constant(_)), sankeyCircular)
      : links;
  };

  sankeyCircular.size = function (_) {
    return arguments.length
      ? ((x0 = y0 = 0), (x1 = +_[0]), (y1 = +_[1]), sankeyCircular)
      : [x1 - x0, y1 - y0];
  };

  sankeyCircular.extent = function (_) {
    return arguments.length
      ? ((x0 = +_[0][0]), (x1 = +_[1][0]), (y0 = +_[0][1]), (y1 = +_[1][1]), sankeyCircular)
      : [
          [x0, y0],
          [x1, y1],
        ];
  };

  sankeyCircular.iterations = function (_) {
    return arguments.length ? ((iterations = +_), sankeyCircular) : iterations;
  };

  sankeyCircular.circularLinkGap = function (_) {
    return arguments.length ? ((circularLinkGap = +_), sankeyCircular) : circularLinkGap;
  };

  sankeyCircular.nodePaddingRatio = function (_) {
    return arguments.length ? ((paddingRatio = +_), sankeyCircular) : paddingRatio;
  };

  sankeyCircular.sortNodes = function (_) {
    return arguments.length ? ((sortNodes = _), sankeyCircular) : sortNodes;
  };

  sankeyCircular.update = function (graph) {
    // 5.  Calculate the nodes' depth based on the incoming and outgoing links
    //     Sets the nodes':
    //     - depth:  the depth in the graph
    //     - column: the depth (0, 1, 2, etc), as is relates to visual position from left to right
    //     - x0, x1: the x coordinates, as is relates to visual position from left to right
    // computeNodeDepths(graph)

    // 3.  Determine how the circular links will be drawn,
    //     either travelling back above the main chart ("top")
    //     or below the main chart ("bottom")
    selectCircularLinkTypes(graph, id);

    // 6.  Calculate the nodes' and links' vertical position within their respective column
    //     Also readjusts sankeyCircular size if circular links are needed, and node x's
    // computeNodeBreadths(graph, iterations, id)
    computeLinkBreadths(graph);

    // Force position of circular link type based on position
    graph.links.forEach(function (link) {
      if (link.circular) {
        link.circularLinkType = link.y0 + link.y1 < y1 ? 'top' : 'bottom';

        link.source.circularLinkType = link.circularLinkType;
        link.target.circularLinkType = link.circularLinkType;
      }
    });

    sortSourceLinks(graph, y1, id, false); // Sort links but do not move nodes
    sortTargetLinks(graph, y1, id);

    // 7.  Sort links per node, based on the links' source/target nodes' breadths
    // 8.  Adjust nodes that overlap links that span 2+ columns
    // var linkSortingIterations = 4; //Possibly let user control this number, like the iterations over node placement
    // for (var iteration = 0; iteration < linkSortingIterations; iteration++) {
    //
    //   sortSourceLinks(graph, y1, id)
    //   sortTargetLinks(graph, y1, id)
    //   resolveNodeLinkOverlaps(graph, y0, y1, id)
    //   sortSourceLinks(graph, y1, id)
    //   sortTargetLinks(graph, y1, id)
    //
    // }

    // 8.1  Adjust node and link positions back to fill height of chart area if compressed
    // fillHeight(graph, y0, y1)

    // 9. Calculate visually appealling path for the circular paths, and create the "d" string
    addCircularPathData(graph, circularLinkGap, y1, id);
    return graph;
  };

  // Populate the sourceLinks and targetLinks for each node.
  // Also, if the source and target are not objects, assume they are indices.
  function computeNodeLinks(graph) {
    graph.nodes.forEach(function (node, i) {
      node.index = i;
      node.sourceLinks = [];
      node.targetLinks = [];
    });
    var nodeById = map(graph.nodes, id);
    graph.links.forEach(function (link, i) {
      link.index = i;
      var source = link.source;
      var target = link.target;
      if (typeof source !== 'object') {
        source = link.source = find(nodeById, source);
      }
      if (typeof target !== 'object') {
        target = link.target = find(nodeById, target);
      }
      source.sourceLinks.push(link);
      target.targetLinks.push(link);
    });
    return graph;
  }

  // Compute the value (size) and cycleness of each node by summing the associated links.
  function computeNodeValues(graph) {
    graph.nodes.forEach(function (node) {
      node.partOfCycle = false;
      // TODO - compute proper node values
      node.value = Math.max(sum(node.sourceLinks, value), sum(node.targetLinks, value));
      node.sourceLinks.forEach(function (link) {
        if (link.circular) {
          node.partOfCycle = true;
          node.circularLinkType = link.circularLinkType;
        }
      });
      node.targetLinks.forEach(function (link) {
        if (link.circular) {
          node.partOfCycle = true;
          node.circularLinkType = link.circularLinkType;
        }
      });
    });
  }

  function getCircleMargins(graph) {
    var totalTopLinksWidth = 0,
      totalBottomLinksWidth = 0,
      totalRightLinksWidth = 0,
      totalLeftLinksWidth = 0;

    var maxColumn = max(graph.nodes, function (node) {
      return node.column;
    });

    graph.links.forEach(function (link) {
      if (link.circular) {
        if (link.circularLinkType == 'top') {
          totalTopLinksWidth = totalTopLinksWidth + link.width;
        } else {
          totalBottomLinksWidth = totalBottomLinksWidth + link.width;
        }

        if (link.target.column == 0) {
          totalLeftLinksWidth = totalLeftLinksWidth + link.width;
        }

        if (link.source.column == maxColumn) {
          totalRightLinksWidth = totalRightLinksWidth + link.width;
        }
      }
    });

    //account for radius of curves and padding between links
    totalTopLinksWidth =
      totalTopLinksWidth > 0
        ? totalTopLinksWidth + verticalMargin + baseRadius
        : totalTopLinksWidth;
    totalBottomLinksWidth =
      totalBottomLinksWidth > 0
        ? totalBottomLinksWidth + verticalMargin + baseRadius
        : totalBottomLinksWidth;
    totalRightLinksWidth =
      totalRightLinksWidth > 0
        ? totalRightLinksWidth + verticalMargin + baseRadius
        : totalRightLinksWidth;
    totalLeftLinksWidth =
      totalLeftLinksWidth > 0
        ? totalLeftLinksWidth + verticalMargin + baseRadius
        : totalLeftLinksWidth;

    return {
      top: totalTopLinksWidth,
      bottom: totalBottomLinksWidth,
      left: totalLeftLinksWidth,
      right: totalRightLinksWidth,
    };
  }

  // Update the x0, y0, x1 and y1 for the sankeyCircular, to allow space for any circular links
  function scaleSankeySize(graph, margin) {
    var maxColumn = max(graph.nodes, function (node) {
      return node.column;
    });

    var currentWidth = x1 - x0;
    var currentHeight = y1 - y0;

    var newWidth = currentWidth + margin.right + margin.left;
    var newHeight = currentHeight + margin.top + margin.bottom;

    var scaleX = currentWidth / newWidth;
    var scaleY = currentHeight / newHeight;

    x0 = x0 * scaleX + margin.left;
    x1 = margin.right == 0 ? x1 : x1 * scaleX;
    y0 = y0 * scaleY + margin.top;
    y1 = y1 * scaleY;

    graph.nodes.forEach(function (node) {
      node.x0 = x0 + node.column * ((x1 - x0 - dx) / maxColumn);
      node.x1 = node.x0 + dx;
    });

    return scaleY;
  }

  // Iteratively assign the depth for each node.
  // Nodes are assigned the maximum depth of incoming neighbors plus one;
  // nodes with no incoming links are assigned depth zero, while
  // nodes with no outgoing links are assigned the maximum depth.
  function computeNodeDepths(graph) {
    var nodes, next, x;

    for (nodes = graph.nodes, next = [], x = 0; nodes.length; ++x, nodes = next, next = []) {
      nodes.forEach(function (node) {
        node.depth = x;
        node.sourceLinks.forEach(function (link) {
          if (next.indexOf(link.target) < 0 && !link.circular) {
            next.push(link.target);
          }
        });
      });
    }

    for (nodes = graph.nodes, next = [], x = 0; nodes.length; ++x, nodes = next, next = []) {
      nodes.forEach(function (node) {
        node.height = x;
        node.targetLinks.forEach(function (link) {
          if (next.indexOf(link.source) < 0 && !link.circular) {
            next.push(link.source);
          }
        });
      });
    }

    // assign column numbers, and get max value
    graph.nodes.forEach(function (node) {
      node.column = sortNodes !== null ? node[sortNodes] : Math.floor(align.call(null, node, x));
    });
  }

  // Assign nodes' breadths, and then shift nodes that overlap (resolveCollisions)
  function computeNodeBreadths(graph, iterations, id) {
    var columns = nest()
      .key(function (d) {
        return d.column;
      })
      .sortKeys(ascending)
      .entries(graph.nodes)
      .map(function (d) {
        return d.values;
      });

    initializeNodeBreadth(id);
    resolveCollisions();

    for (var alpha = 1, n = iterations; n > 0; --n) {
      relaxLeftAndRight((alpha *= 0.99), id);
      resolveCollisions();
    }

    function initializeNodeBreadth(id) {
      //override py if nodePadding has been set
      if (paddingRatio) {
        var padding = Infinity;
        columns.forEach(function (nodes) {
          var thisPadding = (y1 * paddingRatio) / (nodes.length + 1);
          padding = thisPadding < padding ? thisPadding : padding;
        });
        py = padding;
      }

      var ky = min(columns, function (nodes) {
        return (y1 - y0 - (nodes.length - 1) * py) / sum(nodes, value);
      });

      //calculate the widths of the links
      ky = ky * scale;

      graph.links.forEach(function (link) {
        link.width = link.value * ky;
      });

      //determine how much to scale down the chart, based on circular links
      var margin = getCircleMargins(graph);
      var ratio = scaleSankeySize(graph, margin);

      //re-calculate widths
      ky = ky * ratio;

      graph.links.forEach(function (link) {
        link.width = link.value * ky;
      });

      columns.forEach(function (nodes) {
        var nodesLength = nodes.length;
        nodes.forEach(function (node, i) {
          if (node.depth == columns.length - 1 && nodesLength == 1) {
            node.y0 = y1 / 2 - node.value * ky;
            node.y1 = node.y0 + node.value * ky;
          } else if (node.depth == 0 && nodesLength == 1) {
            node.y0 = y1 / 2 - node.value * ky;
            node.y1 = node.y0 + node.value * ky;
          } else if (node.partOfCycle) {
            if (numberOfNonSelfLinkingCycles(node, id) == 0) {
              node.y0 = y1 / 2 + i;
              node.y1 = node.y0 + node.value * ky;
            } else if (node.circularLinkType == 'top') {
              node.y0 = y0 + i;
              node.y1 = node.y0 + node.value * ky;
            } else {
              node.y0 = y1 - node.value * ky - i;
              node.y1 = node.y0 + node.value * ky;
            }
          } else {
            if (margin.top == 0 || margin.bottom == 0) {
              node.y0 = ((y1 - y0) / nodesLength) * i;
              node.y1 = node.y0 + node.value * ky;
            } else {
              node.y0 = (y1 - y0) / 2 - nodesLength / 2 + i;
              node.y1 = node.y0 + node.value * ky;
            }
          }
        });
      });
    }

    // For each node in each column, check the node's vertical position in relation to its targets and sources vertical position
    // and shift up/down to be closer to the vertical middle of those targets and sources
    function relaxLeftAndRight(alpha) {
      columns.forEach(function (nodes) {
        nodes.forEach(function (node) {
          // check the node is not an orphan
          var nodeHeight;
          if (node.sourceLinks.length || node.targetLinks.length) {
            nodeHeight = node.y1 - node.y0;
            node.y0 = y1 / 2 - nodeHeight / 2;
            node.y1 = y1 / 2 + nodeHeight / 2;
            var avg = 0;

            var avgTargetY = mean(node.sourceLinks, linkTargetCenter);
            var avgSourceY = mean(node.targetLinks, linkSourceCenter);
            if (avgTargetY && avgSourceY) {
              avg = (avgTargetY + avgSourceY) / 2;
            } else {
              avg = avgTargetY || avgSourceY;
            }
            var dy = (avg - nodeCenter(node)) * alpha;
            // positive if it node needs to move down
            node.y0 += dy;
            node.y1 += dy;
          }
        });
      });
    }

    // For each column, check if nodes are overlapping, and if so, shift up/down
    function resolveCollisions() {
      columns.forEach(function (nodes) {
        var node,
          dy,
          y = y0,
          n = nodes.length,
          i;

        // Push any overlapping nodes down.
        nodes.sort(ascendingBreadth);

        for (i = 0; i < n; ++i) {
          node = nodes[i];
          dy = y - node.y0;

          if (dy > 0) {
            node.y0 += dy;
            node.y1 += dy;
          }
          y = node.y1 + py;
        }

        // If the bottommost node goes outside the bounds, push it back up.
        dy = y - py - y1;
        if (dy > 0) {
          (y = node.y0 -= dy), (node.y1 -= dy);

          // Push any overlapping nodes back up.
          for (i = n - 2; i >= 0; --i) {
            node = nodes[i];
            dy = node.y1 + py - y;
            if (dy > 0) (node.y0 -= dy), (node.y1 -= dy);
            y = node.y0;
          }
        }
      });
    }
  }

  // Assign the links y0 and y1 based on source/target nodes position,
  // plus the link's relative position to other links to the same node
  function computeLinkBreadths(graph) {
    graph.nodes.forEach(function (node) {
      node.sourceLinks.sort(ascendingTargetBreadth);
      node.targetLinks.sort(ascendingSourceBreadth);
    });
    graph.nodes.forEach(function (node) {
      var y0 = node.y0;
      var y1 = y0;

      // start from the bottom of the node for cycle links
      var y0cycle = node.y1;
      var y1cycle = y0cycle;

      node.sourceLinks.forEach(function (link) {
        if (link.circular) {
          link.y0 = y0cycle - link.width / 2;
          y0cycle = y0cycle - link.width;
        } else {
          link.y0 = y0 + link.width / 2;
          y0 += link.width;
        }
      });
      node.targetLinks.forEach(function (link) {
        if (link.circular) {
          link.y1 = y1cycle - link.width / 2;
          y1cycle = y1cycle - link.width;
        } else {
          link.y1 = y1 + link.width / 2;
          y1 += link.width;
        }
      });
    });
  }

  return sankeyCircular;
}

/// /////////////////////////////////////////////////////////////////////////////////
// Cycle functions
// portion of code to detect circular links based on Colin Fergus' bl.ock https://gist.github.com/cfergus/3956043

// Identify circles in the link objects
function identifyCircles(graph, id, sortNodes) {
  console.log(graph);
  var circularLinkID = 0;
  if (sortNodes === null) {
    // Building adjacency graph
    var adjList = [];
    for (var i = 0; i < graph.links.length; i++) {
      var link = graph.links[i];
      var source = link.source.index;
      var target = link.target.index;
      if (!adjList[source]) adjList[source] = [];
      if (!adjList[target]) adjList[target] = [];

      // Add links if not already in set
      if (adjList[source].indexOf(target) === -1) adjList[source].push(target);
    }

    // Find all elementary circuits
    var cycles = findCircuits(adjList);

    // Sort by circuits length
    cycles.sort(function (a, b) {
      return a.length - b.length;
    });

    var circularLinks = {};
    for (i = 0; i < cycles.length; i++) {
      var cycle = cycles[i];
      var last = cycle.slice(-2);
      if (!circularLinks[last[0]]) circularLinks[last[0]] = {};
      circularLinks[last[0]][last[1]] = true;
    }

    graph.links.forEach(function (link) {
      var target = link.target.index;
      var source = link.source.index;
      // If self-linking or a back-edge
      if (target === source || (circularLinks[source] && circularLinks[source][target])) {
        link.circular = true;
        link.circularLinkID = circularLinkID;
        circularLinkID = circularLinkID + 1;
      } else {
        link.circular = false;
      }
    });
  } else {
    graph.links.forEach(function (link) {
      if (link.source[sortNodes] < link.target[sortNodes]) {
        link.circular = false;
      } else {
        link.circular = true;
        link.circularLinkID = circularLinkID;
        circularLinkID = circularLinkID + 1;
      }
    });
  }
}

// Assign a circular link type (top or bottom), based on:
// - if the source/target node already has circular links, then use the same type
// - if not, choose the type with fewer links
function selectCircularLinkTypes(graph, id) {
  var numberOfTops = 0;
  var numberOfBottoms = 0;
  graph.links.forEach(function (link) {
    if (link.circular) {
      // if either souce or target has type already use that
      if (link.source.circularLinkType || link.target.circularLinkType) {
        // default to source type if available
        link.circularLinkType = link.source.circularLinkType
          ? link.source.circularLinkType
          : link.target.circularLinkType;
      } else {
        link.circularLinkType = numberOfTops < numberOfBottoms ? 'top' : 'bottom';
      }

      if (link.circularLinkType == 'top') {
        numberOfTops = numberOfTops + 1;
      } else {
        numberOfBottoms = numberOfBottoms + 1;
      }

      graph.nodes.forEach(function (node) {
        if (
          getNodeID(node, id) == getNodeID(link.source, id) ||
          getNodeID(node, id) == getNodeID(link.target, id)
        ) {
          node.circularLinkType = link.circularLinkType;
        }
      });
    }
  });

  //correct self-linking links to be same direction as node
  graph.links.forEach(function (link) {
    if (link.circular) {
      //if both source and target node are same type, then link should have same type
      if (link.source.circularLinkType == link.target.circularLinkType) {
        link.circularLinkType = link.source.circularLinkType;
      }
      //if link is selflinking, then link should have same type as node
      if (selfLinking(link, id)) {
        link.circularLinkType = link.source.circularLinkType;
      }
    }
  });
}

// Given a node, find all links for which this is a source in the current 'known' graph
function findLinksOutward(node, graph) {
  var children = [];

  for (var i = 0; i < graph.length; i++) {
    if (node == graph[i].source) {
      children.push(graph[i]);
    }
  }

  return children;
}

// Return the angle between a straight line between the source and target of the link, and the vertical plane of the node
function linkAngle(link) {
  var adjacent = Math.abs(link.y1 - link.y0);
  var opposite = Math.abs(link.target.x0 - link.source.x1);

  return Math.atan(opposite / adjacent);
}

// Check if two circular links potentially overlap
function circularLinksCross(link1, link2) {
  if (link1.source.column < link2.target.column) {
    return false;
  } else if (link1.target.column > link2.source.column) {
    return false;
  } else {
    return true;
  }
}

// Return the number of circular links for node, not including self linking links
function numberOfNonSelfLinkingCycles(node, id) {
  var sourceCount = 0;
  node.sourceLinks.forEach(function (l) {
    sourceCount = l.circular && !selfLinking(l, id) ? sourceCount + 1 : sourceCount;
  });

  var targetCount = 0;
  node.targetLinks.forEach(function (l) {
    targetCount = l.circular && !selfLinking(l, id) ? targetCount + 1 : targetCount;
  });

  return sourceCount + targetCount;
}

// Check if a circular link is the only circular link for both its source and target node
function onlyCircularLink(link) {
  var nodeSourceLinks = link.source.sourceLinks;
  var sourceCount = 0;
  nodeSourceLinks.forEach(function (l) {
    sourceCount = l.circular ? sourceCount + 1 : sourceCount;
  });

  var nodeTargetLinks = link.target.targetLinks;
  var targetCount = 0;
  nodeTargetLinks.forEach(function (l) {
    targetCount = l.circular ? targetCount + 1 : targetCount;
  });

  if (sourceCount > 1 || targetCount > 1) {
    return false;
  } else {
    return true;
  }
}

// creates vertical buffer values per set of top/bottom links
function calcVerticalBuffer(links, circularLinkGap, id) {
  links.sort(sortLinkColumnAscending);
  links.forEach(function (link, i) {
    var buffer = 0;

    if (selfLinking(link, id) && onlyCircularLink(link)) {
      link.circularPathData.verticalBuffer = buffer + link.width / 2;
    } else {
      var j = 0;
      for (j; j < i; j++) {
        if (circularLinksCross(links[i], links[j])) {
          var bufferOverThisLink =
            links[j].circularPathData.verticalBuffer + links[j].width / 2 + circularLinkGap;
          buffer = bufferOverThisLink > buffer ? bufferOverThisLink : buffer;
        }
      }

      link.circularPathData.verticalBuffer = buffer + link.width / 2;
    }
  });

  return links;
}

// calculate the optimum path for a link to reduce overlaps
export function addCircularPathData(graph, circularLinkGap, y1, id) {
  //var baseRadius = 10
  var buffer = 5;
  //var verticalMargin = 25

  var minY = min(graph.links, function (link) {
    return link.source.y0;
  });

  // create object for circular Path Data
  graph.links.forEach(function (link) {
    if (link.circular) {
      link.circularPathData = {};
    }
  });

  // calc vertical offsets per top/bottom links
  var topLinks = graph.links.filter(function (l) {
    return l.circularLinkType == 'top';
  });
  /* topLinks = */ calcVerticalBuffer(topLinks, circularLinkGap, id);

  var bottomLinks = graph.links.filter(function (l) {
    return l.circularLinkType == 'bottom';
  });
  /* bottomLinks = */ calcVerticalBuffer(bottomLinks, circularLinkGap, id);

  // add the base data for each link
  graph.links.forEach(function (link) {
    if (link.circular) {
      link.circularPathData.arcRadius = link.width + baseRadius;
      link.circularPathData.leftNodeBuffer = buffer;
      link.circularPathData.rightNodeBuffer = buffer;
      link.circularPathData.sourceWidth = link.source.x1 - link.source.x0;
      link.circularPathData.sourceX = link.source.x1;
      link.circularPathData.targetX = link.target.x0 - (link.target.x1 - link.target.x0) / 2;
      link.circularPathData.sourceY = link.source.y0;
      link.circularPathData.targetY = link.target.y0;

      // for self linking paths, and that the only circular link in/out of that node
      if (selfLinking(link, id) && onlyCircularLink(link)) {
        link.circularPathData.leftSmallArcRadius = baseRadius + link.width / 2;
        link.circularPathData.leftLargeArcRadius = baseRadius + link.width / 2;
        link.circularPathData.rightSmallArcRadius = baseRadius + link.width / 2;
        link.circularPathData.rightLargeArcRadius = baseRadius + link.width / 2;

        if (link.circularLinkType == 'bottom') {
          link.circularPathData.verticalFullExtent =
            link.source.y1 + verticalMargin + link.circularPathData.verticalBuffer;
          link.circularPathData.verticalLeftInnerExtent =
            link.circularPathData.verticalFullExtent - link.circularPathData.leftLargeArcRadius;
          link.circularPathData.verticalRightInnerExtent =
            link.circularPathData.verticalFullExtent - link.circularPathData.rightLargeArcRadius;
        } else {
          // top links
          link.circularPathData.verticalFullExtent =
            link.source.y0 - verticalMargin - link.circularPathData.verticalBuffer;
          link.circularPathData.verticalLeftInnerExtent =
            link.circularPathData.verticalFullExtent + link.circularPathData.leftLargeArcRadius;
          link.circularPathData.verticalRightInnerExtent =
            link.circularPathData.verticalFullExtent + link.circularPathData.rightLargeArcRadius;
        }
      } else {
        // else calculate normally
        // add left extent coordinates, based on links with same source column and circularLink type
        var thisColumn = link.source.column;
        var thisCircularLinkType = link.circularLinkType;
        var sameColumnLinks = graph.links.filter(function (l) {
          return l.source.column == thisColumn && l.circularLinkType == thisCircularLinkType;
        });

        if (link.circularLinkType == 'bottom') {
          sameColumnLinks.sort(sortLinkSourceYDescending);
        } else {
          sameColumnLinks.sort(sortLinkSourceYAscending);
        }

        var radiusOffset = 0;
        sameColumnLinks.forEach(function (l, i) {
          if (l.circularLinkID == link.circularLinkID) {
            link.circularPathData.leftSmallArcRadius = baseRadius + link.width / 2 + radiusOffset;
            link.circularPathData.leftLargeArcRadius =
              baseRadius + link.width / 2 + i * circularLinkGap + radiusOffset;
          }
          radiusOffset = radiusOffset + l.width;
        });

        // add right extent coordinates, based on links with same target column and circularLink type
        thisColumn = link.target.column;
        sameColumnLinks = graph.links.filter(function (l) {
          return l.target.column == thisColumn && l.circularLinkType == thisCircularLinkType;
        });
        if (link.circularLinkType == 'bottom') {
          sameColumnLinks.sort(sortLinkTargetYDescending);
        } else {
          sameColumnLinks.sort(sortLinkTargetYAscending);
        }

        radiusOffset = 0;
        sameColumnLinks.forEach(function (l, i) {
          if (l.circularLinkID == link.circularLinkID) {
            link.circularPathData.rightSmallArcRadius = baseRadius + link.width / 2 + radiusOffset;
            link.circularPathData.rightLargeArcRadius =
              baseRadius + link.width / 2 + i * circularLinkGap + radiusOffset;
          }
          radiusOffset = radiusOffset + l.width;
        });

        // bottom links
        if (link.circularLinkType == 'bottom') {
          link.circularPathData.verticalFullExtent = link.source.y1 - verticalMargin;
          link.circularPathData.verticalLeftInnerExtent =
            link.circularPathData.verticalFullExtent - link.circularPathData.leftLargeArcRadius;
          link.circularPathData.verticalRightInnerExtent =
            link.circularPathData.verticalFullExtent - link.circularPathData.rightLargeArcRadius;
        } else {
          // top links
          link.circularPathData.verticalFullExtent =
            minY - verticalMargin - link.circularPathData.verticalBuffer;
          link.circularPathData.verticalLeftInnerExtent =
            link.circularPathData.verticalFullExtent + link.circularPathData.leftLargeArcRadius;
          link.circularPathData.verticalRightInnerExtent =
            link.circularPathData.verticalFullExtent + link.circularPathData.rightLargeArcRadius;
        }
      }

      // all links
      link.circularPathData.leftInnerExtent =
        link.circularPathData.sourceX + link.circularPathData.leftNodeBuffer;
      link.circularPathData.rightInnerExtent =
        link.circularPathData.targetX - link.circularPathData.rightNodeBuffer;
      link.circularPathData.leftFullExtent =
        link.circularPathData.sourceX +
        link.circularPathData.leftLargeArcRadius +
        link.circularPathData.leftNodeBuffer;
      link.circularPathData.rightFullExtent =
        link.circularPathData.targetX -
        link.circularPathData.rightLargeArcRadius -
        link.circularPathData.rightNodeBuffer;
    }

    if (link.circular) {
      link.path = createCircularPathString(link);
    } else {
      var normalPath = linkHorizontal()
        .source(function (d) {
          var x = d.source.x0 + (d.source.x1 - d.source.x0);
          var y = d.source.y0;
          return [x, y];
        })
        .target(function (d) {
          var x = d.target.x0 - (d.target.x1 - d.target.x0) / 2;
          var y = d.target.y0;
          return [x, y];
        });
      link.path = normalPath(link);
    }
  });
}

// create a d path using the addCircularPathData
function createCircularPathString(link) {
  var pathString = '';
  // 'pathData' is assigned a value but never used
  // var pathData = {}

  if (link.circularLinkType == 'top') {
    pathString =
      // start at the right of the source node
      'M' +
      link.circularPathData.sourceX +
      ' ' +
      link.circularPathData.sourceY +
      ' ' +
      // line right to buffer point
      'L' +
      link.circularPathData.leftInnerExtent +
      ' ' +
      link.circularPathData.sourceY +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.leftLargeArcRadius +
      ' ' +
      link.circularPathData.leftSmallArcRadius +
      ' 0 0 0 ' +
      // End of arc X //End of arc Y
      link.circularPathData.leftFullExtent +
      ' ' +
      (link.circularPathData.sourceY - link.circularPathData.leftSmallArcRadius) +
      ' ' + // End of arc X
      // line up to buffer point
      'L' +
      link.circularPathData.leftFullExtent +
      ' ' +
      link.circularPathData.verticalLeftInnerExtent +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.leftLargeArcRadius +
      ' ' +
      link.circularPathData.leftLargeArcRadius +
      ' 0 0 0 ' +
      // End of arc X //End of arc Y
      link.circularPathData.leftInnerExtent +
      ' ' +
      link.circularPathData.verticalFullExtent +
      ' ' + // End of arc X
      // line left to buffer point
      'L' +
      link.circularPathData.rightInnerExtent +
      ' ' +
      link.circularPathData.verticalFullExtent +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.rightLargeArcRadius +
      ' ' +
      link.circularPathData.rightLargeArcRadius +
      ' 0 0 0 ' +
      // End of arc X //End of arc Y
      link.circularPathData.rightFullExtent +
      ' ' +
      link.circularPathData.verticalRightInnerExtent +
      ' ' + // End of arc X
      // line down
      'L' +
      link.circularPathData.rightFullExtent +
      ' ' +
      (link.circularPathData.targetY - link.circularPathData.rightSmallArcRadius) +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.rightLargeArcRadius +
      ' ' +
      link.circularPathData.rightSmallArcRadius +
      ' 0 0 0 ' +
      // End of arc X //End of arc Y
      link.circularPathData.rightInnerExtent +
      ' ' +
      link.circularPathData.targetY +
      ' ' + // End of arc X
      // line to end
      'L' +
      link.circularPathData.targetX +
      ' ' +
      link.circularPathData.targetY;
  } else {
    // bottom path
    pathString =
      // start at the right of the source node
      'M' +
      link.circularPathData.sourceX +
      ' ' +
      link.circularPathData.sourceY +
      ' ' +
      // line right to buffer point
      'L' +
      link.circularPathData.leftInnerExtent +
      ' ' +
      link.circularPathData.sourceY +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.leftLargeArcRadius +
      ' ' +
      link.circularPathData.leftSmallArcRadius +
      ' 0 0 1 ' +
      // End of arc X //End of arc Y
      link.circularPathData.leftFullExtent +
      ' ' +
      (link.circularPathData.sourceY + link.circularPathData.leftSmallArcRadius) +
      ' ' + // End of arc X
      // line down to buffer point
      'L' +
      link.circularPathData.leftFullExtent +
      ' ' +
      // TODO this is what we need to adjust to reduce the height
      link.circularPathData.verticalLeftInnerExtent +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.leftLargeArcRadius +
      ' ' +
      link.circularPathData.leftLargeArcRadius +
      ' 0 0 1 ' +
      // End of arc X //End of arc Y
      link.circularPathData.leftInnerExtent +
      ' ' +
      link.circularPathData.verticalFullExtent +
      ' ' + // End of arc X
      // line left to buffer point
      'L' +
      link.circularPathData.rightInnerExtent +
      ' ' +
      link.circularPathData.verticalFullExtent +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.rightLargeArcRadius +
      ' ' +
      link.circularPathData.rightLargeArcRadius +
      ' 0 0 1 ' +
      // End of arc X //End of arc Y
      link.circularPathData.rightFullExtent +
      ' ' +
      link.circularPathData.verticalRightInnerExtent +
      ' ' + // End of arc X
      // line up
      'L' +
      link.circularPathData.rightFullExtent +
      ' ' +
      (link.circularPathData.targetY + link.circularPathData.rightSmallArcRadius) +
      ' ' +
      // Arc around: Centre of arc X and  //Centre of arc Y
      'A' +
      link.circularPathData.rightLargeArcRadius +
      ' ' +
      link.circularPathData.rightSmallArcRadius +
      ' 0 0 1 ' +
      // End of arc X //End of arc Y
      link.circularPathData.rightInnerExtent +
      ' ' +
      link.circularPathData.targetY +
      ' ' + // End of arc X
      // line to end
      'L' +
      link.circularPathData.targetX +
      ' ' +
      // TODO adjust this so that we make room for the arrow
      link.circularPathData.targetY;
  }

  return pathString;
}

// sort links based on the distance between the source and tartget node columns
// if the same, then use Y position of the source node
function sortLinkColumnAscending(link1, link2) {
  if (linkColumnDistance(link1) == linkColumnDistance(link2)) {
    return link1.circularLinkType == 'bottom'
      ? sortLinkSourceYDescending(link1, link2)
      : sortLinkSourceYAscending(link1, link2);
  } else {
    return linkColumnDistance(link2) - linkColumnDistance(link1);
  }
}

// sort ascending links by their source vertical position, y0
function sortLinkSourceYAscending(link1, link2) {
  return link1.y0 - link2.y0;
}

// sort descending links by their source vertical position, y0
function sortLinkSourceYDescending(link1, link2) {
  return link2.y0 - link1.y0;
}

// sort ascending links by their target vertical position, y1
function sortLinkTargetYAscending(link1, link2) {
  return link1.y1 - link2.y1;
}

// sort descending links by their target vertical position, y1
function sortLinkTargetYDescending(link1, link2) {
  return link2.y1 - link1.y1;
}

// return the distance between the link's target and source node, in terms of the nodes' column
function linkColumnDistance(link) {
  return link.target.column - link.source.column;
}

// return the distance between the link's target and source node, in terms of the nodes' X coordinate
function linkXLength(link) {
  return link.target.x0 - link.source.x1;
}

// Return the Y coordinate on the longerLink path * which is perpendicular shorterLink's source.
// * approx, based on a straight line from target to source, when in fact the path is a bezier
function linkPerpendicularYToLinkSource(longerLink, shorterLink) {
  // get the angle for the longer link
  var angle = linkAngle(longerLink);

  // get the adjacent length to the other link's x position
  var heightFromY1ToPependicular = linkXLength(shorterLink) / Math.tan(angle);

  // add or subtract from longer link1's original y1, depending on the slope
  var yPerpendicular =
    incline(longerLink) == 'up'
      ? longerLink.y1 + heightFromY1ToPependicular
      : longerLink.y1 - heightFromY1ToPependicular;

  return yPerpendicular;
}

// Return the Y coordinate on the longerLink path * which is perpendicular shorterLink's source.
// * approx, based on a straight line from target to source, when in fact the path is a bezier
function linkPerpendicularYToLinkTarget(longerLink, shorterLink) {
  // get the angle for the longer link
  var angle = linkAngle(longerLink);

  // get the adjacent length to the other link's x position
  var heightFromY1ToPependicular = linkXLength(shorterLink) / Math.tan(angle);

  // add or subtract from longer link's original y1, depending on the slope
  var yPerpendicular =
    incline(longerLink) == 'up'
      ? longerLink.y1 - heightFromY1ToPependicular
      : longerLink.y1 + heightFromY1ToPependicular;

  return yPerpendicular;
}

// Move any nodes that overlap links which span 2+ columns
function resolveNodeLinkOverlaps(graph, y0, y1, id) {
  graph.links.forEach(function (link) {
    if (link.circular) {
      return;
    }

    if (link.target.column - link.source.column > 1) {
      var columnToTest = link.source.column + 1;
      var maxColumnToTest = link.target.column - 1;

      var i = 1;
      var numberOfColumnsToTest = maxColumnToTest - columnToTest + 1;

      for (columnToTest, i = 1; columnToTest <= maxColumnToTest; columnToTest++, i++) {
        graph.nodes.forEach(function (node) {
          if (node.column == columnToTest) {
            var t = i / (numberOfColumnsToTest + 1);

            // Find all the points of a cubic bezier curve in javascript
            // https://stackoverflow.com/questions/15397596/find-all-the-points-of-a-cubic-bezier-curve-in-javascript

            var B0_t = Math.pow(1 - t, 3);
            var B1_t = 3 * t * Math.pow(1 - t, 2);
            var B2_t = 3 * Math.pow(t, 2) * (1 - t);
            var B3_t = Math.pow(t, 3);

            var py_t = B0_t * link.y0 + B1_t * link.y0 + B2_t * link.y1 + B3_t * link.y1;

            var linkY0AtColumn = py_t - link.width / 2;
            var linkY1AtColumn = py_t + link.width / 2;
            var dy;

            // If top of link overlaps node, push node up
            if (linkY0AtColumn > node.y0 && linkY0AtColumn < node.y1) {
              dy = node.y1 - linkY0AtColumn + 10;
              dy = node.circularLinkType == 'bottom' ? dy : -dy;

              node = adjustNodeHeight(node, dy, y0, y1);

              // check if other nodes need to move up too
              graph.nodes.forEach(function (otherNode) {
                // don't need to check itself or nodes at different columns
                if (
                  getNodeID(otherNode, id) == getNodeID(node, id) ||
                  otherNode.column != node.column
                ) {
                  return;
                }
                if (nodesOverlap(node, otherNode)) {
                  adjustNodeHeight(otherNode, dy, y0, y1);
                }
              });
            } else if (linkY1AtColumn > node.y0 && linkY1AtColumn < node.y1) {
              // If bottom of link overlaps node, push node down
              dy = linkY1AtColumn - node.y0 + 10;

              node = adjustNodeHeight(node, dy, y0, y1);

              // check if other nodes need to move down too
              graph.nodes.forEach(function (otherNode) {
                // don't need to check itself or nodes at different columns
                if (
                  getNodeID(otherNode, id) == getNodeID(node, id) ||
                  otherNode.column != node.column
                ) {
                  return;
                }
                if (otherNode.y0 < node.y1 && otherNode.y1 > node.y1) {
                  adjustNodeHeight(otherNode, dy, y0, y1);
                }
              });
            } else if (linkY0AtColumn < node.y0 && linkY1AtColumn > node.y1) {
              // if link completely overlaps node
              dy = linkY1AtColumn - node.y0 + 10;

              node = adjustNodeHeight(node, dy, y0, y1);

              graph.nodes.forEach(function (otherNode) {
                // don't need to check itself or nodes at different columns
                if (
                  getNodeID(otherNode, id) == getNodeID(node, id) ||
                  otherNode.column != node.column
                ) {
                  return;
                }
                if (otherNode.y0 < node.y1 && otherNode.y1 > node.y1) {
                  adjustNodeHeight(otherNode, dy, y0, y1);
                }
              });
            }
          }
        });
      }
    }
  });
}

// check if two nodes overlap
function nodesOverlap(nodeA, nodeB) {
  // test if nodeA top partially overlaps nodeB
  if (nodeA.y0 > nodeB.y0 && nodeA.y0 < nodeB.y1) {
    return true;
  } else if (nodeA.y1 > nodeB.y0 && nodeA.y1 < nodeB.y1) {
    // test if nodeA bottom partially overlaps nodeB
    return true;
  } else if (nodeA.y0 < nodeB.y0 && nodeA.y1 > nodeB.y1) {
    // test if nodeA covers nodeB
    return true;
  } else {
    return false;
  }
}

// update a node, and its associated links, vertical positions (y0, y1)
function adjustNodeHeight(node, dy, sankeyY0, sankeyY1) {
  if (node.y0 + dy >= sankeyY0 && node.y1 + dy <= sankeyY1) {
    node.y0 = node.y0 + dy;
    node.y1 = node.y1 + dy;

    node.targetLinks.forEach(function (l) {
      l.y1 = l.y1 + dy;
    });

    node.sourceLinks.forEach(function (l) {
      l.y0 = l.y0 + dy;
    });
  }
  return node;
}

// sort and set the links' y0 for each node
function sortSourceLinks(graph, y1, id, moveNodes) {
  graph.nodes.forEach(function (node) {
    // move any nodes up which are off the bottom
    if (moveNodes && node.y + (node.y1 - node.y0) > y1) {
      node.y = node.y - (node.y + (node.y1 - node.y0) - y1);
    }

    var nodesSourceLinks = graph.links.filter(function (l) {
      return getNodeID(l.source, id) == getNodeID(node, id);
    });

    var nodeSourceLinksLength = nodesSourceLinks.length;

    // if more than 1 link then sort
    if (nodeSourceLinksLength > 1) {
      nodesSourceLinks.sort(function (link1, link2) {
        // if both are not circular...
        if (!link1.circular && !link2.circular) {
          // if the target nodes are the same column, then sort by the link's target y
          if (link1.target.column == link2.target.column) {
            return link1.y1 - link2.y1;
          } else if (!sameInclines(link1, link2)) {
            // if the links slope in different directions, then sort by the link's target y
            return link1.y1 - link2.y1;

            // if the links slope in same directions, then sort by any overlap
          } else {
            if (link1.target.column > link2.target.column) {
              var link2Adj = linkPerpendicularYToLinkTarget(link2, link1);
              return link1.y1 - link2Adj;
            }
            if (link2.target.column > link1.target.column) {
              var link1Adj = linkPerpendicularYToLinkTarget(link1, link2);
              return link1Adj - link2.y1;
            }
          }
        }

        // if only one is circular, the move top links up, or bottom links down
        if (link1.circular && !link2.circular) {
          return link1.circularLinkType == 'top' ? -1 : 1;
        } else if (link2.circular && !link1.circular) {
          return link2.circularLinkType == 'top' ? 1 : -1;
        }

        // if both links are circular...
        if (link1.circular && link2.circular) {
          // ...and they both loop the same way (both top)
          if (
            link1.circularLinkType === link2.circularLinkType &&
            link1.circularLinkType == 'top'
          ) {
            // ...and they both connect to a target with same column, then sort by the target's y
            if (link1.target.column === link2.target.column) {
              return link1.target.y1 - link2.target.y1;
            } else {
              // ...and they connect to different column targets, then sort by how far back they
              return link2.target.column - link1.target.column;
            }
          } else if (
            link1.circularLinkType === link2.circularLinkType &&
            link1.circularLinkType == 'bottom'
          ) {
            // ...and they both loop the same way (both bottom)
            // ...and they both connect to a target with same column, then sort by the target's y
            if (link1.target.column === link2.target.column) {
              return link2.target.y1 - link1.target.y1;
            } else {
              // ...and they connect to different column targets, then sort by how far back they
              return link1.target.column - link2.target.column;
            }
          } else {
            // ...and they loop around different ways, the move top up and bottom down
            return link1.circularLinkType == 'top' ? -1 : 1;
          }
        }
      });
    }

    // update y0 for links
    var ySourceOffset = node.y0;

    nodesSourceLinks.forEach(function (link) {
      link.y0 = ySourceOffset + link.width / 2;
      ySourceOffset = ySourceOffset + link.width;
    });

    // correct any circular bottom links so they are at the bottom of the node
    nodesSourceLinks.forEach(function (link, i) {
      if (link.circularLinkType == 'bottom') {
        var j = i + 1;
        var offsetFromBottom = 0;
        // sum the widths of any links that are below this link
        for (j; j < nodeSourceLinksLength; j++) {
          offsetFromBottom = offsetFromBottom + nodesSourceLinks[j].width;
        }
        link.y0 = node.y1 - offsetFromBottom - link.width / 2;
      }
    });
  });
}

// sort and set the links' y1 for each node
function sortTargetLinks(graph, y1, id) {
  graph.nodes.forEach(function (node) {
    var nodesTargetLinks = graph.links.filter(function (l) {
      return getNodeID(l.target, id) == getNodeID(node, id);
    });

    var nodesTargetLinksLength = nodesTargetLinks.length;

    if (nodesTargetLinksLength > 1) {
      nodesTargetLinks.sort(function (link1, link2) {
        // if both are not circular, the base on the source y position
        if (!link1.circular && !link2.circular) {
          if (link1.source.column == link2.source.column) {
            return link1.y0 - link2.y0;
          } else if (!sameInclines(link1, link2)) {
            return link1.y0 - link2.y0;
          } else {
            // get the angle of the link to the further source node (ie the smaller column)
            if (link2.source.column < link1.source.column) {
              var link2Adj = linkPerpendicularYToLinkSource(link2, link1);

              return link1.y0 - link2Adj;
            }
            if (link1.source.column < link2.source.column) {
              var link1Adj = linkPerpendicularYToLinkSource(link1, link2);

              return link1Adj - link2.y0;
            }
          }
        }

        // if only one is circular, the move top links up, or bottom links down
        if (link1.circular && !link2.circular) {
          return link1.circularLinkType == 'top' ? -1 : 1;
        } else if (link2.circular && !link1.circular) {
          return link2.circularLinkType == 'top' ? 1 : -1;
        }

        // if both links are circular...
        if (link1.circular && link2.circular) {
          // ...and they both loop the same way (both top)
          if (
            link1.circularLinkType === link2.circularLinkType &&
            link1.circularLinkType == 'top'
          ) {
            // ...and they both connect to a target with same column, then sort by the target's y
            if (link1.source.column === link2.source.column) {
              return link1.source.y1 - link2.source.y1;
            } else {
              // ...and they connect to different column targets, then sort by how far back they
              return link1.source.column - link2.source.column;
            }
          } else if (
            link1.circularLinkType === link2.circularLinkType &&
            link1.circularLinkType == 'bottom'
          ) {
            // ...and they both loop the same way (both bottom)
            // ...and they both connect to a target with same column, then sort by the target's y
            if (link1.source.column === link2.source.column) {
              return link1.source.y1 - link2.source.y1;
            } else {
              // ...and they connect to different column targets, then sort by how far back they
              return link2.source.column - link1.source.column;
            }
          } else {
            // ...and they loop around different ways, the move top up and bottom down
            return link1.circularLinkType == 'top' ? -1 : 1;
          }
        }
      });
    }

    // update y1 for links
    var yTargetOffset = node.y0;

    nodesTargetLinks.forEach(function (link) {
      link.y1 = yTargetOffset + link.width / 2;
      yTargetOffset = yTargetOffset + link.width;
    });

    // correct any circular bottom links so they are at the bottom of the node
    nodesTargetLinks.forEach(function (link, i) {
      if (link.circularLinkType == 'bottom') {
        var j = i + 1;
        var offsetFromBottom = 0;
        // sum the widths of any links that are below this link
        for (j; j < nodesTargetLinksLength; j++) {
          offsetFromBottom = offsetFromBottom + nodesTargetLinks[j].width;
        }
        link.y1 = node.y1 - offsetFromBottom - link.width / 2;
      }
    });
  });
}

// test if links both slope up, or both slope down
function sameInclines(link1, link2) {
  return incline(link1) == incline(link2);
}

// returns the slope of a link, from source to target
// up => slopes up from source to target
// down => slopes down from source to target
function incline(link) {
  return link.y0 - link.y1 > 0 ? 'up' : 'down';
}

// check if link is self linking, ie links a node to the same node
function selfLinking(link, id) {
  return getNodeID(link.source, id) == getNodeID(link.target, id);
}

function fillHeight(graph, y0, y1) {
  var nodes = graph.nodes;
  var links = graph.links;

  var top = false;
  var bottom = false;

  links.forEach(function (link) {
    if (link.circularLinkType == 'top') {
      top = true;
    } else if (link.circularLinkType == 'bottom') {
      bottom = true;
    }
  });

  if (top == false || bottom == false) {
    var minY0 = min(nodes, function (node) {
      return node.y0;
    });
    var maxY1 = max(nodes, function (node) {
      return node.y1;
    });
    var currentHeight = maxY1 - minY0;
    var chartHeight = y1 - y0;
    var ratio = chartHeight / currentHeight;

    nodes.forEach(function (node) {
      var nodeHeight = (node.y1 - node.y0) * ratio;
      node.y0 = (node.y0 - minY0) * ratio;
      node.y1 = node.y0 + nodeHeight;
    });

    links.forEach(function (link) {
      link.y0 = (link.y0 - minY0) * ratio;
      link.y1 = (link.y1 - minY0) * ratio;
      link.width = link.width * ratio;
    });
  }
}

function resolveNodesOverlap(graph, y0, py) {
  var columns = nest()
    .key(function (d) {
      return d.column;
    })
    .sortKeys(ascending)
    .entries(graph.nodes)
    .map(function (d) {
      return d.values;
    });

  columns.forEach(function (nodes) {
    var node,
      dy,
      y = y0,
      n = nodes.length,
      i;
    // Push any overlapping nodes down.
    nodes.sort(ascendingBreadth);

    for (i = 0; i < n; ++i) {
      node = nodes[i];
      dy = y - node.y0;

      if (dy > 0) {
        node.y0 += dy;
        node.y1 += dy;
        node.targetLinks.forEach(function (l) {
          l.y1 = l.y1 + dy;
        });
        node.sourceLinks.forEach(function (l) {
          l.y0 = l.y0 + dy;
        });
      }
      y = node.y1 + py;
    }
  });
}
