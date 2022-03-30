import React, {useEffect, useState} from 'react';
import {
  QueryRequest,
  QueryResponse,
  QueryServiceClient,
  ServiceError,
  TopNodeMeta,
} from '@parca/client';
import * as parca_query_v1alpha1_query_pb from '@parca/client/src/parca/query/v1alpha1/query_pb';
import {getLastItem, valueFormatter} from '@parca/functions';
import {useGrpcMetadata} from '@parca/components';

import {ProfileSource} from './ProfileSource';
import './TopTable.styles.css';

interface ProfileViewProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
}

export interface IQueryResult {
  response: QueryResponse | null;
  error: ServiceError | null;
}

const Arrow = ({direction}: {direction: string | undefined}) => {
  return (
    <svg
      className={`${direction !== undefined ? 'fill-[#161616] dark:fill-[#ffffff]' : ''}`}
      fill="#777d87"
      height="10"
      viewBox="0 0 11 10"
      width="11"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path clip-rule="evenodd" d="m.573997 0 5.000003 10 5-10h-9.999847z" fill-rule="evenodd" />
    </svg>
  );
};

const useSortableData = (
  response: QueryResponse | null,
  config = {key: 'cumulative', direction: 'desc'}
) => {
  const [sortConfig, setSortConfig] = React.useState<{key: string; direction: string} | null>(
    config
  );

  const rawTableReport = response?.toObject().top?.listList;

  const items = rawTableReport?.map(node => ({
    ...node,
    name: node.meta?.pb_function?.name,
  }));

  const sortedItems = React.useMemo(() => {
    if (!items) return;

    let sortableItems = [...items];
    if (sortConfig !== null) {
      sortableItems.sort((a, b) => {
        if (a[sortConfig.key] < b[sortConfig.key]) {
          return sortConfig.direction === 'asc' ? -1 : 1;
        }
        if (a[sortConfig.key] > b[sortConfig.key]) {
          return sortConfig.direction === 'asc' ? 1 : -1;
        }
        return 0;
      });
    }
    return sortableItems;
  }, [items, sortConfig]);

  const requestSort = key => {
    let direction = 'desc';
    if (sortConfig && sortConfig.key === key && sortConfig.direction === 'desc') {
      direction = 'asc';
    }
    setSortConfig({key, direction});
  };

  return {items: sortedItems, requestSort, sortConfig};
};

export const useQuery = (
  client: QueryServiceClient,
  profileSource: ProfileSource
): IQueryResult => {
  const [result, setResult] = useState<IQueryResult>({
    response: null,
    error: null,
  });
  const metadata = useGrpcMetadata();

  useEffect(() => {
    const req = profileSource.QueryRequest();
    req.setReportType(QueryRequest.ReportType.REPORT_TYPE_TOP);

    client.query(
      req,
      metadata,
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

export const RowLabel = (meta: TopNodeMeta.AsObject | undefined): string => {
  if (meta === undefined) return '<unknown>';
  const mapping = `${
    meta?.mapping?.file !== undefined && meta?.mapping?.file !== ''
      ? `[${getLastItem(meta.mapping.file)}]`
      : ''
  }`;
  if (meta.pb_function?.name !== undefined && meta.pb_function?.name !== '')
    return `${mapping} ${meta.pb_function.name}`;

  const address = `${
    meta.location?.address !== undefined && meta.location?.address !== 0
      ? `0x${meta.location.address.toString(16)}`
      : ''
  }`;
  const fallback = `${mapping} ${address}`;

  return fallback === '' ? '<unknown>' : fallback;
};

export const TopTable = ({queryClient, profileSource}: ProfileViewProps): JSX.Element => {
  const {response, error} = useQuery(queryClient, profileSource);
  const {items, requestSort, sortConfig} = useSortableData(response);

  const unit = response?.toObject().top?.unit as string;

  if (error != null) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>;
  }

  const total = response?.toObject().top?.listList.length;
  if (total === 0) return <>Profile has no samples</>;

  const getClassNamesFor = name => {
    if (!sortConfig) {
      return;
    }
    return sortConfig.key === name ? sortConfig.direction : undefined;
  };

  return (
    <>
      <div className="w-full">
        <table className="iciclegraph-table table-auto text-left w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead className="bg-gray-50 dark:bg-gray-800">
            <tr>
              <th
                className="text-sm cursor-pointer pt-2 pb-2 pl-2"
                onClick={() => requestSort('name')}
              >
                Name
                <span className={`inline-block align-middle ml-2 ${getClassNamesFor('name')}`}>
                  <Arrow direction={getClassNamesFor('name')} />
                </span>
              </th>
              <th
                className="text-left text-sm cursor-pointer pt-2 pb-2"
                onClick={() => requestSort('flat')}
              >
                Flat
                <span className={`inline-block align-middle ml-2 ${getClassNamesFor('flat')}`}>
                  <Arrow direction={getClassNamesFor('flat')} />
                </span>
              </th>
              <th
                className="text-right text-sm cursor-pointer pt-2 pb-2 pr-2"
                onClick={() => requestSort('cumulative')}
              >
                Cumulative
                <span
                  className={`inline-block align-middle ml-2 ${getClassNamesFor('cumulative')}`}
                >
                  <Arrow direction={getClassNamesFor('cumulative')} />
                </span>
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200 dark:bg-gray-900 dark:divide-gray-700">
            {items?.map((report, index) => (
              <tr key={index} className="hover:bg-[#62626212] dark:hover:bg-[#ffffff12]">
                <td className="text-xs py-1.5 pl-2 min-w-[150px] max-w-[450px]">
                  {RowLabel(report.meta)}
                </td>
                <td className="text-xs min-w-[150px] max-w-[150px] py-1.5text-right">
                  {valueFormatter(report.flat, unit, 2)}
                </td>
                <td className="text-xs min-w-[150px] max-w-[150px] py-1.5 text-right pr-2">
                  {valueFormatter(report.cumulative, unit, 2)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
};

export default TopTable;
