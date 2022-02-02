import React, {useState, useEffect} from 'react';
import {
  Targets,
  TargetsRequest,
  TargetsResponse,
  ScrapeServiceClient,
  ServiceError,
} from '@parca/client';
import {NextRouter, withRouter} from 'next/router';
import TargetsTable from '../components/Targets/TargetsTable';

const apiEndpoint = process.env.NEXT_PUBLIC_API_ENDPOINT;

interface TargetsPageProps {
  router: NextRouter;
}

export interface ITargetsResult {
  response: TargetsResponse.AsObject | null;
  error: ServiceError | null;
}

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
  }, []);

  return result;
};

const TargetsPage = (_: TargetsPageProps): JSX.Element => {
  const scrapeClient = new ScrapeServiceClient(apiEndpoint === undefined ? '' : apiEndpoint);
  const {response: targetsResponse, error: targetsError} = useTargets(scrapeClient);
  const getKeyValuePairFromArray = (key: string, value: {targetsList}) => {
    return {[key]: value.targetsList};
  };

  const {targetsMap} = targetsResponse || {};
  const targetNamespaces = targetsMap?.map(item =>
    getKeyValuePairFromArray(item[0] as string, item[1] as {targetsList})
  );

  return (
    <div className="flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
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
        </div>
      </div>
    </div>
  );
};

export default withRouter(TargetsPage);
