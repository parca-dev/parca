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

import {useCallback, useEffect, useMemo, useState} from 'react';

import {Icon} from '@iconify/react';
import {useQueryClient} from '@tanstack/react-query';
import cx from 'classnames';

import {QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {Query} from '@parca/parser';
import {sanitizeLabelValue} from '@parca/utilities';

import {useLabelNames} from '../MatchersInput';
import Select, {type SelectItem} from './Select';

interface Props {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  queryBrowserRef: React.RefObject<HTMLDivElement>;
}

interface QueryRow {
  labelName: string;
  operator: string;
  labelValue: string;
  labelValues: string[];
  isLoading: boolean;
}

const transformLabelsForSelect = (labelNames: string[]): SelectItem[] => {
  return labelNames.map(labelName => ({
    key: labelName,
    element: {active: <>{labelName}</>, expanded: <>{labelName}</>},
  }));
};

const operatorOptions = [
  {
    key: '=',
    element: {
      active: <>=</>,
      expanded: (
        <>
          <span>=</span>
        </>
      ),
    },
  },
  {
    key: '!=',
    element: {
      active: <>{'!='}</>,
      expanded: (
        <>
          <span>{'!='}</span>
        </>
      ),
    },
  },
  {
    key: '=~',
    element: {
      active: <>{'=~'}</>,
      expanded: (
        <>
          <span>{'=~'}</span>
        </>
      ),
    },
  },
  {
    key: '!~',
    element: {
      active: <>{'!~'}</>,
      expanded: (
        <>
          <span>{'!~'}</span>
        </>
      ),
    },
  },
];

const SimpleMatchers = ({
  queryClient,
  setMatchersString,
  // runQuery,
  currentQuery,
  profileType,
  queryBrowserRef,
}: Props): JSX.Element => {
  const [queryRows, setQueryRows] = useState<QueryRow[]>([
    {labelName: '', operator: '=', labelValue: '', labelValues: [], isLoading: false},
  ]);
  const reactQueryClient = useQueryClient();
  const metadata = useGrpcMetadata();

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient, profileType);
  const {response: labelNamesResponse, error: labelNamesError} = result;
  const [showAll, setShowAll] = useState(false);

  const visibleRows = showAll ? queryRows : queryRows.slice(0, 3);
  const hiddenRowsCount = queryRows.length - 3;

  const maxWidthInPixels = `max-w-[${queryBrowserRef.current?.offsetWidth.toString() as string}px]`;

  const currentMatchers = currentQuery.matchersString();

  const labelNameFromMatchers = useMemo(() => {
    if (currentQuery === undefined) return [];

    const matchers = currentQuery.matchers;

    return matchers.map(matcher => matcher.key);
  }, [currentQuery]);

  const fetchLabelValues = useCallback(
    async (labelName: string): Promise<string[]> => {
      if (labelName == null || labelName === '' || profileType == null || profileType === '') {
        return [];
      }
      try {
        const values = await reactQueryClient.fetchQuery(
          [labelName, profileType],
          async () => {
            const response = await queryClient.values(
              {labelName, match: [], profileType},
              {meta: metadata}
            ).response;
            const sanitizedValues = sanitizeLabelValue(response.labelValues);
            return sanitizedValues;
          },
          {
            staleTime: 1000 * 60 * 5, // 5 minutes
          }
        );
        return values;
      } catch (error) {
        console.error('Error fetching label values:', error);
        return [];
      }
    },
    [queryClient, metadata, profileType, reactQueryClient]
  );

  const updateMatchersString = useCallback(
    (rows: QueryRow[]) => {
      const matcherString = rows
        .filter(row => row.labelName.length > 0 && row.labelValue)
        .map(row => `${row.labelName}${row.operator}"${row.labelValue}"`)
        .join(',');
      setMatchersString(matcherString);
    },
    [setMatchersString]
  );

  useEffect(() => {
    if (currentMatchers === '') {
      return;
    }

    const fetchAndSetQueryRows = async (): Promise<void> => {
      const newRows = await Promise.all(
        currentMatchers.split(',').map(async matcher => {
          const match = matcher.match(/([^=!~]+)([=!~]{1,2})(.+)/);
          if (match === null) return null;

          const [, labelName, operator, labelValue] = match;
          const trimmedLabelName = labelName.trim();
          if (trimmedLabelName === '') return null;

          const labelValues = await fetchLabelValues(trimmedLabelName);
          const sanitizedLabelValue =
            labelValue.startsWith('"') && labelValue.endsWith('"')
              ? labelValue.slice(1, -1)
              : labelValue;

          return {
            labelName: trimmedLabelName,
            operator,
            labelValue: sanitizedLabelValue,
            labelValues,
          };
        })
      );

      const filteredRows = newRows.filter((row): row is QueryRow => row !== null);
      setQueryRows(filteredRows);
      updateMatchersString(filteredRows);
    };

    void fetchAndSetQueryRows();
  }, [currentMatchers, fetchLabelValues, updateMatchersString]);

  const labelNames = useMemo(() => {
    return (labelNamesError === undefined || labelNamesError == null) &&
      labelNamesResponse !== undefined &&
      labelNamesResponse != null
      ? labelNamesResponse.labelNames.filter(e => e !== '__name__')
      : [];
  }, [labelNamesError, labelNamesResponse]);

  const labelNameOptions = useMemo(() => {
    const uniqueLabelNames = Array.from(new Set([...labelNames, ...labelNameFromMatchers]));
    return transformLabelsForSelect(uniqueLabelNames);
  }, [labelNames, labelNameFromMatchers]);

  const updateRow = useCallback(
    async (index: number, field: keyof QueryRow, value: string): Promise<void> => {
      const updatedRows = [...queryRows];
      const prevLabelName = updatedRows[index].labelName;
      updatedRows[index] = {...updatedRows[index], [field]: value};

      if (field === 'labelName' && value !== prevLabelName) {
        updatedRows[index].labelValues = [];
        updatedRows[index].labelValue = '';
        updatedRows[index].isLoading = true;
        setQueryRows([...updatedRows]);

        const labelValues = await fetchLabelValues(value);
        updatedRows[index].labelValues = labelValues;
        updatedRows[index].isLoading = false;
      }

      setQueryRows([...updatedRows]);
      updateMatchersString(updatedRows);
    },
    [queryRows, fetchLabelValues, updateMatchersString]
  );

  const handleUpdateRow = useCallback(
    (index: number, field: keyof QueryRow, value: string) => {
      void updateRow(index, field, value);
    },
    [updateRow]
  );

  const addNewRow = useCallback((): void => {
    const newRows = [
      ...queryRows,
      {labelName: '', operator: '=', labelValue: '', labelValues: [], isLoading: false},
    ];
    setQueryRows(newRows);
    updateMatchersString(newRows);
  }, [queryRows, updateMatchersString]);

  const removeRow = useCallback(
    (index: number): void => {
      if (queryRows.length === 1) {
        // Reset the single row instead of removing it
        const resetRow = {
          labelName: '',
          operator: '=',
          labelValue: '',
          labelValues: [],
          isLoading: false,
        };
        setQueryRows([resetRow]);
        updateMatchersString([resetRow]);
      } else {
        const updatedRows = queryRows.filter((_, i) => i !== index);
        setQueryRows(updatedRows);
        updateMatchersString(updatedRows);
      }
    },
    [queryRows, updateMatchersString]
  );

  const handleLabelValueClick = useCallback(
    (index: number) => {
      return async () => {
        const updatedRows = [...queryRows];
        if (updatedRows[index].labelValues.length === 0 && updatedRows[index].labelName !== '') {
          updatedRows[index].isLoading = true;
          setQueryRows([...updatedRows]);

          try {
            const labelValues = await fetchLabelValues(updatedRows[index].labelName);
            updatedRows[index].labelValues = labelValues;
          } catch (error) {
            console.error('Error fetching label values:', error);
          } finally {
            updatedRows[index].isLoading = false;
            setQueryRows([...updatedRows]);
          }
        } else {
          console.log(`Label values already present or empty label name`);
        }
      };
    },
    [queryRows, fetchLabelValues]
  );

  const isRowRegex = (row: QueryRow): boolean => row.operator === '=~' || row.operator === '!~';

  return (
    <div
      className={`flex items-center gap-3 ${maxWidthInPixels} w-full flex-wrap`}
      id="simple-matchers"
    >
      {visibleRows.map((row, index) => (
        <div key={index} className="flex items-center">
          <Select
            items={labelNameOptions}
            onSelection={value => handleUpdateRow(index, 'labelName', value)}
            placeholder="Select label name"
            selectedKey={row.labelName}
            className="rounded-tr-none rounded-br-none ring-0 focus:ring-0 outline-none"
            loading={labelNamesLoading}
            searchable={true}
          />
          <Select
            items={operatorOptions}
            onSelection={value => handleUpdateRow(index, 'operator', value)}
            selectedKey={row.operator}
            className="rounded-none ring-0 focus:ring-0 outline-none"
          />
          <Select
            items={transformLabelsForSelect(row.labelValues)}
            onSelection={value => handleUpdateRow(index, 'labelValue', value)}
            placeholder="Select label value"
            selectedKey={row.labelValue}
            className="rounded-none ring-0 focus:ring-0 outline-none max-w-48"
            optionsClassname={cx('max-w-[300px]', {
              'w-[300px]': isRowRegex(row),
            })}
            searchable={true}
            disabled={row.labelName === ''}
            loading={row.isLoading}
            onButtonClick={() => handleLabelValueClick(index)}
            editable={isRowRegex(row)}
          />
          <button
            onClick={() => removeRow(index)}
            className={cx(
              'p-2 border-gray-200 border rounded rounded-tl-none rounded-bl-none focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900'
            )}
          >
            <Icon icon="carbon:close" className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </button>
        </div>
      ))}

      {queryRows.length > 3 && (
        <button
          onClick={() => setShowAll(!showAll)}
          className="mr-2 px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:bg-gray-900"
        >
          {showAll ? 'Show less' : `Show ${hiddenRowsCount} more`}
        </button>
      )}

      <button
        onClick={addNewRow}
        className="p-2 border-gray-200 dark:bg-gray-900 dark:border-gray-600 border rounded focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
      >
        <Icon icon="material-symbols:add" className="h-5 w-5 text-gray-400" aria-hidden="true" />
      </button>
    </div>
  );
};

export default SimpleMatchers;
