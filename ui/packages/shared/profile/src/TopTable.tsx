import React, {useEffect, useState} from 'react';
import {CalcWidth} from '@parca/dynamicsize';
import ProfileIcicleGraph from './ProfileIcicleGraph';
import {ProfileSource} from './ProfileSource';
import {QueryRequest, QueryResponse, QueryServiceClient, ServiceError} from '@parca/client';
import Card from '../../../app/web/src/components/ui/Card';
import Button from '@parca/web/src/components/ui/Button';
import * as parca_query_v1alpha1_query_pb from '@parca/client/src/parca/query/v1alpha1/query_pb';

interface ProfileViewProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
}

export interface IQueryResult {
  response: QueryResponse | null;
  error: ServiceError | null;
}

function arrayEquals(a, b): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  );
}

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource
): IQueryResult => {
  const [result, setResult] = useState<IQueryResult>({
    response: null,
    error: null,
  });

  useEffect(() => {
    const req = profileSource.QueryRequest();
    req.setReportType(QueryRequest.ReportType.REPORT_TYPE_TOP);

    client.query(
      req,
      (
        error: ServiceError | null,
        responseMessage: parca_query_v1alpha1_query_pb.QueryResponse | null
      ) => {
        setResult({
          response: responseMessage,
          error: error,
        });
      }
    );
  }, [client, profileSource]);

  return result;
};

export const TopTable = ({queryClient, profileSource}: ProfileViewProps): JSX.Element => {
  const [curPath, setCurPath] = useState<string[]>([]);
  const {response, error} = useQuery(queryClient, profileSource);

  if (error != null) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>;
  }

  const tableData = response?.toObject().top?.listList;

  return (
    <>
      <div className="w-full">
        <table className="table-auto text-left">
          <thead>
            <tr>
              <th>Flat</th>
              <th>Cumulative</th>
              <th>Name</th>
            </tr>
          </thead>
          <tbody>
            {tableData?.map((data, index) => (
              <tr key={index}>
                <td>{data.flat}</td>
                <td>{data.cumulative}</td>
                <td>
                  [{getLastItem(data.meta?.mapping?.file)}] {data.meta?.pb_function?.name}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
};

// TODO: Refactor the getLastItem from IcicleGraph.tsx
function getLastItem(thePath: string | undefined): string {
  if (thePath === undefined) {
    return '';
  }
  const index = thePath.lastIndexOf('/');
  if (index === -1) return thePath;

  return thePath.substring(index + 1);
}

export default TopTable;
