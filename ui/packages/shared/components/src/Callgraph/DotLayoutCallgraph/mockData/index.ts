export const dotGraph = `{
 n1 [Label = "n1"];
 n2 [Label = "n2"];
 n3 [Label = "n3"];
 n1 -- n2;
 n1 -- n3;
 n2 -- n2;
}`;

export const jsonGraph = {
  nodes: [
    {
      id: 'n0',
      label: 'A node',
      value: 3,
    },
    {
      id: 'n1',
      label: 'Another node',
      value: 2,
    },
    {
      id: 'n2',
      label: 'And a last one',
      value: 1,
    },
  ],
  edges: [
    {
      id: 'e0',
      source: 'n0',
      target: 'n1',
      label: 'edge 1',
    },
    {
      id: 'e1',
      source: 'n1',
      target: 'n2',
      label: 'edge 2',
    },
    {
      id: 'e2',
      source: 'n2',
      target: 'n0',
      label: 'edge 3 ',
    },
  ],
};

// just a string, doesn't need to be on separate lines
export const graphvizDot = `
digraph {
  N1 [id="node1"]
  N2 [id="node2"]
  N3 [id="node3"]
  N4 [id="node4"]
  N1 -> N2 [id="e1" label="e1 fdskjao fdjksaol"]
  N2 -> N3 [id="e2" label="e2"]
  N3 -> N4 [id="e3" label="e3"]
  N3 -> N1 [id="e4" label="e4"]
  }`;

export const jsonGraphWithGraphvizPositions = {
  name: '%27',
  directed: true,
  strict: false,
  _draw_: [
    {
      op: 'c',
      grad: 'none',
      color: '#fffffe00',
    },
    {
      op: 'C',
      grad: 'none',
      color: '#ffffff',
    },
    {
      op: 'P',
      points: [
        [0.0, 0.0],
        [0.0, 180.0],
        [59.0, 180.0],
        [59.0, 0.0],
      ],
    },
  ],
  bb: '0,0,59,180',
  xdotversion: '1.7',
  _subgraph_cnt: 0,
  objects: [
    {
      _gvid: 0,
      name: 'n0',
      Label: 'A node',
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#ffffff',
        },
        {
          op: 'E',
          rect: [41.0, 162.0, 18.0, 18.0],
        },
      ],
      _ldraw_: [
        {
          op: 'F',
          size: 14.0,
          face: 'Times-Roman',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#0000ff',
        },
        {
          op: 'T',
          pt: [41.0, 157.8],
          align: 'c',
          width: 14.0,
          text: 'n0',
        },
      ],
      fillcolor: 'white',
      fixedsize: 'true',
      fontcolor: 'blue',
      height: '0.5',
      label: '\\N',
      margin: '0',
      pos: '41,162',
      shape: 'circle',
      style: 'filled',
      width: '0.5',
    },
    {
      _gvid: 1,
      name: 'n1',
      Label: 'Another node',
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#ffffff',
        },
        {
          op: 'E',
          rect: [18.0, 90.0, 18.0, 18.0],
        },
      ],
      _ldraw_: [
        {
          op: 'F',
          size: 14.0,
          face: 'Times-Roman',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#0000ff',
        },
        {
          op: 'T',
          pt: [18.0, 85.8],
          align: 'c',
          width: 14.0,
          text: 'n1',
        },
      ],
      fillcolor: 'white',
      fixedsize: 'true',
      fontcolor: 'blue',
      height: '0.5',
      label: '\\N',
      margin: '0',
      pos: '18,90',
      shape: 'circle',
      style: 'filled',
      width: '0.5',
    },
    {
      _gvid: 2,
      name: 'n2',
      Label: 'And a last one',
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#ffffff',
        },
        {
          op: 'E',
          rect: [41.0, 18.0, 18.0, 18.0],
        },
      ],
      _ldraw_: [
        {
          op: 'F',
          size: 14.0,
          face: 'Times-Roman',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#0000ff',
        },
        {
          op: 'T',
          pt: [41.0, 13.8],
          align: 'c',
          width: 14.0,
          text: 'n2',
        },
      ],
      fillcolor: 'white',
      fixedsize: 'true',
      fontcolor: 'blue',
      height: '0.5',
      label: '\\N',
      margin: '0',
      pos: '41,18',
      shape: 'circle',
      style: 'filled',
      width: '0.5',
    },
  ],
  edges: [
    {
      _gvid: 0,
      tail: 0,
      head: 1,
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'b',
          points: [
            [35.67, 144.76],
            [32.98, 136.58],
            [29.65, 126.45],
            [26.61, 117.2],
          ],
        },
      ],
      _hdraw_: [
        {
          op: 'S',
          style: 'solid',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'P',
          points: [
            [29.91, 116.04],
            [23.47, 107.63],
            [23.26, 118.23],
          ],
        },
      ],
      pos: 'e,23.465,107.63 35.666,144.76 32.976,136.58 29.647,126.45 26.607,117.2',
    },
    {
      _gvid: 1,
      tail: 1,
      head: 2,
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'b',
          points: [
            [23.33, 72.76],
            [26.02, 64.58],
            [29.35, 54.45],
            [32.39, 45.2],
          ],
        },
      ],
      _hdraw_: [
        {
          op: 'S',
          style: 'solid',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'P',
          points: [
            [35.74, 46.23],
            [35.53, 35.63],
            [29.09, 44.04],
          ],
        },
      ],
      pos: 'e,35.535,35.633 23.334,72.765 26.024,64.578 29.353,54.448 32.393,45.195',
    },
    {
      _gvid: 2,
      tail: 2,
      head: 0,
      _draw_: [
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'b',
          points: [
            [42.62, 36.17],
            [43.52, 46.53],
            [44.55, 60.01],
            [45.0, 72.0],
            [45.6, 87.99],
            [45.6, 92.01],
            [45.0, 108.0],
            [44.69, 116.33],
            [44.09, 125.39],
            [43.46, 133.62],
          ],
        },
      ],
      _hdraw_: [
        {
          op: 'S',
          style: 'solid',
        },
        {
          op: 'c',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'C',
          grad: 'none',
          color: '#000000',
        },
        {
          op: 'P',
          points: [
            [39.95, 133.58],
            [42.62, 143.83],
            [46.93, 134.15],
          ],
        },
      ],
      pos: 'e,42.625,143.83 42.625,36.167 43.524,46.533 44.548,60.013 45,72 45.602,87.989 45.602,92.011 45,108 44.686,116.33 44.095,125.39 43.462,133.62',
    },
  ],
};
