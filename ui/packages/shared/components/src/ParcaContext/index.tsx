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

import {QueryServiceClient} from '@parca/client';
import type {NavigateFunction} from '@parca/utilities';

import {NoDataPrompt} from '../NoDataPrompt';
import Spinner from '../Spinner';

interface SourceViewContextMenuItem {
  id: string;
  label: string;
  action: (selectedCode) => void;
}

interface Props {
  loader?: ReactNode;
  noDataPrompt?: ReactNode;
  profileExplorer?: {
    PaddingX: number;
    metricsGraph: {
      maxHeightStyle: {
        default: string;
        compareMode: string;
      };
    };
  };
  perf?: {
    onRender?: ProfilerOnRenderCallback;
    markInteraction: (interactionName: string, sampleCount: number | string | bigint) => void;
  };
  onError?: (error: RpcError) => void;
  queryServiceClient: QueryServiceClient;
  navigateTo: NavigateFunction;
  enableSourcesView?: boolean;
  authenticationErrorMessage?: string;
  isDarkMode: boolean;
  flamegraphHint?: ReactNode;
  viewComponent?: {
    emitQuery: (query: string) => void;
    createViewComponent?: ReactNode;
  };
  profileViewExternalMainActions?: ReactNode;
  profileViewExternalSubActions?: ReactNode;
  sourceViewContextMenuItems?: SourceViewContextMenuItem[];
}

export const defaultValue: Props = {
  loader: <Spinner />,
  noDataPrompt: <NoDataPrompt />,
  profileExplorer: {
    PaddingX: 58,
    metricsGraph: {
      maxHeightStyle: {
        default: 'calc(47vw - 24px)',
        compareMode: 'calc(23.5vw - 24px)',
      },
    },
  },
  perf: {
    onRender: () => {},
    markInteraction: () => {},
  },
  queryServiceClient: {} as unknown as QueryServiceClient,
  navigateTo: () => {},
  enableSourcesView: false,
  isDarkMode: false,
};

const ParcaContext = createContext<Props>(defaultValue);

export const ParcaContextProvider = ({
  children,
  value,
}: {
  children: ReactNode;
  value?: Props;
}): JSX.Element => {
  return (
    <ParcaContext.Provider value={{...defaultValue, ...(value ?? {})}}>
      {children}
    </ParcaContext.Provider>
  );
};

export const useParcaContext = (): Props => {
  const context = useContext(ParcaContext);
  if (context == null) {
    return defaultValue;
  }
  return context;
};

export default ParcaContext;
