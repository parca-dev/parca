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

import React from 'react';
import RowBasedFlamegraph from '../RowBasedFlamegraph';
import {Provider} from 'react-redux';
import {store} from '@parca/store';
import parca20mGraphData from './benchdata/parca-20m.json';
import {Flamegraph, FlamegraphNode} from '@parca/client';
import {nodeLabel} from '../..';

const {store: reduxStore} = store();

const parca20mGraph = parca20mGraphData as Flamegraph;

console.log('parca20mGraph', parca20mGraph);

const mapChildren = (children: FlamegraphNode[], data: Flamegraph): any => {
  return children.map(node => {
    const name = nodeLabel(node, data.stringTable, data.mapping, data.locations, data.function);
    return {
      name,
      value: node.cumulative,
      children: mapChildren(node.children, data),
    };
  });
};

const rowBasedData = {
  name: 'root',
  value: parca20mGraph.root?.cumulative,
  children: mapChildren(parca20mGraph.root?.children ?? [], parca20mGraph),
};

export default function ({callback = () => {}}): React.ReactElement {
  return (
    <div ref={callback}>
      <Provider store={reduxStore}>
        <RowBasedFlamegraph data={rowBasedData} />
      </Provider>
    </div>
  );
}
