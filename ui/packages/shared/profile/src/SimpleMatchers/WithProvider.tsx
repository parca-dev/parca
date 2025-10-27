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

import {useCallback, useMemo} from 'react';

import {useQueryClient} from '@tanstack/react-query';

import SimpleMatchers from '.';
import {LabelProvider, LabelSource} from '../contexts/SimpleMatchersLabelContext';
import {useUnifiedLabels} from '../contexts/UnifiedLabelsContext';
import {useLabelNames} from '../hooks/useLabels';

const SimpleMatchersWithProvider = (): JSX.Element => {
  const {
    queryClient,
    setMatchersString,
    runQuery,
    currentQuery: query,
    profileType,
    queryBrowserRef,
    start,
    end,
    searchExecutedTimestamp,
    externalLabelSource,
  } = useUnifiedLabels();

  const {
    loading,
    result,
    refetch: refetchLabelNames,
  } = useLabelNames(queryClient, profileType, start, end);

  const reactQueryClient = useQueryClient();

  const labelNameFromMatchers = useMemo(() => {
    if (query === undefined) return [];

    const matchers = query.matchers;

    return matchers.map(matcher => matcher.key);
  }, [query]);

  const labelSources = useMemo(() => {
    const sources: LabelSource[] = [];

    const profileLabelNames =
      result.error != null
        ? []
        : result.response?.labelNames.filter((e: string) => e !== '__name__') ?? [];
    const uniqueProfileLabelNames = Array.from(new Set(profileLabelNames));

    sources.push({
      type: 'cpu',
      labelNames: uniqueProfileLabelNames,
      isLoading: loading,
      error: result.error ?? null,
    });

    if (externalLabelSource != null) {
      sources.push(externalLabelSource);
    }

    return sources;
  }, [result, loading, externalLabelSource]);

  const refetchLabelValues = useCallback(
    async (labelName?: string) => {
      await reactQueryClient.refetchQueries({
        predicate: query => {
          const key = query.queryKey;
          const matchesStructure =
            Array.isArray(key) &&
            key.length === 4 &&
            typeof key[0] === 'string' &&
            key[1] === profileType;

          if (!matchesStructure) return false;

          if (labelName !== undefined && labelName !== '') {
            return key[0] === labelName;
          }

          return true;
        },
      });
    },
    [reactQueryClient, profileType]
  );

  return (
    <LabelProvider
      labelSources={labelSources}
      labelNameFromMatchers={labelNameFromMatchers}
      refetchLabelNames={refetchLabelNames}
      refetchLabelValues={refetchLabelValues}
    >
      <SimpleMatchers
        queryClient={queryClient}
        setMatchersString={setMatchersString}
        runQuery={runQuery}
        currentQuery={query}
        profileType={profileType}
        queryBrowserRef={queryBrowserRef}
        start={start}
        end={end}
        searchExecutedTimestamp={searchExecutedTimestamp}
      />
    </LabelProvider>
  );
};

export default SimpleMatchersWithProvider;
