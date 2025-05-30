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
  Callgraph as CallgraphType,
  Flamegraph,
  FlamegraphArrow,
  QueryServiceClient,
  Source,
  TableArrow,
} from '@parca/client';

import {ProfileSource} from '../../ProfileSource';

export interface FlamegraphData {
  loading: boolean;
  data?: Flamegraph;
  arrow?: FlamegraphArrow;
  total?: bigint;
  filtered?: bigint;
  error?: any;
  metadataMappingFiles?: string[];
  metadataLoading: boolean;
  metadataLabels?: string[];
}

export interface TopTableData {
  loading: boolean;
  arrow?: TableArrow;
  total?: bigint;
  filtered?: bigint;
  error?: any;
  unit?: string;
}

export interface CallgraphData {
  loading: boolean;
  data?: CallgraphType;
  total?: bigint;
  filtered?: bigint;
  error?: any;
}

export interface SourceData {
  loading: boolean;
  data?: Source;
  error?: any;
}

export type VisualizationType = 'icicle' | 'callgraph' | 'table' | 'source' | 'iciclechart';

export interface ProfileViewProps {
  total: bigint;
  filtered: bigint;
  flamegraphData: FlamegraphData;
  flamechartData: FlamegraphData;
  topTableData?: TopTableData;
  callgraphData?: CallgraphData;
  sourceData?: SourceData;
  profileSource: ProfileSource;
  queryClient?: QueryServiceClient;
  compare?: boolean;
  onDownloadPProf: () => void;
  pprofDownloading?: boolean;
  showVisualizationSelector?: boolean;
}
