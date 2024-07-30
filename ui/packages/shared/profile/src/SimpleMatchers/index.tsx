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
import cx from 'classnames';

import {QueryServiceClient} from '@parca/client';
import {Select, useGrpcMetadata, type SelectItem} from '@parca/components';
import {Query} from '@parca/parser';
import {sanitizeLabelValue} from '@parca/utilities';

import {useLabelNames} from '../MatchersInput';

interface Props {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
}

interface QueryRow {
  labelName: string;
  operator: string;
  labelValue: string;
  labelValues: string[];
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
}: Props): JSX.Element => {
  const [queryRows, setQueryRows] = useState<QueryRow[]>([
    {labelName: '', operator: '=', labelValue: '', labelValues: []},
  ]);
  const metadata = useGrpcMetadata();

  const {loading: labelNamesLoading, result} = useLabelNames(queryClient, profileType);
  const {response: labelNamesResponse, error: labelNamesError} = result;

  const currentMatchers = currentQuery.matchersString();

  const fetchLabelValues = useCallback(
    async (labelName: string): Promise<string[]> => {
      try {
        const response = await queryClient.values(
          {labelName, match: [], profileType},
          {meta: metadata}
        ).response;
        return sanitizeLabelValue(response.labelValues);
      } catch (error) {
        console.error('Error fetching label values:', error);
        return [];
      }
    },
    [queryClient, metadata, profileType]
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
          const [labelName, operator, labelValue] = matcher.split(/(=|!=|=~|!~)/);
          if (labelName === '') return null;

          const labelValues = await fetchLabelValues(labelName);
          const sanitizedLabelValue =
            labelValue.startsWith('"') && labelValue.endsWith('"')
              ? labelValue.slice(1, -1)
              : labelValue;

          return {
            labelName,
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

  const getAvailableLabelNames = useCallback(
    (currentIndex: number) => {
      const usedLabelNames = queryRows
        .filter((_, index) => index !== currentIndex)
        .map(row => row.labelName);
      return labelNames.filter(name => !usedLabelNames.includes(name));
    },
    [labelNames, queryRows]
  );

  const updateRow = useCallback(
    async (index: number, field: keyof QueryRow, value: string): Promise<void> => {
      const updatedRows = [...queryRows];
      updatedRows[index] = {...updatedRows[index], [field]: value};

      if (field === 'labelName') {
        updatedRows[index].labelValues = await fetchLabelValues(value);
        updatedRows[index].labelValue = ''; // Reset the label value when changing label name
      }

      setQueryRows(updatedRows);
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
    const newRows = [...queryRows, {labelName: '', operator: '=', labelValue: '', labelValues: []}];
    setQueryRows(newRows);
    updateMatchersString(newRows);
  }, [queryRows, updateMatchersString]);

  const removeRow = useCallback(
    (index: number): void => {
      if (queryRows.length === 1) {
        // Reset the single row instead of removing it
        const resetRow = {labelName: '', operator: '=', labelValue: '', labelValues: []};
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

  return (
    <div className="flex items-center gap-3">
      {queryRows.map((row, index) => (
        <div key={index} className="flex items-center">
          <Select
            items={transformLabelsForSelect(getAvailableLabelNames(index))}
            onSelection={value => handleUpdateRow(index, 'labelName', value)}
            placeholder="Select label name"
            selectedKey={row.labelName}
            className="rounded-tr-none rounded-br-none"
            loading={labelNamesLoading}
            searchable={true}
          />
          <Select
            items={operatorOptions}
            onSelection={value => handleUpdateRow(index, 'operator', value)}
            selectedKey={row.operator}
            className="rounded-none"
          />
          <Select
            items={transformLabelsForSelect(row.labelValues)}
            onSelection={value => handleUpdateRow(index, 'labelValue', value)}
            placeholder="Select label value"
            selectedKey={row.labelValue}
            className="rounded-none"
            optionsClassname="max-w-[300px]"
            searchable={true}
          />
          <button
            onClick={() => removeRow(index)}
            className={cx(
              'p-2 border-gray-200 border rounded rounded-tl-none rounded-bl-none focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500'
            )}
          >
            <Icon icon="carbon:close" className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </button>
        </div>
      ))}

      <button
        onClick={addNewRow}
        className="p-2 border-gray-200 border rounded focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
      >
        <Icon icon="material-symbols:add" className="h-5 w-5 text-gray-400" aria-hidden="true" />
      </button>
    </div>
  );
};

export default SimpleMatchers;
