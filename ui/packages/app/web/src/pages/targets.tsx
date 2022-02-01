import React, {useState, useEffect} from 'react';
import {
  Targets,
  TargetsRequest,
  TargetsResponse,
  ScrapeServiceClient,
  ServiceError,
} from '@parca/client';
import {NextRouter, withRouter} from 'next/router';
import TargetsTable from '../components/TargetsTable';

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

  // TODO remove the mock data below
  // const targetsMap = [['first_list', {targetsList: [{ health: 1}, {health: 2}, {health: 3}]}], ['second_list', {targetsList: [{ health: 4}, {health: 5}, {health: 6}]}]]
  const {targetsMap} = targetsResponse || {};
  const targetsLists = targetsMap?.map(item =>
    getKeyValuePairFromArray(item[0] as string, item[1] as {targetsList})
  );

  return (
    <div className="flex flex-col">
      <div className="-my-2 overflow-x-auto sm:-mx-6 lg:-mx-8">
        <div className="py-2 align-middle inline-block min-w-full sm:px-6 lg:px-8">
          <div className="shadow overflow-hidden border-b border-gray-200 sm:rounded-lg">
            {targetsLists?.map(target => {
              const targetName = Object.keys(target)[0];
              const targetEndpointsList = target[targetName];
              return (
                <div key={targetName}>
                  <div>Name: {targetName}</div>
                  <TargetsTable endpoints={targetEndpointsList} />
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
