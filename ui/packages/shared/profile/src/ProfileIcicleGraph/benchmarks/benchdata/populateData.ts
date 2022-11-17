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

import {GrpcWebFetchTransport} from '@protobuf-ts/grpcweb-transport';
import * as client from '@parca/client';
import {QueryServiceClient} from '@parca/client';
import fs from 'fs-extra';
import {fileURLToPath} from 'url';
import fetch, {Headers} from 'node-fetch';
import path from 'path';

globalThis.fetch = fetch as any;
globalThis.Headers = Headers as any;
const DIR_NAME = path.dirname(fileURLToPath(import.meta.url));

const apiEndpoint = 'https://demo.parca.dev';

const queryClient = new QueryServiceClient(
  new GrpcWebFetchTransport({
    baseUrl: `${apiEndpoint}/api`,
  })
);

const populateDataIfNeeded = async (from: Date, filename: string): Promise<void> => {
  const filePath = path.join(DIR_NAME, filename);
  if (Object.keys(await readFile(filePath)).length > 0) {
    return;
  }
  const {response} = await queryClient.query({
    options: {
      oneofKind: 'merge',
      merge: {
        start: client.Timestamp.fromDate(from),
        end: client.Timestamp.fromDate(new Date()),
        query: 'parca_agent_cpu:samples:count:cpu:nanoseconds:delta{container="parca"}',
      },
    },
    reportType: client.QueryRequest_ReportType.FLAMEGRAPH_TABLE,
    mode: client.QueryRequest_Mode.MERGE,
  });
  if (response.report.oneofKind !== 'flamegraph') {
    throw new Error('Expected flamegraph report');
  }
  await writeToFile(response.report.flamegraph, filePath);
};

const writeToFile = async (data: client.Flamegraph, filename: string): Promise<void> => {
  return await fs.writeFile(filename, JSON.stringify(data));
};

const readFile = async (filename: string): Promise<JSON> => {
  return await fs.readJSON(filename);
};

const run = async (): Promise<void> => {
  await Promise.all([
    populateDataIfNeeded(new Date(new Date().getTime() - 1000 * 60), 'parca-1m.json'),
    populateDataIfNeeded(new Date(new Date().getTime() - 1000 * 60 * 10), 'parca-10m.json'),
    populateDataIfNeeded(new Date(new Date().getTime() - 1000 * 60 * 20), 'parca-20m.json'),
  ]);
};

export default run;
