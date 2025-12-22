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

import {RpcError} from '@protobuf-ts/runtime-rpc';

import {FlamegraphArrow, QueryServiceClient, Source, TableArrow} from '@parca/client';

import {ProfileSource} from '../../ProfileSource';

export interface FlamegraphData {
  loading: boolean;
  arrow?: FlamegraphArrow;
  total?: bigint;
  filtered?: bigint;
  error: RpcError | null;
  metadataMappingFiles?: string[];
  metadataLoading: boolean;
  metadataLabels?: string[];
  metadataRefetch?: () => Promise<void>;
}

export interface TopTableData {
  loading: boolean;
  arrow?: TableArrow;
  total?: bigint;
  filtered?: bigint;
  error: RpcError | null;
  unit?: string;
}

export interface SourceData {
  loading: boolean;
  data?: Source;
  error: RpcError | null;
}

export interface SandwichData {
  callees: FlamegraphData;
  callers: FlamegraphData;
}

export type VisualizationType =
  | 'flamegraph'
  | 'callgraph'
  | 'table'
  | 'source'
  | 'flamechart'
  | 'sandwich';

export interface ProfileViewProps {
  total: bigint;
  filtered: bigint;
  flamegraphData: FlamegraphData;
  flamechartData: FlamegraphData;
  sandwichData: SandwichData;
  topTableData?: TopTableData;
  sourceData?: SourceData;
  profileSource: ProfileSource;
  queryClient?: QueryServiceClient;
  compare?: boolean;
  onDownloadPProf: () => void;
  pprofDownloading?: boolean;
  showVisualizationSelector?: boolean;
}
