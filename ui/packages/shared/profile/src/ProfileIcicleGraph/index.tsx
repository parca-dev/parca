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

import React, {useCallback, useEffect, useMemo, useState} from 'react';

import {Icon} from '@iconify/react';
import {AnimatePresence, motion} from 'framer-motion';

import {Flamegraph, FlamegraphArrow} from '@parca/client';
import {
  Button,
  IcicleGraphSkeleton,
  IconButton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {
  capitalizeOnlyFirstLetter,
  divide,
  selectQueryParam,
  type NavigateFunction,
} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import DiffLegend from '../components/DiffLegend';
import GroupByDropdown from './ActionButtons/GroupByDropdown';
import SortBySelect from './ActionButtons/SortBySelect';
import IcicleGraph from './IcicleGraph';
import IcicleGraphArrow, {FIELD_FUNCTION_NAME} from './IcicleGraphArrow';
import ColorStackLegend from './IcicleGraphArrow/ColorStackLegend';
import useMappingList from './IcicleGraphArrow/useMappingList';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
  width: number;
  graph?: Flamegraph;
  arrow?: FlamegraphArrow;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  error?: any;
  isHalfScreen: boolean;
  mappings?: string[];
  mappingsLoading?: boolean;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return <div className="flex justify-center p-10">{errorMessage}</div>;
};

const ShowHideLegendButton = ({
  navigateTo,
  isHalfScreen,
}: {
  navigateTo?: NavigateFunction;
  isHalfScreen: boolean;
}): JSX.Element => {
  const [colorStackLegend, setStoreColorStackLegend] = useURLState({
    param: 'color_stack_legend',
    navigateTo,
  });
  const [binaryFrameFilter, setBinaryFrameFilter] = useURLState({
    param: 'binary_frame_filter',
    navigateTo,
  });

  const {compareMode} = useProfileViewContext();

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

  const resetLegend = (): void => {
    setBinaryFrameFilter([]);
  };

  return (
    <>
      {colorProfileName === 'default' || compareMode ? null : (
        <>
          {isHalfScreen ? (
            <>
              <IconButton
                className="rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 items-center flex border border-gray-200 dark:border-gray-600 dark:text-white justify-center !py-2 !px-3 cursor-pointer min-h-[38px]"
                icon={isColorStackLegendEnabled ? 'ph:eye-closed' : 'ph:eye'}
                toolTipText={isColorStackLegendEnabled ? 'Hide legend' : 'Show legend'}
                onClick={() => setColorStackLegend(isColorStackLegendEnabled ? 'false' : 'true')}
                id="h-show-legend-button"
              />
              {binaryFrameFilter !== undefined && binaryFrameFilter.length > 0 && (
                <IconButton
                  className="rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 items-center flex border border-gray-200 dark:border-gray-600 dark:text-white justify-center !py-2 !px-3 cursor-pointer min-h-[38px]"
                  icon="system-uicons:reset"
                  toolTipText="Reset the legend selection"
                  onClick={() => resetLegend()}
                  id="h-reset-legend-button"
                />
              )}
            </>
          ) : (
            <>
              <Button
                className="gap-2 w-max"
                variant="neutral"
                onClick={() => setColorStackLegend(isColorStackLegendEnabled ? 'false' : 'true')}
                id="h-show-legend-button"
              >
                {isColorStackLegendEnabled ? 'Hide legend' : 'Show legend'}
                <Icon icon={isColorStackLegendEnabled ? 'ph:eye-closed' : 'ph:eye'} width={20} />
              </Button>
              {binaryFrameFilter !== undefined && binaryFrameFilter.length > 0 && (
                <Button
                  className="gap-2 w-max"
                  variant="neutral"
                  onClick={() => resetLegend()}
                  id="h-reset-legend-button"
                >
                  Reset Legend
                  <Icon icon="system-uicons:reset" width={20} />
                </Button>
              )}
            </>
          )}
        </>
      )}
    </>
  );
};

const GroupAndSortActionButtons = ({navigateTo}: {navigateTo?: NavigateFunction}): JSX.Element => {
  const [storeSortBy = FIELD_FUNCTION_NAME, setStoreSortBy] = useURLState({
    param: 'sort_by',
    navigateTo,
  });
  const {compareMode} = useProfileViewContext();

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

  return (
    <>
      <GroupByDropdown groupBy={groupBy} toggleGroupBy={toggleGroupBy} />
      <SortBySelect
        compareMode={compareMode}
        sortBy={storeSortBy as string}
        setSortBy={setStoreSortBy}
      />
    </>
  );
};

const ProfileIcicleGraph = function ProfileIcicleGraphNonMemo({
  graph,
  arrow,
  total,
  filtered,
  curPath,
  setNewCurPath,
  profileType,
  navigateTo,
  loading,
  setActionButtons,
  error,
  width,
  isHalfScreen,
  mappings,
}: ProfileIcicleGraphProps): JSX.Element {
  const {onError, authenticationErrorMessage, isDarkMode} = useParcaContext();
  const {compareMode} = useProfileViewContext();
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const isColorStackLegendEnabled = selectQueryParam('color_stack_legend') === 'true';

  const mappingsList = useMappingList(mappings);

  const [storeSortBy = FIELD_FUNCTION_NAME] = useURLState({
    param: 'sort_by',
    navigateTo,
  });

  const [invertStack = '', setInvertStack] = useURLState({
    param: 'invert_call_stack',
    navigateTo,
  });

  const isInvert = invertStack === 'true';

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
    setActionButtons?.(
      <div className="flex w-full justify-end gap-2 pb-2">
        <div className="ml-2 flex w-full flex-col items-start justify-between gap-2 md:flex-row md:items-end">
          {<GroupAndSortActionButtons navigateTo={navigateTo} />}
          {isHalfScreen ? (
            <IconButton
              icon={isInvert ? 'ph:sort-ascending' : 'ph:sort-descending'}
              toolTipText={isInvert ? 'Original Call Stack' : 'Invert Call Stack'}
              onClick={() => setInvertStack(isInvert ? '' : 'true')}
              className="rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 items-center flex border border-gray-200 dark:border-gray-600 dark:text-white justify-center py-2 px-3 cursor-pointer min-h-[38px]"
            />
          ) : (
            <Button
              variant="neutral"
              className="gap-2 w-max"
              onClick={() => setInvertStack(isInvert ? '' : 'true')}
            >
              {isInvert ? 'Original Call Stack' : 'Invert Call Stack'}
              <Icon icon={isInvert ? 'ph:sort-ascending' : 'ph:sort-descending'} width={20} />
            </Button>
          )}
          <ShowHideLegendButton isHalfScreen={isHalfScreen} navigateTo={navigateTo} />
          {isHalfScreen ? (
            <IconButton
              icon="system-uicons:reset"
              disabled={curPath.length === 0}
              toolTipText="Reset View"
              onClick={() => setNewCurPath([])}
              className="rounded-md focus:outline-none focus:ring-2 focus:ring-offset-2 items-center flex border border-gray-200 dark:border-gray-600 dark:text-white justify-center py-2 px-3 cursor-pointer min-h-[38px]"
            />
          ) : (
            <Button
              variant="neutral"
              className="gap-2 w-max"
              onClick={() => setNewCurPath([])}
              disabled={curPath.length === 0}
            >
              Reset View
              <Icon icon="system-uicons:reset" width={20} />
            </Button>
          )}
        </div>
      </div>
    );
  }, [
    navigateTo,
    isInvert,
    setInvertStack,
    arrow,
    curPath,
    setNewCurPath,
    setActionButtons,
    loading,
    isHalfScreen,
    isLoading,
  ]);

  const loadingState =
    !loading && (arrow !== undefined || graph !== undefined) && mappings !== undefined;

  useEffect(() => {
    if (loadingState) {
      setIsLoading(false);
    } else {
      setIsLoading(true);
    }
  }, [loadingState]);

  if (error != null) {
    onError?.(error);

    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
  }

  // eslint-disable-next-line react-hooks/rules-of-hooks
  const icicleGraph = useMemo(() => {
    if (isLoading) {
      return (
        <div className="h-auto overflow-clip">
          <IcicleGraphSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
        </div>
      );
    }

    if (graph === undefined && arrow === undefined)
      return <div className="mx-auto text-center">No data...</div>;

    if (total === 0n && !loading)
      return <div className="mx-auto text-center">Profile has no samples</div>;

    if (graph !== undefined)
      return (
        <IcicleGraph
          width={width}
          graph={graph}
          total={total}
          filtered={filtered}
          curPath={curPath}
          setCurPath={setNewCurPath}
          profileType={profileType}
          navigateTo={navigateTo}
        />
      );

    if (arrow !== undefined)
      return (
        <IcicleGraphArrow
          width={width}
          arrow={arrow}
          total={total}
          filtered={filtered}
          curPath={curPath}
          setCurPath={setNewCurPath}
          profileType={profileType}
          navigateTo={navigateTo}
          sortBy={storeSortBy as string}
          flamegraphLoading={isLoading}
          isHalfScreen={isHalfScreen}
          mappingsListFromMetadata={mappingsList}
        />
      );
  }, [
    isLoading,
    graph,
    arrow,
    total,
    loading,
    width,
    filtered,
    curPath,
    setNewCurPath,
    profileType,
    navigateTo,
    storeSortBy,
    isHalfScreen,
    isDarkMode,
    mappingsList,
  ]);

  if (isTrimmed) {
    console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`);
  }

  return (
    <AnimatePresence>
      <motion.div
        className="relative h-full w-full"
        key="icicle-graph-loaded"
        initial={{opacity: 0}}
        animate={{opacity: 1}}
        transition={{duration: 0.5}}
      >
        {compareMode ? <DiffLegend /> : null}
        {isColorStackLegendEnabled && (
          <ColorStackLegend
            navigateTo={navigateTo}
            compareMode={compareMode}
            mappings={mappings}
            loading={isLoading}
          />
        )}
        <div className="min-h-48" id="h-icicle-graph">
          <>{icicleGraph}</>
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
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileIcicleGraph;
