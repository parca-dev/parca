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

import {useCallback, useMemo, useRef, useState} from 'react';

import {Switch} from '@headlessui/react';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import {useQueryClient} from '@tanstack/react-query';
import {type SelectInstance} from 'react-select';

import {ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {Button, DateTimeRange, DateTimeRangePicker, useParcaContext} from '@parca/components';
import {ProfileType, Query} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';

import MatchersInput, {useLabelNames} from '../MatchersInput';
import ProfileTypeSelector from '../ProfileTypeSelector';
import {SelectWithRefresh} from '../SelectWithRefresh';
import {SimpleMatchers} from '../SimpleMatchers';
import ViewMatchers from '../ViewMatchers';
import {LabelProvider, LabelSource} from '../contexts/SimpleMatchersLabelContext';

interface SelectOption {
  label: string;
  value: string;
}

interface QueryControlsProps {
  queryClient: QueryServiceClient;
  query: Query;
  profileType: string | ProfileType;
  timeRangeSelection: DateTimeRange;
  setTimeRangeSelection: (range: DateTimeRange) => void;
  setMatchersString: (matchers: string) => void;
  setQueryExpression: (updateTs?: boolean) => void;
  searchDisabled: boolean;
  showProfileTypeSelector?: boolean;
  showSumBySelector?: boolean;
  showAdvancedMode?: boolean;
  disableExplorativeQuerying?: boolean;
  profileTypesData?: ProfileTypesResponse;
  profileTypesLoading?: boolean;
  selectedProfileName?: string;
  setProfileName?: (name: string | undefined) => void;
  profileTypesError?: RpcError;
  viewComponent?: {
    disableProfileTypesDropdown?: boolean;
    disableExplorativeQuerying?: boolean;
    labelnames?: string[];
    createViewComponent?: React.ReactNode;
  };
  setQueryBrowserMode?: (mode: string) => void;
  advancedModeForQueryBrowser?: boolean;
  setAdvancedModeForQueryBrowser?: (mode: boolean) => void;
  queryBrowserRef?: React.RefObject<HTMLDivElement>;
  labels?: string[];
  sumBySelection?: string[];
  sumBySelectionLoading?: boolean;
  setUserSumBySelection?: (sumBy: string[]) => void;
  sumByRef?: React.RefObject<SelectInstance>;
  externalLabelSource?: {
    type: string;
    labelNames: string[];
    isLoading: boolean;
    error?: Error | null;
    fetchLabelValues?: (labelName: string) => Promise<string[]>;
    refetchLabelNames?: () => Promise<void>;
    refetchLabelValues?: (labelName?: string) => Promise<void>;
  };
}

export function QueryControls({
  queryClient,
  query,
  profileType,
  timeRangeSelection,
  setTimeRangeSelection,
  setMatchersString,
  setQueryExpression,
  searchDisabled,
  showProfileTypeSelector = false,
  showSumBySelector = false,
  showAdvancedMode = true,
  profileTypesData,
  profileTypesLoading = false,
  selectedProfileName,
  setProfileName,
  profileTypesError,
  viewComponent,
  setQueryBrowserMode,
  advancedModeForQueryBrowser = false,
  setAdvancedModeForQueryBrowser,
  queryBrowserRef,
  labels = [],
  sumBySelection = [],
  sumBySelectionLoading = false,
  setUserSumBySelection,
  sumByRef,
  externalLabelSource,
}: QueryControlsProps): JSX.Element {
  const {timezone} = useParcaContext();
  const defaultQueryBrowserRef = useRef<HTMLDivElement>(null);
  const actualQueryBrowserRef = queryBrowserRef ?? defaultQueryBrowserRef;
  const [searchExecutedTimestamp, setSearchExecutedTimestamp] = useState<number>(0);
  const reactQueryClient = useQueryClient();

  const {
    loading,
    result,
    refetch: refetchLabelNames,
  } = useLabelNames(
    queryClient,
    profileType as string,
    timeRangeSelection.getFromMs(),
    timeRangeSelection.getToMs()
  );

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

  return (
    <LabelProvider
      labelSources={labelSources}
      labelNameFromMatchers={labelNameFromMatchers}
      refetchLabelNames={refetchLabelNames}
      refetchLabelValues={refetchLabelValues}
    >
      <div
        className="flex w-full flex-wrap items-start gap-2"
        {...testId(TEST_IDS.QUERY_CONTROLS_CONTAINER)}
      >
        {showProfileTypeSelector && (
          <div>
            <label className="text-xs" {...testId(TEST_IDS.PROFILE_TYPE_LABEL)}>
              Profile type
            </label>
            <ProfileTypeSelector
              profileTypesData={profileTypesData}
              loading={profileTypesLoading}
              selectedKey={selectedProfileName}
              onSelection={setProfileName ?? (() => {})}
              error={profileTypesError}
              disabled={viewComponent?.disableProfileTypesDropdown}
            />
          </div>
        )}

        <div
          className="w-full flex-1 flex flex-col gap-1 mt-auto"
          ref={actualQueryBrowserRef}
          {...testId(TEST_IDS.QUERY_BROWSER_CONTAINER)}
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <label className="text-xs" {...testId(TEST_IDS.QUERY_LABEL)}>
                Query
              </label>
              {showAdvancedMode && viewComponent?.disableExplorativeQuerying !== true && (
                <>
                  <Switch
                    checked={advancedModeForQueryBrowser}
                    onChange={() => {
                      setAdvancedModeForQueryBrowser?.(!advancedModeForQueryBrowser);
                      setQueryBrowserMode?.(advancedModeForQueryBrowser ? 'simple' : 'advanced');
                    }}
                    className={`${
                      advancedModeForQueryBrowser ? 'bg-indigo-600' : 'bg-gray-400 dark:bg-gray-800'
                    } relative inline-flex h-[20px] w-[44px] shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-white/75`}
                    {...testId(TEST_IDS.ADVANCED_MODE_SWITCH)}
                  >
                    <span className="sr-only">Use setting</span>
                    <span
                      aria-hidden="true"
                      className={`${
                        advancedModeForQueryBrowser ? 'translate-x-6' : 'translate-x-0'
                      } pointer-events-none inline-block h-[16px] w-[16px] transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out`}
                    />
                  </Switch>
                  <label className="text-xs" {...testId(TEST_IDS.QUERY_MODE_LABEL)}>
                    Advanced Mode
                  </label>
                </>
              )}
            </div>
            {viewComponent?.createViewComponent}
          </div>

          {viewComponent?.disableExplorativeQuerying === true &&
          viewComponent?.labelnames !== undefined &&
          viewComponent?.labelnames.length >= 1 ? (
            <ViewMatchers
              labelNames={viewComponent.labelnames}
              setMatchersString={setMatchersString}
              profileType={selectedProfileName ?? profileType.toString()}
              runQuery={setQueryExpression}
              currentQuery={query}
              queryClient={queryClient}
              start={timeRangeSelection.getFromMs()}
              end={timeRangeSelection.getToMs()}
            />
          ) : showAdvancedMode && advancedModeForQueryBrowser ? (
            <MatchersInput
              setMatchersString={setMatchersString}
              runQuery={setQueryExpression}
              currentQuery={query}
              profileType={selectedProfileName ?? profileType.toString()}
              queryClient={queryClient}
              start={timeRangeSelection.getFromMs()}
              end={timeRangeSelection.getToMs()}
              externalLabelNames={externalLabelSource?.labelNames}
              externalLabelNamesLoading={externalLabelSource?.isLoading}
              externalFetchLabelValues={externalLabelSource?.fetchLabelValues}
              externalRefetchLabelNames={externalLabelSource?.refetchLabelNames}
              externalRefetchLabelValues={externalLabelSource?.refetchLabelValues}
            />
          ) : (
            <SimpleMatchers
              setMatchersString={setMatchersString}
              runQuery={setQueryExpression}
              currentQuery={query}
              profileType={selectedProfileName ?? profileType.toString()}
              queryBrowserRef={actualQueryBrowserRef}
              queryClient={queryClient}
              start={timeRangeSelection.getFromMs()}
              end={timeRangeSelection.getToMs()}
              searchExecutedTimestamp={searchExecutedTimestamp}
            />
          )}
        </div>

        {showSumBySelector && (
          <div {...testId(TEST_IDS.SUM_BY_CONTAINER)}>
            <div className="mb-0.5 mt-1.5 flex items-center justify-between">
              <label className="text-xs" {...testId(TEST_IDS.SUM_BY_LABEL)}>
                Sum by
              </label>
            </div>
            <SelectWithRefresh<SelectOption, true>
              id="h-sum-by-selector"
              data-testid={testId(TEST_IDS.SUM_BY_SELECT)['data-testid']}
              defaultValue={[]}
              isMulti
              isClearable={false}
              name="colors"
              options={labels.map(label => ({label, value: label}))}
              className="parca-select-container text-sm w-full max-w-80"
              classNamePrefix="parca-select"
              value={sumBySelection.map(sumBy => ({label: sumBy, value: sumBy}))}
              onChange={newValue => {
                setUserSumBySelection?.(newValue.map(option => option.value));
              }}
              placeholder="Labels..."
              styles={{
                indicatorSeparator: () => ({display: 'none'}),
                menu: provided => ({
                  ...provided,
                  marginBottom: 0,
                  boxShadow:
                    '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
                  marginTop: 10,
                  zIndex: 50,
                  minWidth: '320px',
                  position: 'absolute',
                }),
                // menu: provided => ({...provided, width: 'max-content', zIndex: 50}), // Setting the same zIndex as drop down menus
              }}
              isLoading={sumBySelectionLoading}
              isDisabled={!(profileType as ProfileType)?.delta}
              // @ts-expect-error
              ref={sumByRef}
              onKeyDown={e => {
                const currentRef = sumByRef?.current as unknown as SelectInstance | null;
                if (currentRef == null) {
                  return;
                }
                const inputRef = currentRef.inputRef;
                if (inputRef == null) {
                  return;
                }

                if (
                  e.key === 'Enter' &&
                  inputRef.value === '' &&
                  currentRef.state.focusedOptionId === null // menu is not open
                ) {
                  setQueryExpression(true);
                  currentRef.blur();
                }
              }}
              onRefresh={refetchLabelNames}
              refreshTitle="Refresh label names"
              refreshTestId="sum-by-refresh-button"
              menuTestId={TEST_IDS.SUM_BY_SELECT_FLYOUT}
            />
          </div>
        )}

        <DateTimeRangePicker
          onRangeSelection={setTimeRangeSelection}
          range={timeRangeSelection}
          timezone={timezone}
          {...testId(TEST_IDS.DATE_TIME_RANGE_PICKER)}
        />

        <div>
          <label className="text-xs" {...testId(TEST_IDS.SEARCH_BUTTON_LABEL)}>
            &nbsp;
          </label>
          <Button
            disabled={searchDisabled}
            onClick={(e: React.MouseEvent<HTMLElement>) => {
              e.preventDefault();
              setSearchExecutedTimestamp(Date.now());
              setQueryExpression(true);
            }}
            id="h-matcher-search-button"
            {...testId(TEST_IDS.SEARCH_BUTTON)}
          >
            Search
          </Button>
        </div>
      </div>
    </LabelProvider>
  );
}
