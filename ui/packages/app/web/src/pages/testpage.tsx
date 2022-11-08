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

import {
  ParcaContextProvider,
  Button,
  useParcaContext,
  parcaContextDefaultValue,
} from '@parca/components';

import {useReducer, ProfilerOnRenderCallback, Profiler, useState} from 'react';

const ProfilerComponent = () => {
  const [count, setCount] = useState(0);

  return (
    <>
      {count}
      <p>Profiler here</p>
      <Button
        onClick={() => {
          setCount(count + 10);
        }}
      >
        Increment
      </Button>
    </>
  );
};

const ChildComponent = () => {
  const {perf} = useParcaContext();

  return (
    <>
      <Profiler id="ProfilerComponent" onRender={perf.onRender}>
        <ProfilerComponent />
      </Profiler>
    </>
  );
};

const Testpage = () => {
  const logRender: ProfilerOnRenderCallback = (
    id,
    phase,
    actualDuration,
    baseDuration,
    _startTime,
    _commitTime,
    _interactions
  ) => {
    console.log('Rendered', id, phase, 'act:', actualDuration, 'ms', ' base:', baseDuration, 'ms');
  };

  return (
    <ParcaContextProvider
      value={{
        loader: <p>loading...</p>,
        perf: {
          onRender: logRender,
        },
      }}
    >
      <ChildComponent />
    </ParcaContextProvider>
  );
};

export default Testpage;
