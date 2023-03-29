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

import React, {useEffect, useState} from 'react';

import {PromiseClient, createPromiseClient} from '@bufbuild/connect';
import {createConnectTransport} from '@bufbuild/connect-web';
import {RpcError} from '@protobuf-ts/runtime-rpc';

import {
  Agent,
  AgentsResponse,
  AgentsService,
  ScrapeService,
  Target,
  Targets,
  TargetsResponse,
} from '@parca/client/src/connect';
import {EmptyState} from '@parca/components';

import AgentsTable from '../components/Targets/AgentsTable';
import TargetsTable from '../components/Targets/TargetsTable';

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

export interface ITargetsResult {
  response: TargetsResponse | null;
  error: RpcError | null;
}

export const useTargets = (client: PromiseClient<typeof ScrapeService>): ITargetsResult => {
  const [result, setResult] = useState<ITargetsResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    client
      .targets({})
      .then(response => setResult({response, error: null}))
      .catch(error => setResult({error, response: null}));
  }, [client]);

  return result;
};

const scrapeClient = createPromiseClient(
  ScrapeService,
  createConnectTransport({
    baseUrl: apiEndpoint === undefined ? `${window.PATH_PREFIX}/api` : `${apiEndpoint}/api`,
  })
);

const sortTargets = (targets: {[x: string]: any}[]) =>
  targets.sort((a, b) => {
    return Object.keys(a)[0].localeCompare(Object.keys(b)[0]);
  });

export interface IAgentsResult {
  response: AgentsResponse | null;
  error: RpcError | null;
}

export const useAgents = (client: PromiseClient<typeof AgentsService>): IAgentsResult => {
  const [result, setResult] = useState<IAgentsResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    client
      .agents({})
      .then(response => setResult({response, error: null}))
      .catch(error => setResult({error, response: null}));
  }, [client]);

  return result;
};

const agentsClient = createPromiseClient(
  AgentsService,
  createConnectTransport({
    baseUrl: apiEndpoint === undefined ? `${window.PATH_PREFIX}/api` : `${apiEndpoint}/api`,
  })
);

const sortAgents = (agents: Agent[]) =>
  agents.sort((a: Agent, b: Agent) => a.id.localeCompare(b.id));

const TargetsPage = (): JSX.Element => {
  const {response: targetsResponse, error: targetsError} = useTargets(scrapeClient);
  const {response: agentsResponse, error: agentsError} = useAgents(agentsClient);

  if (targetsError !== null) {
    return <div>Targets Error: {targetsError.toString()}</div>;
  }
  if (agentsError !== null && agentsError.code !== 'UNIMPLEMENTED') {
    return <div>Agents Error: {agentsError.toString()}</div>;
  }

  const getKeyValuePairFromArray = (key: string, value: Targets) => {
    return {[key]: value.targets};
  };

  const {targets} = targetsResponse ?? {};
  const targetNamespaces = Object.entries(targets ?? {}).map(item =>
    getKeyValuePairFromArray(item[0], item[1])
  );

  const agents = agentsResponse?.agents ?? [];

  return (
    <div className="flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="inline-block min-w-full py-2 align-middle sm:px-6 lg:px-8">
          <EmptyState
            isEmpty={targetNamespaces?.length <= 0 && agents.length <= 0}
            title="No targets available"
            body={
              <p>
                For additional information see the{' '}
                <a
                  className="text-blue-500"
                  href="https://www.parca.dev/docs/parca-agent-design#target-discovery"
                >
                  Target Discovery
                </a>{' '}
                documentation
              </p>
            }
          >
            <>
              {agents.length > 0 ? (
                <div className="overflow-hidden border-b border-gray-200 shadow sm:rounded-lg">
                  <div className="my-2 border-b-2 p-2">
                    <div className="my-2">
                      <span className="text-xl font-semibold">Parca Agents</span>
                    </div>
                    <AgentsTable agents={sortAgents(agents)} />
                  </div>
                </div>
              ) : (
                <></>
              )}
              {Object.keys(targets ?? {}).length > 0 ? (
                <div className="overflow-hidden border-b border-gray-200 shadow sm:rounded-lg">
                  {sortTargets(targetNamespaces)?.map(namespace => {
                    const name = Object.keys(namespace)[0];
                    const targets = namespace[name].sort((a: Target, b: Target) => {
                      return a.url.localeCompare(b.url);
                    });
                    return (
                      <div key={name} className="my-2 border-b-2 p-2">
                        <div className="my-2">
                          <span className="text-xl font-semibold">{name}</span>
                        </div>
                        <TargetsTable targets={targets} />
                      </div>
                    );
                  })}
                </div>
              ) : (
                <></>
              )}
            </>
          </EmptyState>
        </div>
      </div>
    </div>
  );
};

export default TargetsPage;
