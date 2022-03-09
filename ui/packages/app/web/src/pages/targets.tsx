import React, {useState, useEffect} from 'react';
import {TargetsRequest, TargetsResponse, ScrapeServiceClient, ServiceError} from '@parca/client';
import {EmptyState} from '@parca/components';
import TargetsTable from '../components/Targets/TargetsTable';

export interface ITargetsResult {
  response: TargetsResponse.AsObject | null;
  error: ServiceError | null;
}

const apiEndpoint = process.env.REACT_APP_PUBLIC_API_ENDPOINT;

export const useTargets = (client: ScrapeServiceClient): ITargetsResult => {
  const [result, setResult] = useState<ITargetsResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    client.targets(
      new TargetsRequest(),
      (error: ServiceError | null, responseMessage: TargetsResponse | null) => {
        const res = responseMessage == null ? null : responseMessage.toObject();

        setResult({
          response: res,
          error: error,
        });
      }
    );
    /* eslint-disable react-hooks/exhaustive-deps */
  }, []);

  return result;
};

const TargetsPage = (): JSX.Element => {
  const scrapeClient = new ScrapeServiceClient(
    apiEndpoint === undefined ? '/api' : `${apiEndpoint}/api`
  );
  const {response: targetsResponse} = useTargets(scrapeClient);
  const getKeyValuePairFromArray = (key: string, value: {targetsList}) => {
    return {[key]: value.targetsList};
  };

  const {targetsMap} = targetsResponse ?? {};
  const targetNamespaces = targetsMap?.map(item =>
    getKeyValuePairFromArray(item[0], item[1] as {targetsList})
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
              {targetNamespaces?.map(namespace => {
                const name = Object.keys(namespace)[0];
                const targets = namespace[name];
                return (
                  <div key={name} className="p-2 border-b-2">
                    <div>{name}</div>
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
