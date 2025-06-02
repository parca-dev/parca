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

import React, {LegacyRef, ReactNode, useEffect, useMemo, useState} from 'react';

import {AnimatePresence, motion} from 'framer-motion';
import {useMeasure} from 'react-use';

import {FlamegraphArrow} from '@parca/client';
import {IcicleGraphSkeleton, useParcaContext, useURLState} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {capitalizeOnlyFirstLetter, divide} from '@parca/utilities';

import {MergedProfileSource, ProfileSource} from '../ProfileSource';
import DiffLegend from '../ProfileView/components/DiffLegend';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {TimelineGuide} from '../TimelineGuide';
import {FIELD_FUNCTION_NAME, IcicleGraphArrow} from './IcicleGraphArrow';
import useMappingList from './IcicleGraphArrow/useMappingList';
import {CurrentPathFrame, boundsFromProfileSource} from './IcicleGraphArrow/utils';

const numberFormatter = new Intl.NumberFormat('en-US');

export type ResizeHandler = (width: number, height: number) => void;

interface ProfileIcicleGraphProps {
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
  isIcicleChart?: boolean;
}

const ErrorContent = ({errorMessage}: {errorMessage: string | ReactNode}): JSX.Element => {
  return (
    <div className="flex flex-col justify-center p-10 text-center gap-6 text-sm">
      {errorMessage}
    </div>
  );
};

export const validateIcicleChartQuery = (
  profileSource: MergedProfileSource
): {isValid: boolean; isNonDelta: boolean; isDurationTooLong: boolean} => {
  const isNonDelta = !profileSource.ProfileType().delta;
  const isDurationTooLong = profileSource.mergeTo - profileSource.mergeFrom > 60000;
  return {isValid: !isNonDelta && !isDurationTooLong, isNonDelta, isDurationTooLong};
};

const ProfileIcicleGraph = function ProfileIcicleGraphNonMemo({
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
  isIcicleChart = false,
  profileSource,
}: ProfileIcicleGraphProps): JSX.Element {
  const {onError, authenticationErrorMessage, isDarkMode, iciclechartHelpText} = useParcaContext();
  const {compareMode} = useProfileViewContext();
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const [icicleChartRef, {height: icicleChartHeight}] = useMeasure();

  const mappingsList = useMappingList(metadataMappingFiles);

  const [colorBy, setColorBy] = useURLState('color_by');

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

  const icicleGraph = useMemo(() => {
    const {
      isValid: isIcicleChartValid,
      isNonDelta,
      isDurationTooLong,
    } = isIcicleChart
      ? validateIcicleChartQuery(profileSource as MergedProfileSource)
      : {isValid: true, isNonDelta: false, isDurationTooLong: false};
    const isInvalidIcicleChartQuery = isIcicleChart && !isIcicleChartValid;
    if (isLoading && !isInvalidIcicleChartQuery) {
      return (
        <div className="h-auto overflow-clip">
          <IcicleGraphSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
        </div>
      );
    }

    // Do necessary checks to ensure that icicle chart can be rendered for this query.
    if (isInvalidIcicleChartQuery) {
      if (isNonDelta) {
        return (
          <ErrorContent
            errorMessage={
              <>
                <span>To use the Icicle chart, please switch to a Delta profile.</span>
                {iciclechartHelpText ?? null}
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
                  Icicle chart is unavailable for queries longer than one minute. Please select a
                  point in the metrics graph to continue.
                </span>
                {iciclechartHelpText ?? null}
              </>
            }
          />
        );
      } else {
        return (
          <ErrorContent
            errorMessage={
              <>
                <span>The Icicle chart is not available for this query.</span>
                {iciclechartHelpText ?? null}
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
          {isIcicleChart ? (
            <TimelineGuide
              bounds={boundsFromProfileSource(profileSource)}
              width={width}
              height={icicleChartHeight ?? 420}
              margin={0}
              ticks={12}
              timeUnit="nanoseconds"
            />
          ) : null}
          <div ref={icicleChartRef as LegacyRef<HTMLDivElement>}>
            <IcicleGraphArrow
              width={width}
              arrow={arrow}
              total={total}
              filtered={filtered}
              curPath={curPathArrow}
              setCurPath={setNewCurPathArrow}
              profileType={profileType}
              isHalfScreen={isHalfScreen}
              mappingsListFromMetadata={mappingsList}
              compareAbsolute={isCompareAbsolute}
              isIcicleChart={isIcicleChart}
              profileSource={profileSource}
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
    curPathArrow,
    setNewCurPathArrow,
    profileType,
    isHalfScreen,
    isDarkMode,
    mappingsList,
    isCompareAbsolute,
    isIcicleChart,
    profileSource,
    icicleChartHeight,
    icicleChartRef,
    iciclechartHelpText,
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

    return (
      <ErrorContent
        errorMessage={
          <>
            <span>{capitalizeOnlyFirstLetter(error.message)}</span>
            {isIcicleChart ? iciclechartHelpText ?? null : null}
          </>
        }
      />
    );
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
