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

import React, {Fragment, useCallback, useEffect, useMemo, useState} from 'react';

import {Menu, Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import {Table} from 'apache-arrow';

import {Flamegraph} from '@parca/client';
import {Button, Select, useURLState} from '@parca/components';
import {useContainerDimensions} from '@parca/hooks';
import {divide, selectQueryParam, type NavigateFunction} from '@parca/utilities';

import DiffLegend from '../components/DiffLegend';
import IcicleGraph from './IcicleGraph';
import IcicleGraphArrow, {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
} from './IcicleGraphArrow';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width?: number;
  graph?: Flamegraph;
  table?: Table<any>;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
}

const ProfileIcicleGraph = ({
  graph,
  table,
  total,
  filtered,
  curPath,
  setNewCurPath,
  sampleUnit,
  navigateTo,
  loading,
  setActionButtons,
}: ProfileIcicleGraphProps): JSX.Element => {
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';
  const {ref, dimensions} = useContainerDimensions();
  const [storeSortBy, setStoreSortBy] = useURLState({param: 'sort_by', navigateTo});
  const [localSortBy = FIELD_FUNCTION_NAME, setLocalSortBy] = useState(storeSortBy as string);
  const setSortBy = useCallback(
    (key: string) => {
      setLocalSortBy(key);
      setStoreSortBy(key);
    },
    [setLocalSortBy, setStoreSortBy]
  );

  const [storeGroupBy = [FIELD_FUNCTION_NAME], setStoreGroupBy] = useURLState({
    param: 'group_by',
    navigateTo,
  });
  const groupBy = useMemo(() => {
    if (storeGroupBy !== undefined) {
      if (typeof storeGroupBy === 'string') {
        return [storeGroupBy];
      }
      return storeGroupBy as string[];
    }
    return [FIELD_FUNCTION_NAME];
  }, [storeGroupBy]);

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );
  const toggleGroupBy = useCallback(
    (key: string): void => {
      groupBy.includes(key)
        ? setGroupBy(groupBy.filter(v => v !== key)) // remove
        : setGroupBy([...groupBy, key]); // add
    },
    [groupBy, setGroupBy]
  );

  const [
    totalFormatted,
    totalUnfilteredFormatted,
    isTrimmed,
    trimmedFormatted,
    trimmedPercentage,
    isFiltered,
    filteredPercentage,
  ] = useMemo(() => {
    if (graph === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    // const trimmed = graph.trimmed;
    const trimmed = 0n;

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
  }, [graph, filtered, total]);

  useEffect(() => {
    if (setActionButtons === undefined) {
      return;
    }
    setActionButtons(
      <div className="flex w-full justify-end gap-2 pb-2">
        <div className="flex w-full items-center justify-between space-x-2">
          {table !== undefined && (
            <>
              <GroupByDropdown groupBy={groupBy} toggleGroupBy={toggleGroupBy} />
              <SortBySelect compareMode={compareMode} sortBy={localSortBy} setSortBy={setSortBy} />
            </>
          )}
          <div>
            <label className="inline-block"></label>
            <Button
              color="neutral"
              onClick={() => setNewCurPath([])}
              disabled={curPath.length === 0}
              variant="neutral"
            >
              Reset View
            </Button>
          </div>
        </div>
      </div>
    );
  }, [setNewCurPath, curPath, setActionButtons, table, compareMode, localSortBy, groupBy]);

  if (graph === undefined && table === undefined) return <div>no data...</div>;

  if (total === 0n && !loading) return <>Profile has no samples</>;

  if (isTrimmed) {
    console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`);
  }

  return (
    <div className="relative">
      {compareMode && <DiffLegend />}
      <div ref={ref} className="min-h-48">
        {graph !== undefined && (
          <IcicleGraph
            width={dimensions?.width}
            graph={graph}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
          />
        )}
        {table !== undefined && (
          <IcicleGraphArrow
            width={dimensions?.width}
            table={table}
            total={total}
            filtered={filtered}
            curPath={curPath}
            setCurPath={setNewCurPath}
            sampleUnit={sampleUnit}
            navigateTo={navigateTo}
            sortBy={localSortBy}
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
  },
  {
    value: FIELD_LABELS,
    label: 'Labels',
    description: 'Stacktraces are grouped by pprof labels.',
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
          <Menu.Button className="relative w-full cursor-default rounded-md border bg-gray-50 py-2 pl-3 pr-10 text-left text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-900 sm:text-sm">
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
                  {groupByOptions.map(({value, label, description}) => (
                    <div key={value} className="relative flex items-start">
                      <div className="flex h-6 items-center">
                        <input
                          id={value}
                          name={value}
                          type="checkbox"
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                          checked={groupBy.includes(value)}
                          onChange={() => {
                            toggleGroupBy(value);
                          }}
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
