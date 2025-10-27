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

import {useCallback, useEffect, useState} from 'react';

import {Icon} from '@iconify/react';
import {useQueryClient} from '@tanstack/react-query';
import cx from 'classnames';

import {QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import {millisToProtoTimestamp, sanitizeLabelValue} from '@parca/utilities';

import {useLabels} from '../contexts/SimpleMatchersLabelContext';
import {transformLabelName} from '../contexts/utils';
import Select, {type SelectItem} from './Select';

interface Props {
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  runQuery: () => void;
  currentQuery: Query;
  profileType: string;
  queryBrowserRef: React.RefObject<HTMLDivElement>;
  start?: number;
  end?: number;
  searchExecutedTimestamp?: number;
}

interface QueryRow {
  labelName: string;
  operator: string;
  labelValue: string;
  labelValues: string[];
  isLoading: boolean;
}

export const transformLabelsForSelect = (labelNames: string[]): SelectItem[] => {
  return labelNames.map(labelName => ({
    key: labelName,
    element: {
      active: <>{transformLabelName(labelName)}</>,
      expanded: <>{transformLabelName(labelName)}</>,
    },
  }));
};

const operatorOptions = [
  {
    key: '=',
    element: {
      active: <>Equals</>,
      expanded: (
        <>
          <span>Equals</span>
        </>
      ),
    },
  },
  {
    key: '!=',
    element: {
      active: <>Not Equals</>,
      expanded: (
        <>
          <span>Not Equals</span>
        </>
      ),
    },
  },
  {
    key: '=~',
    element: {
      active: <>Regex</>,
      expanded: (
        <>
          <span>Regex</span>
        </>
      ),
    },
  },
  {
    key: '!~',
    element: {
      active: <>Not Regex</>,
      expanded: (
        <>
          <span>Not Regex</span>
        </>
      ),
    },
  },
];

const SimpleMatchers = ({
  queryClient,
  setMatchersString,
  currentQuery,
  profileType,
  queryBrowserRef,
  start,
  end,
  searchExecutedTimestamp,
}: Props): JSX.Element => {
  const [queryRows, setQueryRows] = useState<QueryRow[]>([
    {labelName: '', operator: '=', labelValue: '', labelValues: [], isLoading: false},
  ]);
  const reactQueryClient = useQueryClient();
  const metadata = useGrpcMetadata();

  const [showAll, setShowAll] = useState(false);
  const [isActivelyEditing, setIsActivelyEditing] = useState(false);

  // Reset editing mode when search is executed
  useEffect(() => {
    if (searchExecutedTimestamp !== undefined && searchExecutedTimestamp > 0) {
      setIsActivelyEditing(false);
      setShowAll(false);
    }
  }, [searchExecutedTimestamp]);

  const {
    labelNameOptions,
    isLoading: labelNamesLoading,
    fetchLabelValues: contextFetchLabelValues,
    refetchLabelNames,
  } = useLabels();

  const visibleRows = showAll || isActivelyEditing ? queryRows : queryRows.slice(0, 3);
  const hiddenRowsCount = queryRows.length - 3;

  const maxWidthInPixels = `max-w-[${queryBrowserRef.current?.offsetWidth.toString() as string}px]`;

  const currentMatchers = currentQuery.matchersString();

  const fetchLabelValues = useCallback(
    async (labelName: string): Promise<string[]> => {
      if (labelName == null || labelName === '') {
        return [];
      }

      if (contextFetchLabelValues != null) {
        try {
          const values = await contextFetchLabelValues(labelName);
          if (values.length > 0) return values;
        } catch (error) {
          console.error('Error fetching from external source:', error);
        }
      }

      if (profileType == null || profileType === '') {
        return [];
      }
      try {
        const values = await reactQueryClient.fetchQuery(
          [labelName, profileType, start, end],
          async () => {
            const response = await queryClient.values(
              {
                labelName,
                match: [],
                profileType,
                ...(start !== undefined && end !== undefined
                  ? {
                      start: millisToProtoTimestamp(start),
                      end: millisToProtoTimestamp(end),
                    }
                  : {}),
              },
              {meta: metadata}
            ).response;
            console.log('Fetched label values:', response.labelValues);
            const sanitizedValues = sanitizeLabelValue(response.labelValues);
            return sanitizedValues;
          }
        );
        return values;
      } catch (error) {
        console.error('Error fetching label values:', error);
        return [];
      }
    },
    [contextFetchLabelValues, queryClient, metadata, profileType, reactQueryClient, start, end]
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

  // Helper to ensure selected label name is in the options (for page load before API returns)
  const getLabelNameOptionsWithSelected = useCallback(
    (selectedLabelName: string): typeof labelNameOptions => {
      if (selectedLabelName === '') return labelNameOptions;

      // Check if the selected label name exists in any group
      const existsInOptions = labelNameOptions.some(group =>
        group.values.some(item => item.key === selectedLabelName)
      );

      if (existsInOptions) return labelNameOptions;

      // Add it temporarily to a group with type '' (matching non-matching labels group)
      if (labelNameOptions.length === 0) {
        return [
          {
            type: '',
            values: [
              {
                key: selectedLabelName,
                element: {active: <>{selectedLabelName}</>, expanded: <>{selectedLabelName}</>},
              },
            ],
          },
        ];
      }

      // Find the group with type '' (non-matching labels) or create it
      const emptyTypeGroupIndex = labelNameOptions.findIndex(group => group.type === '');

      if (emptyTypeGroupIndex !== -1) {
        // Add to existing '' type group
        const updatedOptions = [...labelNameOptions];
        updatedOptions[emptyTypeGroupIndex] = {
          ...updatedOptions[emptyTypeGroupIndex],
          values: [
            ...updatedOptions[emptyTypeGroupIndex].values,
            {
              key: selectedLabelName,
              element: {active: <>{selectedLabelName}</>, expanded: <>{selectedLabelName}</>},
            },
          ],
        };
        return updatedOptions;
      } else {
        // Create new '' type group
        return [
          ...labelNameOptions,
          {
            type: '',
            values: [
              {
                key: selectedLabelName,
                element: {active: <>{selectedLabelName}</>, expanded: <>{selectedLabelName}</>},
              },
            ],
          },
        ];
      }
    },
    [labelNameOptions]
  );

  const fetchLabelValuesUnified = useCallback(
    async (labelName: string): Promise<string[]> => {
      return await fetchLabelValues(labelName);
    },
    [fetchLabelValues]
  );

  useEffect(() => {
    if (currentMatchers === '') {
      const defaultRow = {
        labelName: '',
        operator: '=',
        labelValue: '',
        labelValues: [],
        isLoading: false,
      };
      setQueryRows([defaultRow]);
      return;
    }

    let isCancelled = false;

    const fetchAndSetQueryRows = async (): Promise<void> => {
      const newRows = await Promise.all(
        currentMatchers.split(',').map(async matcher => {
          const match = matcher.match(/([^=!~]+)([=!~]{1,2})(.+)/);
          if (match === null) return null;

          const [, labelName, operator, labelValue] = match;
          const trimmedLabelName = labelName.trim();
          if (trimmedLabelName === '') return null;

          const labelValues = await fetchLabelValuesUnified(trimmedLabelName);
          const sanitizedLabelValue =
            labelValue.startsWith('"') && labelValue.endsWith('"')
              ? labelValue.slice(1, -1)
              : labelValue;

          return {
            labelName: trimmedLabelName,
            operator,
            labelValue: sanitizedLabelValue,
            labelValues,
            isLoading: false,
          };
        })
      );

      if (!isCancelled) {
        const filteredRows = newRows.filter((row): row is QueryRow => row !== null);
        setQueryRows(filteredRows);
      }
    };

    void fetchAndSetQueryRows();

    return () => {
      isCancelled = true;
    };
  }, [currentMatchers, fetchLabelValuesUnified]);

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

        const labelValues = await fetchLabelValuesUnified(value);
        updatedRows[index].labelValues = labelValues;
        updatedRows[index].isLoading = false;
      }

      setQueryRows([...updatedRows]);
      updateMatchersString(updatedRows);
    },
    [queryRows, fetchLabelValuesUnified, updateMatchersString]
  );

  const handleUpdateRow = useCallback(
    (index: number, field: keyof QueryRow, value: string) => {
      // Only set actively editing if not already showing all
      // If showAll is true, user has explicitly expanded, so keep that state
      if (!showAll) {
        setIsActivelyEditing(true);
      }
      void updateRow(index, field, value);
    },
    [updateRow, showAll]
  );

  const addNewRow = useCallback((): void => {
    // Only set actively editing if not already showing all label names and values
    // If showAll is true, user has explicitly expanded, so keep that state
    if (!showAll) {
      setIsActivelyEditing(true);
    }
    const newRows = [
      ...queryRows,
      {labelName: '', operator: '=', labelValue: '', labelValues: [], isLoading: false},
    ];
    setQueryRows(newRows);
    updateMatchersString(newRows);
  }, [queryRows, updateMatchersString, showAll]);

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
            const labelValues = await fetchLabelValuesUnified(updatedRows[index].labelName);
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
    [queryRows, fetchLabelValuesUnified]
  );

  const isRowRegex = (row: QueryRow): boolean => row.operator === '=~' || row.operator === '!~';

  const handleRefetchForLabelValues = useCallback(
    async (labelName: string): Promise<void> => {
      await fetchLabelValuesUnified(labelName);
    },
    [fetchLabelValuesUnified]
  );

  return (
    <div
      className={`flex items-center gap-3 ${maxWidthInPixels} w-full flex-wrap`}
      id="simple-matchers"
      {...testId(TEST_IDS.SIMPLE_MATCHERS_CONTAINER)}
    >
      {visibleRows.map((row, index) => (
        <div key={index} className="flex items-center" {...testId(TEST_IDS.SIMPLE_MATCHER_ROW)}>
          <Select
            items={getLabelNameOptionsWithSelected(row.labelName)}
            onSelection={value => handleUpdateRow(index, 'labelName', value)}
            placeholder="Select label name"
            selectedKey={row.labelName}
            className="rounded-tr-none rounded-br-none ring-0 focus:ring-0 outline-none"
            loading={labelNamesLoading}
            searchable={true}
            {...testId(TEST_IDS.LABEL_NAME_SELECT)}
            refetchValues={refetchLabelNames}
          />
          <Select
            items={operatorOptions}
            onSelection={value => handleUpdateRow(index, 'operator', value)}
            selectedKey={row.operator}
            className="rounded-none ring-0 focus:ring-0 outline-none"
            {...testId(TEST_IDS.OPERATOR_SELECT)}
          />
          <Select
            items={transformLabelsForSelect(
              row.labelValue !== '' && !row.labelValues.includes(row.labelValue)
                ? [...row.labelValues, row.labelValue]
                : row.labelValues
            )}
            onSelection={value => handleUpdateRow(index, 'labelValue', value)}
            placeholder="Select label value"
            selectedKey={row.labelValue}
            className="rounded-none ring-0 focus:ring-0 outline-none max-w-48"
            optionsClassname={cx('max-w-[600px]', {
              'w-[300px]': isRowRegex(row),
            })}
            searchable={true}
            disabled={row.labelName === ''}
            loading={row.isLoading}
            onButtonClick={() => handleLabelValueClick(index)}
            editable={isRowRegex(row)}
            {...testId(TEST_IDS.LABEL_VALUE_SELECT)}
            refetchValues={async () => await handleRefetchForLabelValues(row.labelName)}
            showLoadingInButton={true}
          />
          <button
            onClick={() => removeRow(index)}
            className={cx(
              'p-2 border-gray-200 border rounded rounded-tl-none rounded-bl-none focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900'
            )}
            {...testId(TEST_IDS.REMOVE_MATCHER_BUTTON)}
          >
            <Icon icon="carbon:close" className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </button>
        </div>
      ))}

      {queryRows.length > 3 && !isActivelyEditing && (
        <button
          onClick={() => {
            if (showAll) {
              // Clicking "Show less" - collapse and stop editing mode
              setShowAll(false);
              setIsActivelyEditing(false);
            } else {
              // Clicking "Show more" - just expand
              setShowAll(true);
            }
          }}
          className="mr-2 px-3 py-1 text-sm font-medium text-gray-700 dark:text-gray-200 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 dark:bg-gray-900"
          {...testId(showAll ? TEST_IDS.SHOW_LESS_BUTTON : TEST_IDS.SHOW_MORE_BUTTON)}
        >
          {showAll ? 'Show less' : `Show ${hiddenRowsCount} more`}
        </button>
      )}

      <button
        onClick={addNewRow}
        className="p-2 border-gray-200 dark:bg-gray-900 dark:border-gray-600 border rounded focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
        {...testId(TEST_IDS.ADD_MATCHER_BUTTON)}
      >
        <Icon icon="material-symbols:add" className="h-5 w-5 text-gray-400" aria-hidden="true" />
      </button>
    </div>
  );
};

export default SimpleMatchers;
