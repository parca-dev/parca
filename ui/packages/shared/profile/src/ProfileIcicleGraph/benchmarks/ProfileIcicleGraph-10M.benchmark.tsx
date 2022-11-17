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
import ProfileIcicleGraph from '..';
import {Provider} from 'react-redux';
import {store} from '@parca/store';
import parca10mGraphData from './benchdata/parca-10m.json';
import {Flamegraph} from '@parca/client';

const {store: reduxStore} = store();

const parca10mGraph = parca10mGraphData as Flamegraph;

export default function ({callback = () => {}}): React.ReactElement {
  return (
    <div ref={callback}>
      <Provider store={reduxStore}>
        <ProfileIcicleGraph
          graph={parca10mGraph}
          sampleUnit={parca10mGraph.unit}
          curPath={[]}
          setNewCurPath={() => {}}
        />
      </Provider>
    </div>
  );
}
