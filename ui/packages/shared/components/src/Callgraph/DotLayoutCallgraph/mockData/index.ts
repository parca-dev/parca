export const dotGraph = `{
 n1 [Label = "n1"];
 n2 [Label = "n2"];
 n3 [Label = "n3"];
 n1 -- n2;
 n1 -- n3;
 n2 -- n2;
}`;

export const jsonGraph = {
  total: '4358676',
  unit: 'count',
  nodes: [
    {
      id: 'n0',
      label: 'A node',
      value: 30,
    },
    {
      id: 'n1',
      label: 'Another node',
      value: 20,
    },
    {
      id: 'n2',
      label: 'And a last one',
      value: 10,
    },
    {
      id: 'n3',
      label: 'n3',
      value: 2,
    },
    {
      id: 'n4',
      label: 'n4',
      value: 1,
    },
    {
      id: 'n5',
      label: 'n5',
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
      label: 'edge 3',
    },
    {
      id: 'e3',
      source: 'n0',
      target: 'n2',
      label: 'edge 4',
    },
    {
      id: 'e4',
      source: 'n2',
      target: 'n3',
      label: 'edge 5',
    },
    {
      id: 'e5',
      source: 'n3',
      target: 'n4',
      label: 'edge 6',
    },
    {
      id: 'e5',
      source: 'n4',
      target: 'n5',
      label: 'edge 6',
    },
    {
      id: 'e5',
      source: 'n3',
      target: 'n5',
      label: 'edge 6',
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

export const node_with_meta_data = (id, name) => ({
  id,
  cumulative: Math.floor(Math.random() * 10),
  diff: '0',
  meta: {
    location: {
      id: {
        '0': 201,
        '1': 124,
        '2': 147,
        '3': 175,
        '4': 103,
        '5': 42,
        '6': 76,
        '7': 211,
        '8': 186,
        '9': 184,
        '10': 7,
        '11': 19,
        '12': 62,
        '13': 204,
        '14': 76,
        '15': 130,
      },
      address: `${Math.floor(Math.random() * 10)}${Math.floor(Math.random() * 10)}${Math.floor(
        Math.random() * 10
      )}`,
      mappingId: {
        '0': 72,
        '1': 191,
        '2': 155,
        '3': 45,
        '4': 34,
        '5': 49,
        '6': 65,
        '7': 169,
        '8': 170,
        '9': 93,
        '10': 28,
        '11': 111,
        '12': 102,
        '13': 196,
        '14': 92,
        '15': 210,
      },
      isFolded: false,
    },
    mapping: {
      id: {
        '0': 72,
        '1': 191,
        '2': 155,
        '3': 45,
        '4': 34,
        '5': 49,
        '6': 65,
        '7': 169,
        '8': 170,
        '9': 93,
        '10': 28,
        '11': 111,
        '12': 102,
        '13': 196,
        '14': 92,
        '15': 210,
      },
      start: '0',
      limit: '0',
      offset: '0',
      file: '',
      buildId: '',
      hasFunctions: true,
      hasFilenames: false,
      hasLineNumbers: false,
      hasInlineFrames: false,
    },
    function: {
      id: {
        '0': 46,
        '1': 90,
        '2': 8,
        '3': 65,
        '4': 135,
        '5': 181,
        '6': 65,
        '7': 61,
        '8': 148,
        '9': 246,
        '10': 90,
        '11': 244,
        '12': 208,
        '13': 193,
        '14': 40,
        '15': 177,
      },
      startLine: '0',
      name: name,
      systemName: 'runtime.gopark',
      filename: `/opt/homebrew/Cellar/go/1.18.2/libexec/src/runtime/proc.go/${name}`,
    },
    line: {
      functionId: {
        '0': 46,
        '1': 90,
        '2': 8,
        '3': 65,
        '4': 135,
        '5': 181,
        '6': 65,
        '7': 61,
        '8': 148,
        '9': 246,
        '10': 90,
        '11': 244,
        '12': 208,
        '13': 193,
        '14': 40,
        '15': 177,
      },
      line: '361',
    },
  },
});

export const jsonGraphWithMetaData = {
  total: '4358676',
  unit: 'count',
  nodes: [
    node_with_meta_data('root', 'root node'),
    node_with_meta_data('n1', 'normal node'),
    node_with_meta_data('n2', 'second node'),
    node_with_meta_data('n3', 'child'),
  ],
  edges: [
    {
      id: 'edge1',
      source: 'root',
      target: 'n1',
      cumulative: '10',
    },
    {
      id: 'edge2',
      source: 'root',
      target: 'n2',
      cumulative: '8',
    },
    {
      id: 'edge2',
      source: 'n1',
      target: 'n3',
      cumulative: '4',
    },
  ],
};
