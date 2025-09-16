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

import React, {useCallback, useEffect, useRef, useState} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';

import {QueryServiceClient} from '@parca/client';
import {useGrpcMetadata} from '@parca/components';
import {Query} from '@parca/parser';
import {testId, TEST_IDS} from '@parca/test-utils';
import {millisToProtoTimestamp, sanitizeLabelValue} from '@parca/utilities';

import CustomSelect, {SelectItem} from '../SimpleMatchers/Select';

interface Props {
  labelNames: string[];
  profileType: string;
  runQuery: () => void;
  currentQuery: Query;
  queryClient: QueryServiceClient;
  setMatchersString: (arg: string) => void;
  start?: number;
  end?: number;
}

const ViewMatchers: React.FC<Props> = ({
  labelNames,
  profileType,
  queryClient,
  runQuery,
  setMatchersString,
  start,
  end,
  currentQuery,
}) => {
  const [labelValuesMap, setLabelValuesMap] = useState<Record<string, string[]>>({});
  const [isLoading, setIsLoading] = useState<Record<string, boolean>>({});
  const metadata = useGrpcMetadata();

  const currentMatchers = currentQuery.matchersString();

  const parseCurrentMatchers = useCallback((matchersString: string): Record<string, string> => {
    const matches = matchersString.match(/(\w+)="([^"]+)"/g);
    if (matches === null) return {};

    return matches.reduce<Record<string, string>>(
      (acc, match) => {
        const [label, value] = match.split('=');
        if (label !== undefined) {
          acc[label] = value.replace(/"/g, '');
        }
        return acc;
      },
      // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
      {} as Record<string, string>
    );
  }, []);

  const initialSelections = parseCurrentMatchers(currentMatchers);
  const selectionsRef = useRef<Record<string, string | null>>(initialSelections);

  const runQueryRef = useRef(runQuery);
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    runQueryRef.current = runQuery;
  }, [runQuery]);

  useEffect(() => {
    selectionsRef.current = initialSelections;
  }, [initialSelections]);

  const fetchLabelValues = useCallback(
    async (labelName: string): Promise<string[]> => {
      try {
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
        return sanitizeLabelValue(response.labelValues);
      } catch (error) {
        console.error('Error fetching label values:', error);
        return [];
      }
    },
    [queryClient, metadata, profileType, start, end]
  );

  useEffect(() => {
    const fetchAllLabelValues = async (): Promise<void> => {
      const newLabelValuesMap: Record<string, string[]> = {};
      const newIsLoading: Record<string, boolean> = {};

      for (const labelName of labelNames) {
        newIsLoading[labelName] = true;
        setIsLoading(prev => ({...prev, [labelName]: true}));

        const values = await fetchLabelValues(labelName);
        newLabelValuesMap[labelName] = values;
        newIsLoading[labelName] = false;
      }

      setLabelValuesMap(newLabelValuesMap);
      setIsLoading(newIsLoading);
    };

    void fetchAllLabelValues();
  }, [labelNames, fetchLabelValues]);

  const updateMatcherString = useCallback(() => {
    const matcherParts = Object.entries(selectionsRef.current)
      .filter(([_, v]) => v !== null && v !== '')
      .map(([ln, v]) => `${ln}="${v as string}"`);

    const matcherString = matcherParts.join(',');
    setMatchersString(matcherString);

    if (timeoutRef.current !== null) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      runQueryRef.current();
    }, 300);
  }, [setMatchersString]);

  const handleSelection = useCallback(
    (labelName: string, value: string | null): void => {
      selectionsRef.current = {
        ...selectionsRef.current,
        [labelName]: value,
      };

      updateMatcherString();
    },
    [updateMatcherString]
  );

  const handleReset = useCallback(
    (labelName: string): void => {
      handleSelection(labelName, null);
    },
    [handleSelection]
  );

  const transformValuesForSelect = useCallback((values: string[]): SelectItem[] => {
    return values.map(value => ({
      key: value,
      element: {active: <>{value}</>, expanded: <>{value}</>},
    }));
  }, []);

  return (
    <div className="flex flex-wrap gap-2" {...testId(TEST_IDS.VIEW_MATCHERS_CONTAINER)}>
      {labelNames.map(labelName => (
        <div key={labelName} className="flex items-center">
          <div className="relative border shadow-sm px-4 py-2 text-left cursor-default focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500 text-sm flex gap-2 items-center justify-between bg-gray-100 dark:bg-gray-700 rounded-l-md border-gray-300 dark:border-gray-600">
            {labelName}
          </div>
          <CustomSelect
            searchable={true}
            placeholder="Select value"
            items={transformValuesForSelect(labelValuesMap[labelName] ?? [])}
            onSelection={(value: string) => handleSelection(labelName, value)}
            selectedKey={selectionsRef.current[labelName] ?? undefined}
            className={cx(
              'rounded-l-none border-l-0',
              selectionsRef.current[labelName] != null && 'border-r-0 rounded-r-none'
            )}
            loading={isLoading[labelName] ?? false}
          />
          {selectionsRef.current[labelName] != null && (
            <button
              onClick={() => handleReset(labelName)}
              className="p-2 border-gray-200 bg-white dark:bg-gray-900 dark:border-gray-600 border rounded-r-md focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
              aria-label={`Reset ${labelName} selection`}
            >
              <Icon icon="mdi:close" className="h-5 w-5 text-gray-400" aria-hidden="true" />
            </button>
          )}
        </div>
      ))}
    </div>
  );
};

export default ViewMatchers;
