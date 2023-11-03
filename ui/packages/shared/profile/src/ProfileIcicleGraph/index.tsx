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

import React, {Fragment, useCallback, useEffect, useMemo} from 'react';

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';

import {Flamegraph, FlamegraphArrow} from '@parca/client';
import {Button, Select, useParcaContext, useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {
  capitalizeOnlyFirstLetter,
  divide,
  selectQueryParam,
  type NavigateFunction,
} from '@parca/utilities';

import DiffLegend from '../components/DiffLegend';
import IcicleGraph from './IcicleGraph';
import IcicleGraphArrow, {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from './IcicleGraphArrow';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width: number;
  graph?: Flamegraph;
  arrow?: FlamegraphArrow;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  error?: any;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return <div className="flex justify-center p-10">{errorMessage}</div>;
};

const ShowHideLegendButton = ({navigateTo}: {navigateTo?: NavigateFunction}): JSX.Element => {
  const [colorStackLegend, setStoreColorStackLegend] = useURLState({
    param: 'color_stack_legend',
    navigateTo,
  });

  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const isColorStackLegendEnabled = colorStackLegend === 'true';

  const [colorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );

  const setColorStackLegend = useCallback(
    (value: string): void => {
      setStoreColorStackLegend(value);
    },
    [setStoreColorStackLegend]
  );

  return (
    <>
      {colorProfileName === 'default' || compareMode ? null : (
        <Button
          className="gap-2"
          variant="neutral"
          onClick={() => setColorStackLegend(isColorStackLegendEnabled ? 'false' : 'true')}
        >
          {isColorStackLegendEnabled ? 'Hide legend' : 'Show legend'}
          <Icon icon={isColorStackLegendEnabled ? 'ph:eye-closed' : 'ph:eye'} width={20} />
        </Button>
      )}
    </>
  );
};

const GroupAndSortActionButtons = ({navigateTo}: {navigateTo?: NavigateFunction}): JSX.Element => {
  const [storeSortBy = FIELD_FUNCTION_NAME, setStoreSortBy] = useURLState({
    param: 'sort_by',
    navigateTo,
  });
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const [storeGroupBy = [FIELD_FUNCTION_NAME], setStoreGroupBy] = useURLState({
    param: 'group_by',
    navigateTo,
  });

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const groupBy = useMemo(() => {
    if (storeGroupBy !== undefined) {
      if (typeof storeGroupBy === 'string') {
        return [storeGroupBy];
      }
      return storeGroupBy;
    }
    return [FIELD_FUNCTION_NAME];
  }, [storeGroupBy]);

  const toggleGroupBy = useCallback(
    (key: string): void => {
      groupBy.includes(key)
        ? setGroupBy(groupBy.filter(v => v !== key)) // remove
        : setGroupBy([...groupBy, key]); // add
    },
    [groupBy, setGroupBy]
  );

  const [showRuntimeRubyStr, setShowRuntimeRuby] = useURLState({
    param: 'show_runtime_ruby',
    navigateTo,
  });

  const [showRuntimePythonStr, setShowRuntimePython] = useURLState({
    param: 'show_runtime_python',
    navigateTo,
  });

  const [showInterpretedOnlyStr, setShowInterpretedOnly] = useURLState({
    param: 'show_interpreted_only',
    navigateTo,
  });

  return (
    <>
      <GroupByDropdown groupBy={groupBy} toggleGroupBy={toggleGroupBy} />
      <SortBySelect
        compareMode={compareMode}
        sortBy={storeSortBy as string}
        setSortBy={setStoreSortBy}
      />
      <RuntimeFilterDropdown
        showRuntimeRuby={showRuntimeRubyStr === 'true'}
        toggleShowRuntimeRuby={() => setShowRuntimeRuby(showRuntimeRubyStr === 'true' ? 'false' : 'true')}
        showRuntimePython={showRuntimePythonStr === 'true'}
        toggleShowRuntimePython={() => setShowRuntimePython(showRuntimePythonStr === 'true' ? 'false' : 'true')}
        showInterpretedOnly={showInterpretedOnlyStr === 'true'}
        toggleShowInterpretedOnly={() => setShowInterpretedOnly(showInterpretedOnlyStr === 'true' ? 'false' : 'true')}
      />
    </>
  );
};

const RuntimeToggle = ({
  id,
  state,
  toggle,
  label,
  description,
}: {
  id: string;
  state: boolean;
  toggle: () => void;
  label: string;
  description: string;
}): JSX.Element => {
  return (
    <div key={id} className="relative flex items-start">
      <div className="flex h-6 items-center">
        <input
          id={id}
          name={id}
          type="checkbox"
          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
          checked={state}
          onChange={() => toggle()}
        />
      </div>
      <div className="ml-3 text-sm leading-6">
        <label htmlFor={id} className="font-medium text-gray-900">
          {label}
        </label>
        <p className="text-gray-500">{description}</p>
      </div>
    </div>
  )
}

const RuntimeFilterDropdown = ({
  showRuntimeRuby,
  toggleShowRuntimeRuby,
  showRuntimePython,
  toggleShowRuntimePython,
  showInterpretedOnly,
  toggleShowInterpretedOnly,
}: {
  showRuntimeRuby: boolean;
  toggleShowRuntimeRuby: () => void;
  showRuntimePython: boolean;
  toggleShowRuntimePython: () => void;
  showInterpretedOnly: boolean;
  toggleShowInterpretedOnly: () => void;
}): React.JSX.Element => {
  return (
    <div>
      <label className="text-sm">Runtimes</label>
      <Menu as="div" className="relative text-left">
        <div>
          <Menu.Button className="relative w-full cursor-default rounded-md border bg-white py-2 pl-3 pr-10 text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm">
            <span className="ml-3 block overflow-x-hidden text-ellipsis">Runtimes</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 ml-3 flex items-center pr-2 text-gray-400">
              <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
            </span>
          </Menu.Button>
        </div>

        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Menu.Items className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  <RuntimeToggle
                    id="show-runtime-ruby"
                    state={showRuntimeRuby}
                    toggle={toggleShowRuntimeRuby}
                    label="Ruby"
                    description="Show Ruby runtime functions."
                  />
                  <RuntimeToggle
                    id="show-runtime-python"
                    state={showRuntimePython}
                    toggle={toggleShowRuntimePython}
                    label="Python"
                    description="Show Python runtime functions."
                  />
                  <RuntimeToggle
                    id="show-interpreted-only"
                    state={showInterpretedOnly}
                    toggle={toggleShowInterpretedOnly}
                    label="Interpreted Only"
                    description="Show only interpreted functions."
                  />
                </div>
              </fieldset>
            </div>
          </Menu.Items>
        </Transition>
      </Menu>
    </div>
  );
};

const ProfileIcicleGraph = function ProfileIcicleGraphNonMemo({
  graph,
  arrow,
  total,
  filtered,
  curPath,
  setNewCurPath,
  sampleUnit,
  navigateTo,
  loading,
  setActionButtons,
  error,
  width,
}: ProfileIcicleGraphProps): JSX.Element {
  const {loader, onError, authenticationErrorMessage} = useParcaContext();
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const [storeSortBy = FIELD_FUNCTION_NAME] = useURLState({
    param: 'sort_by',
    navigateTo,
  });

  const [
    totalFormatted,
    totalUnfilteredFormatted,
    isTrimmed,
    trimmedFormatted,
    trimmedPercentage,
    isFiltered,
    filteredPercentage,
  ] = useMemo(() => {
    if (graph === undefined && arrow === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    const trimmed: bigint = graph?.trimmed ?? arrow?.trimmed ?? 0n;

    const totalUnfiltered = total + filtered;
    // safeguard against division by zero
    const totalUnfilteredDivisor = totalUnfiltered > 0 ? totalUnfiltered : 1n;

    return [
      numberFormatter.format(total),
      numberFormatter.format(totalUnfiltered),
      trimmed > 0,
      numberFormatter.format(trimmed),
      numberFormatter.format(divide(trimmed * 100n, totalUnfilteredDivisor)),
      filtered > 0,
      numberFormatter.format(divide(total * 100n, totalUnfilteredDivisor)),
    ];
  }, [graph, arrow, filtered, total]);

  useEffect(() => {
    if (setActionButtons === undefined) {
      return;
    }
    setActionButtons(
      <div className="flex w-full justify-end gap-2 pb-2">
        <div className="ml-2 flex w-full flex-col items-start justify-between gap-2 md:flex-row md:items-end">
          {arrow !== undefined && <GroupAndSortActionButtons navigateTo={navigateTo} />}
          <ShowHideLegendButton navigateTo={navigateTo} />
          <Button
            variant="neutral"
            onClick={() => setNewCurPath([])}
            disabled={curPath.length === 0}
          >
            Reset View
          </Button>
        </div>
      </div>
    );
  }, [navigateTo, arrow, curPath, setNewCurPath, setActionButtons]);

  if (loading) {
    return <div className="h-96">{loader}</div>;
  }

  if (error != null) {
    onError?.(error);

    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
  }

  if (graph === undefined && arrow === undefined)
    return <div className="mx-auto text-center">No data...</div>;

  if (total === 0n && !loading)
    return <div className="mx-auto text-center">Profile has no samples</div>;

  if (isTrimmed) {
    console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`);
  }

  return (
    <div className="relative">
      {compareMode && <DiffLegend />}
      <div className="min-h-48">
        {graph !== undefined && (
          <IcicleGraph
            width={width}
            graph={graph}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
          />
        )}
        {arrow !== undefined && (
          <IcicleGraphArrow
            width={width}
            arrow={arrow}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
            sortBy={storeSortBy as string}
          />
        )}
      </div>
      <p className="my-2 text-xs">
        Showing {totalFormatted}{' '}
        {isFiltered ? (
          <span>
            ({filteredPercentage}%) filtered of {totalUnfilteredFormatted}{' '}
          </span>
        ) : (
          <></>
        )}
        values.{' '}
      </p>
    </div>
  );
};

const groupByOptions = [
  {
    value: FIELD_FUNCTION_NAME,
    label: 'Function Name',
    description: 'Stacktraces are grouped by function names.',
    disabled: true,
  },
  {
    value: FIELD_LABELS,
    label: 'Labels',
    description: 'Stacktraces are grouped by pprof labels.',
    disabled: false,
  },
  {
    value: FIELD_FUNCTION_FILE_NAME,
    label: 'Filename',
    description: 'Stacktraces are grouped by filenames.',
    disabled: false,
  },
  {
    value: FIELD_LOCATION_ADDRESS,
    label: 'Address',
    description: 'Stacktraces are grouped by addresses.',
    disabled: false,
  },
  {
    value: FIELD_MAPPING_FILE,
    label: 'Binary',
    description: 'Stacktraces are grouped by binaries.',
    disabled: false,
  },
];

const GroupByDropdown = ({
  groupBy,
  toggleGroupBy,
}: {
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
}): React.JSX.Element => {
  const label =
    groupBy.length === 0
      ? 'Nothing'
      : groupBy.length === 1
      ? groupByOptions.find(option => option.value === groupBy[0])?.label
      : 'Multiple';

  return (
    <div>
      <label className="text-sm">Group</label>
      <Menu as="div" className="relative text-left">
        <div>
          <Menu.Button className="relative w-full cursor-default rounded-md border bg-white py-2 pl-3 pr-10 text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm">
            <span className="ml-3 block overflow-x-hidden text-ellipsis">{label}</span>
            <span className="pointer-events-none absolute inset-y-0 right-0 ml-3 flex items-center pr-2 text-gray-400">
              <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
            </span>
          </Menu.Button>
        </div>

        <Transition
          as={Fragment}
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <Menu.Items className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  {groupByOptions.map(({value, label, description, disabled}) => (
                    <div key={value} className="relative flex items-start">
                      <div className="flex h-6 items-center">
                        <input
                          id={value}
                          name={value}
                          type="checkbox"
                          disabled={disabled}
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                          checked={groupBy.includes(value)}
                          onChange={() => toggleGroupBy(value)}
                        />
                      </div>
                      <div className="ml-3 text-sm leading-6">
                        <label htmlFor={value} className="font-medium text-gray-900">
                          {label}
                        </label>
                        <p className="text-gray-500">{description}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </fieldset>
            </div>
          </Menu.Items>
        </Transition>
      </Menu>
    </div>
  );
};

const SortBySelect = ({
  sortBy,
  setSortBy,
  compareMode,
}: {
  sortBy: string;
  setSortBy: (key: string) => void;
  compareMode: boolean;
}): React.JSX.Element => {
  return (
    <div>
      <label className="text-sm">Sort</label>
      <Select
        items={[
          {
            key: FIELD_FUNCTION_NAME,
            disabled: false,
            element: {
              active: <>Function</>,
              expanded: (
                <>
                  <span>Function</span>
                </>
              ),
            },
          },
          {
            key: FIELD_CUMULATIVE,
            disabled: false,
            element: {
              active: <>Cumulative</>,
              expanded: (
                <>
                  <span>Cumulative</span>
                </>
              ),
            },
          },
          {
            key: FIELD_DIFF,
            disabled: !compareMode,
            element: {
              active: <>Diff</>,
              expanded: (
                <>
                  <span>Diff</span>
                </>
              ),
            },
          },
        ]}
        selectedKey={sortBy}
        onSelection={key => setSortBy(key)}
        placeholder={'Sort By'}
        primary={false}
        disabled={false}
      />
    </div>
  );
};

export default ProfileIcicleGraph;
