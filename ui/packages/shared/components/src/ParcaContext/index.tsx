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

import {FC, ProfilerOnRenderCallback, ReactNode, createContext, useContext} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {QueryServiceClient} from '@parca/client';
import type {ColorConfig, NavigateFunction} from '@parca/utilities';

import {DateTimeRange} from '../DateTimeRangePicker/utils';
import {NoDataPrompt} from '../NoDataPrompt';
import Spinner, {SpinnerProps} from '../Spinner';

export interface ProfileData {
  line: number;
  cumulative: number;
  flat: number;
}

export interface SourceViewContextMenuItem {
  id: string;
  label: string;
  action: (selectedCode: string, profileData: ProfileData[]) => void;
}

export interface AdditionalMetricsGraphProps {
  querySelection: {
    expression: string;
    from: number;
    to: number;
    timeSelection: string;
    sumBy?: string[];
    mergeFrom?: string;
    mergeTo?: string;
  };
  queryClient: QueryServiceClient;
  suffix?: '_a' | '_b';
  timeRange: DateTimeRange;
  onTimeRangeChange: (range: DateTimeRange) => void;
  commitTimeRange: () => void;
  selectTimeRange: (range: DateTimeRange) => void;
}

interface Props {
  Spinner: FC<SpinnerProps>;
  loader?: ReactNode;
  noDataPrompt?: ReactNode;
  profileExplorer?: {
    PaddingX: number;
    metricsGraph: {
      height: number;
      maxHeightStyle: {
        default: string;
        compareMode: string;
      };
    };
  };
  perf?: {
    onRender?: ProfilerOnRenderCallback;
    markInteraction: (interactionName: string, sampleCount: number | string | bigint) => void;
    setMeasurement?: (name: string, value: number) => void;
    captureMessage?: (message: string, level?: 'info' | 'warning' | 'error') => void;
  };
  onError?: (error: RpcError) => void;
  queryServiceClient: QueryServiceClient;
  // Function to navigate to a new URL with query parameters. This function should handle URL encoding of parameters internally.
  navigateTo: NavigateFunction;
  enableSourcesView?: boolean;
  enableSandwichView?: boolean;
  enableFlamechartView?: boolean;
  authenticationErrorMessage?: string;
  isDarkMode: boolean;
  flamegraphHint?: ReactNode;
  viewComponent?: {
    emitQuery: (query: string) => void;
    createViewComponent?: ReactNode;
    disableProfileTypesDropdown?: boolean;
    labelnames?: string[];
    disableExplorativeQuerying?: boolean;
    profileFilterDefaults?: unknown[];
  };
  profileViewExternalMainActions?: ReactNode;
  profileViewExternalSubActions?: ReactNode;
  sourceViewContextMenuItems?: SourceViewContextMenuItem[];
  additionalFlamegraphColorProfiles?: Record<string, ColorConfig>;
  timezone?: string;
  preferencesModal?: boolean;
  checkDebuginfoStatusHandler?: (buildId: string) => void;
  flamechartHelpText?: ReactNode;
  additionalMetricsGraph?: (props: AdditionalMetricsGraphProps) => ReactNode;
}

export const defaultValue: Props = {
  loader: <Spinner />,
  Spinner,
  noDataPrompt: <NoDataPrompt />,
  profileExplorer: {
    PaddingX: 32,
    metricsGraph: {
      height: 402,
      maxHeightStyle: {
        default: 'calc(47vw - 24px)',
        compareMode: 'calc(23.5vw - 24px)',
      },
    },
  },
  perf: {
    onRender: () => {},
    markInteraction: () => {},
    setMeasurement: () => {},
    captureMessage: () => {},
  },
  queryServiceClient: {} as unknown as QueryServiceClient,
  navigateTo: () => {},
  enableSourcesView: false,
  enableSandwichView: false,
  isDarkMode: false,
  preferencesModal: false,
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
