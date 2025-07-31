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

import {useMemo} from 'react';

import {Switch} from '@headlessui/react';
import {RpcError} from '@protobuf-ts/runtime-rpc';
import Select, {type SelectInstance} from 'react-select';

import {ProfileTypesResponse, QueryServiceClient} from '@parca/client';
import {Button, DateTimeRange, DateTimeRangePicker} from '@parca/components';
import {ProfileType, Query} from '@parca/parser';

import MatchersInput from '../MatchersInput';
import MatchersInputMask from '../MatchersInputMask';
import ProfileTypeSelector from '../ProfileTypeSelector';
import SimpleMatchers from '../SimpleMatchers';
import ViewMatchers from '../ViewMatchers';

interface SelectOption {
  label: string;
  value: string;
}

interface QueryControlsProps {
  showProfileTypeSelector: boolean;
  showSumBySelector: boolean;
  disableExplorativeQuerying: boolean;
  profileTypesData?: ProfileTypesResponse;
  profileTypesLoading: boolean;
  selectedProfileName: string;
  setProfileName: (name: string | undefined) => void;
  profileTypesError?: RpcError;
  viewComponent?: {
    disableProfileTypesDropdown?: boolean;
    disableExplorativeQuerying?: boolean;
    labelnames?: string[];
    createViewComponent?: React.ReactNode;
  };
  queryBrowserMode: string;
  setQueryBrowserMode: (mode: string) => void;
  advancedModeForQueryBrowser: boolean;
  setAdvancedModeForQueryBrowser: (mode: boolean) => void;
  setMatchersString: (matchers: string) => void;
  setQueryExpression: (updateTs?: boolean) => void;
  query: Query;
  queryBrowserRef: React.RefObject<HTMLDivElement>;
  timeRangeSelection: DateTimeRange;
  setTimeRangeSelection: (range: DateTimeRange) => void;
  searchDisabled: boolean;
  queryClient: QueryServiceClient;
  labels: string[];
  sumBySelection: string[];
  sumBySelectionLoading: boolean;
  setUserSumBySelection: (sumBy: string[]) => void;
  sumByRef: React.RefObject<SelectInstance>;
  profileType: ProfileType;
}

export function QueryControls({
  showProfileTypeSelector,
  profileTypesData,
  profileTypesLoading,
  selectedProfileName,
  setProfileName,
  viewComponent,
  setQueryBrowserMode,
  advancedModeForQueryBrowser,
  setAdvancedModeForQueryBrowser,
  setMatchersString,
  setQueryExpression,
  query,
  queryBrowserRef,
  timeRangeSelection,
  setTimeRangeSelection,
  searchDisabled,
  queryClient,
  labels,
  sumBySelection,
  sumBySelectionLoading,
  setUserSumBySelection,
  sumByRef,
  profileType,
  showSumBySelector,
  profileTypesError,
}: QueryControlsProps): JSX.Element {
  // Memoized styles for the sum by selector
  // This prevents unnecessary re-renders and recalculations of styles
  /* eslint-disable @typescript-eslint/no-explicit-any, @typescript-eslint/strict-boolean-expressions */
  const sumBySelectStyles = useMemo(
    () => ({
      indicatorSeparator: () => ({display: 'none'}),
      menu: (provided: any) => ({...provided, width: 'max-content', zIndex: 50}), // Setting the same zIndex as drop down menus
      control: (provided: any, state: any) => ({
        ...provided,
        borderColor: state.isFocused ? '#6366f1' : provided.borderColor,
        boxShadow: state.isFocused ? '0 0 0 1px #6366f1' : provided.boxShadow,
        '&:hover': {
          borderColor: state.isFocused ? '#6366f1' : provided.borderColor,
        },
      }),
    }),
    []
  );
  /* eslint-enable @typescript-eslint/no-explicit-any, @typescript-eslint/strict-boolean-expressions */

  return (
    <div className="flex w-full flex-wrap items-start gap-2">
      {showProfileTypeSelector && (
        <div className="max-w-56">
          <label className="text-xs">Profile type</label>
          <ProfileTypeSelector
            profileTypesData={profileTypesData}
            loading={profileTypesLoading}
            selectedKey={selectedProfileName}
            onSelection={setProfileName}
            error={profileTypesError}
            disabled={viewComponent?.disableProfileTypesDropdown}
          />
        </div>
      )}

      <div className="w-full flex-1 flex flex-col gap-1 mt-auto" ref={queryBrowserRef}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <label className="text-xs">Query</label>
            {viewComponent?.disableExplorativeQuerying !== true && (
              <>
                <Switch
                  checked={advancedModeForQueryBrowser}
                  onChange={() => {
                    setAdvancedModeForQueryBrowser(!advancedModeForQueryBrowser);
                    setQueryBrowserMode(advancedModeForQueryBrowser ? 'simple' : 'advanced');
                  }}
                  className={`${
                    advancedModeForQueryBrowser ? 'bg-indigo-600' : 'bg-gray-400 dark:bg-gray-800'
                  } relative inline-flex h-[20px] w-[44px] shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-white/75`}
                >
                  <span className="sr-only">Use setting</span>
                  <span
                    aria-hidden="true"
                    className={`${
                      advancedModeForQueryBrowser ? 'translate-x-6' : 'translate-x-0'
                    } pointer-events-none inline-block h-[16px] w-[16px] transform rounded-full bg-white shadow-lg ring-0 transition duration-200 ease-in-out`}
                  />
                </Switch>
                <label className="text-xs">Advanced Mode</label>
              </>
            )}
          </div>
          {viewComponent?.createViewComponent}
        </div>

        {viewComponent?.disableExplorativeQuerying === true &&
        viewComponent?.labelnames !== undefined &&
        viewComponent?.labelnames.length >= 1 ? (
          <MatchersInputMask
            disabled={selectedProfileName === ''}
            placeholder="Select a profile first to enter a filter..."
          >
            <ViewMatchers
              labelNames={viewComponent.labelnames}
              setMatchersString={setMatchersString}
              profileType={selectedProfileName}
              runQuery={setQueryExpression}
              currentQuery={query}
              queryClient={queryClient}
              start={timeRangeSelection.getFromMs()}
              end={timeRangeSelection.getToMs()}
            />
          </MatchersInputMask>
        ) : advancedModeForQueryBrowser ? (
          <MatchersInput
            setMatchersString={setMatchersString}
            runQuery={setQueryExpression}
            currentQuery={query}
            profileType={selectedProfileName}
            queryClient={queryClient}
            start={timeRangeSelection.getFromMs()}
            end={timeRangeSelection.getToMs()}
          />
        ) : (
          <MatchersInputMask
            disabled={selectedProfileName === ''}
            placeholder="Select a profile first to enter a filter..."
          >
            <SimpleMatchers
              key={query.toString()}
              setMatchersString={setMatchersString}
              runQuery={setQueryExpression}
              currentQuery={query}
              profileType={selectedProfileName}
              queryBrowserRef={queryBrowserRef}
              queryClient={queryClient}
              start={timeRangeSelection.getFromMs()}
              end={timeRangeSelection.getToMs()}
            />
          </MatchersInputMask>
        )}
      </div>

      {showSumBySelector && (
        <div>
          <label className="text-xs">Sum by</label>
          <Select<SelectOption, true>
            id="h-sum-by-selector"
            defaultValue={[]}
            isMulti
            isClearable={false}
            name="colors"
            options={labels.map(label => ({label, value: label}))}
            className="parca-select-container text-sm w-full max-w-80"
            classNamePrefix="parca-select"
            value={(sumBySelection ?? []).map(sumBy => ({label: sumBy, value: sumBy}))}
            onChange={newValue => {
              setUserSumBySelection(newValue.map(option => option.value));
            }}
            placeholder="Labels..."
            styles={sumBySelectStyles}
            isLoading={sumBySelectionLoading}
            isDisabled={!profileType.delta}
            // @ts-expect-error
            ref={sumByRef}
            onKeyDown={e => {
              const currentRef = sumByRef.current as unknown as SelectInstance | null;
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
          />
        </div>
      )}

      <DateTimeRangePicker onRangeSelection={setTimeRangeSelection} range={timeRangeSelection} />

      <div>
        <label className="text-xs">&nbsp;</label>
        <Button
          disabled={searchDisabled}
          onClick={(e: React.MouseEvent<HTMLElement>) => {
            e.preventDefault();
            setQueryExpression(true);
          }}
          id="h-matcher-search-button"
        >
          Search
        </Button>
      </div>
    </div>
  );
}
