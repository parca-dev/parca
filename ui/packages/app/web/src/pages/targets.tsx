import React, {useEffect, useState} from 'react';
import {ScrapeServiceClient, Target, TargetsResponse, TargetsRequest_State} from '@parca/client';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {EmptyState} from '@parca/components';
import TargetsTable from '../components/Targets/TargetsTable';
import {GrpcWebFetchTransport} from '@protobuf-ts/grpcweb-transport';

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

export interface ITargetsResult {
  response: TargetsResponse | null;
  error: RpcError | null;
}

export const useTargets = (client: ScrapeServiceClient): ITargetsResult => {
  const [result, setResult] = useState<ITargetsResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    const call = client.targets({
      state: TargetsRequest_State.ANY_UNSPECIFIED,
    });

    call.response
      .then(response => setResult({response, error: null}))
      .catch(error => setResult({error, response: null}));
  }, [client]);

  return result;
};

const scrapeClient = new ScrapeServiceClient(
  new GrpcWebFetchTransport({
    baseUrl: apiEndpoint === undefined ? '/api' : `${apiEndpoint}/api`,
  })
);

const sortTargets = (targets: {[x: string]: any}[]) =>
  targets.sort((a, b) => {
    return Object.keys(a)[0].localeCompare(Object.keys(b)[0]);
  });

const TargetsPage = (): JSX.Element => {
  const {response: targetsResponse, error} = useTargets(scrapeClient);

  if (error !== null) {
    return <div>Error</div>;
  }

  const getKeyValuePairFromArray = (key: string, value: {targets}) => {
    return {[key]: value.targets};
  };

  const {targets} = targetsResponse ?? {};
  const targetNamespaces = Object.entries(targets ?? {}).map(item =>
    getKeyValuePairFromArray(item[0], item[1] as {targets})
  );

  return (
    <div className="flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
          <EmptyState
            isEmpty={targetNamespaces?.length <= 0}
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
            <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
              {sortTargets(targetNamespaces)?.map(namespace => {
                const name = Object.keys(namespace)[0];
                const targets = namespace[name].sort((a: Target, b: Target) => {
                  return a.url.localeCompare(b.url);
                });
                return (
                  <div key={name} className="my-2 p-2 border-b-2">
                    <div className="my-2">
                      <span className="font-semibold text-xl">{name}</span>
                    </div>
                    <TargetsTable targets={targets} />
                  </div>
                );
              })}
            </div>
          </EmptyState>
        </div>
      </div>
    </div>
  );
};

export default TargetsPage;
