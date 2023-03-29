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

import {ProfilerOnRenderCallback, ReactNode, createContext, useContext} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {NoDataPrompt} from '../NoDataPrompt';
import Spinner from '../Spinner';

interface Props {
  loader: ReactNode;
  noDataPrompt: ReactNode;
  perf?: {
    onRender?: ProfilerOnRenderCallback;
    markInteraction: (interactionName: string, sampleCount: number | string | bigint) => void;
  };
  onError?: (error: RpcError, originatingFeature: string) => void;
}

export const defaultValue: Props = {
  loader: <Spinner />,
  noDataPrompt: <NoDataPrompt />,
  perf: {
    onRender: () => {},
    markInteraction: () => {},
  },
};

const ParcaContext = createContext<Props>(defaultValue);

export const ParcaContextProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: Props;
}): JSX.Element => {
  return <ParcaContext.Provider value={value ?? defaultValue}>{children}</ParcaContext.Provider>;
};

export const useParcaContext = (): Props => {
  const context = useContext(ParcaContext);
  if (context == null) {
    return defaultValue;
  }
  return context;
};

export default ParcaContext;
