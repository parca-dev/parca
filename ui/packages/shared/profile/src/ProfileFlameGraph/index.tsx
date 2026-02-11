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

import React, {LegacyRef, ReactNode, useCallback, useEffect, useMemo, useState} from 'react';

import cx from 'classnames';
import {AnimatePresence, motion} from 'framer-motion';
import {useMeasure} from 'react-use';

import {FlamegraphArrow} from '@parca/client';
import {
  FlameGraphSkeleton,
  SandwichFlameGraphSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {TEST_IDS, testId} from '@parca/test-utils';
import {capitalizeOnlyFirstLetter, divide} from '@parca/utilities';

import {MergedProfileSource, ProfileSource} from '../ProfileSource';
import DiffLegend from '../ProfileView/components/DiffLegend';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {useProfileMetadata} from '../ProfileView/hooks/useProfileMetadata';
import {useVisualizationState} from '../ProfileView/hooks/useVisualizationState';
import {TimelineGuide} from '../TimelineGuide';
import {FlameGraphArrow} from './FlameGraphArrow';
import {CurrentPathFrame, boundsFromProfileSource} from './FlameGraphArrow/utils';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileFlameGraphProps {
  width: number;
  arrow?: FlamegraphArrow;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  profileSource: ProfileSource;
  curPathArrow: CurrentPathFrame[] | [];
  setNewCurPathArrow: (path: CurrentPathFrame[]) => void;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  error?: any;
  isHalfScreen: boolean;
  metadataMappingFiles?: string[];
  metadataLoading?: boolean;
  isFlameChart?: boolean;
  isInSandwichView?: boolean;
  isRenderedAsFlamegraph?: boolean;
  tooltipId?: string;
  maxFrameCount?: number;
  isExpanded?: boolean;
}

const ErrorContent = ({errorMessage}: {errorMessage: string | ReactNode}): JSX.Element => {
  return (
    <div className="flex flex-col justify-center p-10 text-center gap-6 text-sm">
      {errorMessage}
    </div>
  );
};

export const validateFlameChartQuery = (
  profileSource: MergedProfileSource
): {isValid: boolean; isNonDelta: boolean; isDurationTooLong: boolean} => {
  const isNonDelta = !profileSource.ProfileType().delta;
  const duration = profileSource.mergeTo - profileSource.mergeFrom;
  const isDurationTooLong = duration > 60_000_000_000n; // 60 seconds in nanoseconds
  return {isValid: !isNonDelta && !isDurationTooLong, isNonDelta, isDurationTooLong};
};

const ProfileFlameGraph = function ProfileFlameGraphNonMemo({
  arrow,
  total,
  filtered,
  curPathArrow,
  setNewCurPathArrow,
  profileType,
  loading,
  error,
  width,
  isHalfScreen,
  metadataMappingFiles,
  isFlameChart = false,
  profileSource,
  isInSandwichView = false,
  isRenderedAsFlamegraph = false,
  tooltipId,
  maxFrameCount,
  isExpanded = false,
  metadataLoading = false,
}: ProfileFlameGraphProps): JSX.Element {
  const {onError, authenticationErrorMessage, isDarkMode, flamechartHelpText} = useParcaContext();
  const {compareMode} = useProfileViewContext();
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [flameChartRef, {height: flameChartHeight}] = useMeasure();
  const {colorBy, setColorBy} = useVisualizationState();

  // Create local state for paths when in sandwich view to avoid URL updates
  const [localCurPathArrow, setLocalCurPathArrow] = useState<CurrentPathFrame[]>([]);

  const setCurPathArrowWrapper = useCallback(
    (path: CurrentPathFrame[]) => {
      if (isInSandwichView) {
        setLocalCurPathArrow(path);
      } else {
        setNewCurPathArrow(path);
      }
    },
    [isInSandwichView, setNewCurPathArrow]
  );

  // Determine which paths to use based on isInSandwichView flag
  const effectiveCurPathArrow = isInSandwichView ? localCurPathArrow : curPathArrow;

  const {mappingsList, filenamesList} = useProfileMetadata({
    flamegraphArrow: arrow,
    metadataMappingFiles,
    metadataLoading,
    colorBy,
  });

  // By default, we want delta profiles (CPU) to be relatively compared.
  // For non-delta profiles, like goroutines or memory, we want the profiles to be compared absolutely.
  const compareAbsoluteDefault = profileType?.delta === false ? 'true' : 'false';

  const [compareAbsolute = compareAbsoluteDefault] = useURLState('compare_absolute');
  const isCompareAbsolute = compareAbsolute === 'true';

  const mappingsListCount = useMemo(
    () => mappingsList.filter(m => m !== '').length,
    [mappingsList]
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
    if (arrow === undefined) {
      return ['0', '0', false, '0', '0', false, '0', '0'];
    }

    const trimmed: bigint = arrow?.trimmed ?? 0n;

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
  }, [arrow, filtered, total]);

  const loadingState = !loading && arrow !== undefined && metadataMappingFiles !== undefined;

  // If there is only one mapping file, we want to color by filename by default.
  useEffect(() => {
    if (mappingsListCount === 1 && colorBy !== 'filename') {
      setColorBy('filename');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mappingsListCount]);

  useEffect(() => {
    if (loadingState) {
      setIsLoading(false);
    } else {
      setIsLoading(true);
    }
  }, [loadingState]);

  const flameGraph = useMemo(() => {
    const {
      isValid: isFlameChartValid,
      isNonDelta,
      isDurationTooLong,
    } = isFlameChart
      ? validateFlameChartQuery(profileSource as MergedProfileSource)
      : {isValid: true, isNonDelta: false, isDurationTooLong: false};
    const isInvalidFlameChartQuery = isFlameChart && !isFlameChartValid;

    if (isLoading && !isInvalidFlameChartQuery) {
      return (
        <div className="h-auto overflow-clip" data-testid={TEST_IDS.FLAMEGRAPH_SKELETON}>
          {isRenderedAsFlamegraph ? (
            <SandwichFlameGraphSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
          ) : (
            <FlameGraphSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
          )}
        </div>
      );
    }

    // Do necessary checks to ensure that flame chart can be rendered for this query.
    if (isInvalidFlameChartQuery) {
      if (isNonDelta) {
        return (
          <ErrorContent
            errorMessage={
              <>
                <span>To use the Flame chart, please switch to a Delta profile.</span>
                {flamechartHelpText ?? null}
              </>
            }
          />
        );
      } else if (isDurationTooLong) {
        return (
          <ErrorContent
            errorMessage={
              <>
                <span>
                  Flame chart is unavailable for queries longer than one minute. Please select a
                  point in the metrics graph to continue.
                </span>
                {flamechartHelpText ?? null}
              </>
            }
          />
        );
      } else {
        return (
          <ErrorContent
            errorMessage={
              <>
                <span>The Flame chart is not available for this query.</span>
                {flamechartHelpText ?? null}
              </>
            }
          />
        );
      }
    }

    if (arrow === undefined) return <div className="mx-auto text-center">No data...</div>;

    if (total === 0n && !loading)
      return <div className="mx-auto text-center">Profile has no samples</div>;

    if (arrow !== undefined) {
      return (
        <div className="relative">
          {isFlameChart ? (
            <TimelineGuide
              bounds={boundsFromProfileSource(profileSource)}
              width={width}
              height={flameChartHeight ?? 420}
              margin={0}
              ticks={12}
              timeUnit="nanoseconds"
            />
          ) : null}
          <div ref={flameChartRef as LegacyRef<HTMLDivElement>}>
            <FlameGraphArrow
              width={width}
              arrow={arrow}
              total={total}
              filtered={filtered}
              curPath={effectiveCurPathArrow}
              setCurPath={setCurPathArrowWrapper}
              profileType={profileType}
              isHalfScreen={isHalfScreen}
              mappingsListFromMetadata={mappingsList}
              filenamesListFromMetadata={filenamesList}
              compareAbsolute={isCompareAbsolute}
              isFlameChart={isFlameChart}
              profileSource={profileSource}
              isRenderedAsFlamegraph={isRenderedAsFlamegraph}
              isInSandwichView={isInSandwichView}
              tooltipId={tooltipId}
              maxFrameCount={maxFrameCount}
              isExpanded={isExpanded}
              colorBy={colorBy}
            />
          </div>
        </div>
      );
    }
  }, [
    isLoading,
    arrow,
    total,
    loading,
    width,
    filtered,
    profileType,
    isHalfScreen,
    isDarkMode,
    isCompareAbsolute,
    isFlameChart,
    profileSource,
    flameChartHeight,
    flameChartRef,
    flamechartHelpText,
    isRenderedAsFlamegraph,
    isInSandwichView,
    effectiveCurPathArrow,
    setCurPathArrowWrapper,
    tooltipId,
    maxFrameCount,
    isExpanded,
    mappingsList,
    filenamesList,
    colorBy,
  ]);

  useEffect(() => {
    if (isTrimmed) {
      console.info(`Trimmed ${trimmedFormatted} (${trimmedPercentage}%) too small values.`);
    }
  }, [isTrimmed, trimmedFormatted, trimmedPercentage]);

  if (error != null) {
    onError?.(error);

    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    // Check for specific merge errors
    const errorMessageLower = error.message?.toLowerCase() ?? '';
    const isMergeError: boolean = errorMessageLower.includes('failed to merge flame chart records');
    const isTimestampError: boolean = errorMessageLower.includes(
      'multiple samples for the same timestamp is not allowed'
    );

    if (isMergeError || isTimestampError) {
      return (
        <ErrorContent
          errorMessage={
            <>
              <span className="font-semibold">Unable to display overlapping data</span>
              <span className="text-gray-600 dark:text-gray-400">
                The selected data contains overlapping samples from multiple nodes or threads that
                cannot be merged.
              </span>
              <span className="text-gray-600 dark:text-gray-400">
                To view this data, please apply more specific filters:
              </span>
              <ul className="list-disc list-inside text-left max-w-md mx-auto text-gray-600 dark:text-gray-400">
                <li>Select a specific node from the node selector</li>
                <li>Filter by either CPU or thread</li>
              </ul>
            </>
          }
        />
      );
    }

    return (
      <ErrorContent
        errorMessage={
          <>
            <span>{capitalizeOnlyFirstLetter(error.message)}</span>
            {isFlameChart ? flamechartHelpText ?? null : null}
          </>
        }
      />
    );
  }

  return (
    <AnimatePresence>
      <motion.div
        className="relative h-full w-full"
        key="flame-graph-loaded"
        initial={{opacity: 0}}
        animate={{opacity: 1}}
        transition={{duration: 0.5}}
      >
        {compareMode ? <DiffLegend /> : null}
        <div
          className={cx(!isInSandwichView ? 'min-h-48' : '')}
          id="h-flame-graph"
          {...testId(TEST_IDS.FLAMEGRAPH_CONTAINER)}
        >
          <>{flameGraph}</>
        </div>
        {!isInSandwichView && (
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
        )}
      </motion.div>
    </AnimatePresence>
  );
};

export default ProfileFlameGraph;
