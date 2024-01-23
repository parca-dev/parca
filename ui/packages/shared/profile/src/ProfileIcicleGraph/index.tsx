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

import React, {useCallback, useEffect, useMemo} from 'react';

import {Icon} from '@iconify/react';
import {AnimatePresence, motion} from 'framer-motion';

import {Flamegraph, FlamegraphArrow} from '@parca/client';
import {
  Button,
  IcicleActionButtonPlaceholder,
  IcicleGraphSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {capitalizeOnlyFirstLetter, divide, type NavigateFunction} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import DiffLegend from '../components/DiffLegend';
import GroupByDropdown from './ActionButtons/GroupByDropdown';
import RuntimeFilterDropdown from './ActionButtons/RuntimeFilterDropdown';
import SortBySelect from './ActionButtons/SortBySelect';
import IcicleGraph from './IcicleGraph';
import IcicleGraphArrow, {FIELD_FUNCTION_NAME} from './IcicleGraphArrow';

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
  isHalfScreen: boolean;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return <div className="flex justify-center p-10">{errorMessage}</div>;
};

const ShowHideLegendButton = ({navigateTo}: {navigateTo?: NavigateFunction}): JSX.Element => {
  const [colorStackLegend, setStoreColorStackLegend] = useURLState({
    param: 'color_stack_legend',
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

  return (
    <>
      {colorProfileName === 'default' || compareMode ? null : (
        <Button
          className="gap-2 w-max"
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
        toggleShowRuntimeRuby={() =>
          setShowRuntimeRuby(showRuntimeRubyStr === 'true' ? 'false' : 'true')
        }
        showRuntimePython={showRuntimePythonStr === 'true'}
        toggleShowRuntimePython={() =>
          setShowRuntimePython(showRuntimePythonStr === 'true' ? 'false' : 'true')
        }
        showInterpretedOnly={showInterpretedOnlyStr === 'true'}
        toggleShowInterpretedOnly={() =>
          setShowInterpretedOnly(showInterpretedOnlyStr === 'true' ? 'false' : 'true')
        }
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
  sampleUnit,
  navigateTo,
  loading,
  setActionButtons,
  error,
  width,
  isHalfScreen,
}: ProfileIcicleGraphProps): JSX.Element {
  const {onError, authenticationErrorMessage} = useParcaContext();
  const {compareMode} = useProfileViewContext();

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
    if (loading && setActionButtons !== undefined) {
      setActionButtons(<IcicleActionButtonPlaceholder />);
      return;
    }

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
            className="w-max"
            onClick={() => setNewCurPath([])}
            disabled={curPath.length === 0}
          >
            Reset View
          </Button>
        </div>
      </div>
    );
  }, [navigateTo, arrow, curPath, setNewCurPath, setActionButtons, loading]);

  if (loading) {
    return (
      <div className="h-auto overflow-clip">
        <IcicleGraphSkeleton isHalfScreen={isHalfScreen} />
      </div>
    );
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
    <AnimatePresence>
      <motion.div
        className="relative"
        key="icicle-graph-loaded"
        initial={{opacity: 0}}
        animate={{opacity: 1}}
        transition={{duration: 0.5}}
      >
        {compareMode ? <DiffLegend /> : null}
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
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileIcicleGraph;
