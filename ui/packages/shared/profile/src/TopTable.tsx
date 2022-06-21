import React from 'react';
import {getLastItem, valueFormatter, isSearchMatch, SEARCH_STRING_COLOR} from '@parca/functions';
import {useAppSelector, selectCompareMode, selectSearchNodeString} from '@parca/store';
import {
  QueryResponse,
  QueryServiceClient,
  QueryRequest_ReportType,
  TopNodeMeta,
} from '@parca/client';
import {ProfileSource} from './ProfileSource';
import {useQuery} from './useQuery';
import {hexifyAddress} from './utils';

import './TopTable.styles.css';

interface ProfileViewProps {
  queryClient: QueryServiceClient;
  profileSource: ProfileSource;
  sampleUnit: string;
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

  const rawTableReport =
    response !== null && response.report.oneofKind === 'top' ? response.report.top.list : [];

  const items = rawTableReport.map(node => ({
    ...node,
    diff: Number(node.diff),
    cumulative: Number(node.cumulative),
    flat: Number(node.flat),
    name: node.meta?.function?.name,
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

export const RowLabel = (meta: TopNodeMeta | undefined): string => {
  if (meta === undefined) return '<unknown>';
  const mapping = `${
    meta?.mapping?.file !== undefined && meta?.mapping?.file !== ''
      ? `[${getLastItem(meta.mapping.file)}]`
      : ''
  }`;
  if (meta.function?.name !== undefined && meta.function?.name !== '')
    return `${mapping} ${meta.function.name}`;

  const address = hexifyAddress(meta.location?.address);
  const fallback = `${mapping} ${address as string}`;

  return fallback === '' ? '<unknown>' : fallback;
};

export const TopTable = ({
  queryClient,
  profileSource,
  sampleUnit,
}: ProfileViewProps): JSX.Element => {
  const {response, error} = useQuery(queryClient, profileSource, QueryRequest_ReportType.TOP);
  const {items, requestSort, sortConfig} = useSortableData(response);
  const currentSearchString = useAppSelector(selectSearchNodeString);

  const compareMode = useAppSelector(selectCompareMode);

  const unit = sampleUnit;

  if (error != null) {
    return <div className="p-10 flex justify-center">An error occurred: {error.message}</div>;
  }

  const total =
    response !== null && response.report.oneofKind === 'top' ? response.report.top.list.length : 0;
  if (total === 0) return <>Profile has no samples</>;

  const getClassNamesFor = name => {
    if (!sortConfig) {
      return;
    }
    return sortConfig.key === name ? sortConfig.direction : undefined;
  };

  const addPlusSign = (num: string) => {
    if (num.charAt(0) === '0' || num.charAt(0) === '-') {
      return num;
    }

    return `+${num}`;
  };

  return (
    <>
      <div className="w-full font-robotoMono">
        <table className="iciclegraph-table table-fixed text-left w-full divide-y divide-gray-200 dark:divide-gray-700">
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
                className="text-right text-sm cursor-pointer pt-2 pb-2 w-[150px]"
                onClick={() => requestSort('flat')}
              >
                Flat
                <span className={`inline-block align-middle ml-2 ${getClassNamesFor('flat')}`}>
                  <Arrow direction={getClassNamesFor('flat')} />
                </span>
              </th>
              <th
                className="text-right text-sm cursor-pointer pt-2 pb-2 pr-2 w-[150px]"
                onClick={() => requestSort('cumulative')}
              >
                Cumulative
                <span
                  className={`inline-block align-middle ml-2 ${getClassNamesFor('cumulative')}`}
                >
                  <Arrow direction={getClassNamesFor('cumulative')} />
                </span>
              </th>
              {compareMode && (
                <th
                  className="text-right text-sm cursor-pointer pt-2 pb-2 pr-2 w-[150px]"
                  onClick={() => requestSort('diff')}
                >
                  Diff
                  <span className={`inline-block align-middle ml-2 ${getClassNamesFor('diff')}`}>
                    <Arrow direction={getClassNamesFor('diff')} />
                  </span>
                </th>
              )}
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200 dark:bg-gray-900 dark:divide-gray-700">
            {items?.map((report, index) => {
              const name = RowLabel(report.meta);
              return (
                <tr
                  key={index}
                  className="hover:bg-[#62626212] dark:hover:bg-[#ffffff12]"
                  style={{
                    opacity:
                      currentSearchString !== undefined &&
                      currentSearchString !== '' &&
                      !isSearchMatch(currentSearchString, name)
                        ? 0.5
                        : 1,
                  }}
                >
                  <td className="text-xs py-1.5 pl-2">{name}</td>
                  <td className="text-xs py-1.5 text-right">
                    {valueFormatter(report.flat, unit, 2)}
                  </td>
                  <td className="text-xs py-1.5 text-right pr-2">
                    {valueFormatter(report.cumulative, unit, 2)}
                  </td>
                  {compareMode && (
                    <td className="text-xs py-1.5 text-right pr-2">
                      {addPlusSign(valueFormatter(report.diff, unit, 2))}
                    </td>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </>
  );
};

export default TopTable;
