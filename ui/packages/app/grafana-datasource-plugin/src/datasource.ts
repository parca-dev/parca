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

import defaults from 'lodash/defaults';

import {
  DataQueryRequest,
  DataQueryResponse,
  DataSourceApi,
  DataSourceInstanceSettings,
  MutableDataFrame,
  FieldType,
} from '@grafana/data';

import { ParcaQuery, ParcaDataSourceOptions, defaultQuery } from './types';
import { downloadPprof, GrafanaParcaData, MergedProfileSource } from '@parca/profile';
import { GrpcWebFetchTransport } from '@protobuf-ts/grpcweb-transport';
import { QueryRequest_ReportType, QueryServiceClient } from '@parca/client';
import { saveAsBlob } from '@parca/functions';

export class DataSource extends DataSourceApi<ParcaQuery, ParcaDataSourceOptions> {
  instanceSettings: DataSourceInstanceSettings<ParcaDataSourceOptions>;
  queryClient: QueryServiceClient;

  constructor(instanceSettings: DataSourceInstanceSettings<ParcaDataSourceOptions>) {
    super(instanceSettings);
    this.instanceSettings = instanceSettings;
    if (instanceSettings.jsonData.APIEndpoint == null) {
      throw new Error('APIEndpoint is not set');
    }
    this.queryClient = new QueryServiceClient(
      new GrpcWebFetchTransport({
        baseUrl: `${instanceSettings.jsonData.APIEndpoint}`,
      })
    );
  }

  async query(options: DataQueryRequest<ParcaQuery>): Promise<DataQueryResponse> {
    const { range } = options;
    const from = range.from.valueOf();
    const to = range.to.valueOf();

    // Return a constant for each query.
    const data = await Promise.all(
      options.targets.map(async (target) => {
        const query = defaults(target, defaultQuery);

        const frame = new MutableDataFrame({
          refId: query.refId,
          fields: [{ name: 'data', type: FieldType.other }],
        });
        frame.appendRow([await this.getData(from, to, query)]);
        return frame;
      })
    );

    return { data };
  }

  async getData(from: number, to: number, query: ParcaQuery): Promise<GrafanaParcaData> {
    const profileSource = new MergedProfileSource(from, to, query.parcaQuery);
    const flamegraphReq = profileSource.QueryRequest();
    flamegraphReq.reportType = QueryRequest_ReportType.FLAMEGRAPH_UNSPECIFIED;
    const topTableReq = profileSource.QueryRequest();
    topTableReq.reportType = QueryRequest_ReportType.TOP;

    try {
      const [
        {
          response: { report: flamegraphReport },
        },
        {
          response: { report: topTableReport },
        },
      ] = await Promise.all([this.queryClient.query(flamegraphReq), this.queryClient.query(topTableReq)]);

      return {
        flamegraphData: {
          loading: false,
          data: flamegraphReport.oneofKind === 'flamegraph' ? flamegraphReport.flamegraph : undefined,
        },
        topTableData: {
          loading: false,
          data: topTableReport.oneofKind === 'top' ? topTableReport.top : undefined,
        },
        actions: {
          downloadPprof: () => {
            void (async () => {
              const blob = await downloadPprof(profileSource.QueryRequest(), this.queryClient, {});
              saveAsBlob(blob, 'profile.pb.gz');
            })();
          },
          getQueryClient: () => this.queryClient,
        },
      };
    } catch (err) {
      return { error: new Error(JSON.stringify(err)) };
    }
  }

  async testDatasource(): Promise<{ status: string; message?: string }> {
    // Implement a health check for your data source.
    if (this.instanceSettings.jsonData.APIEndpoint == null) {
      return { status: 'error', message: 'APIEndpoint is not set' };
    }
    // const healthcheckURL = `${this.instanceSettings.jsonData.APIEndpoint}/grpc-health-probe`;
    return {
      status: 'success',
      message: 'Success',
    };
  }
}
