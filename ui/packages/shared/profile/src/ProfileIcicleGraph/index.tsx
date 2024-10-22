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

import React, {useEffect, useMemo, useState} from 'react';

import {Table, tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';

import {Flamegraph, FlamegraphArrow} from '@parca/client';
import {IcicleGraphSkeleton, useParcaContext, useURLState} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {capitalizeOnlyFirstLetter, divide, selectQueryParam} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import DiffLegend from '../components/DiffLegend';
import {IcicleGraph} from './IcicleGraph';
import {FIELD_FUNCTION_NAME, IcicleGraphArrow} from './IcicleGraphArrow';
import ColorStackLegend from './IcicleGraphArrow/ColorStackLegend';
import useMappingList, {useFilenamesList} from './IcicleGraphArrow/useMappingList';

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
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  error?: any;
  isHalfScreen: boolean;
  metadataMappingFiles?: string[];
  metadataLoading?: boolean;
}

const ErrorContent = ({errorMessage}: {errorMessage: string}): JSX.Element => {
  return <div className="flex justify-center p-10">{errorMessage}</div>;
};

const ProfileIcicleGraph = function ProfileIcicleGraphNonMemo({
  graph,
  arrow,
  total,
  filtered,
  curPath,
  setNewCurPath,
  profileType,
  loading,
  error,
  width,
  isHalfScreen,
  metadataMappingFiles,
}: ProfileIcicleGraphProps): JSX.Element {
  const {onError, authenticationErrorMessage, isDarkMode} = useParcaContext();
  const {compareMode} = useProfileViewContext();
  const [isLoading, setIsLoading] = useState<boolean>(true);
  const isColorStackLegendEnabled = selectQueryParam('color_stack_legend') === 'true';

  const table: Table<any> | null = useMemo(() => {
    return arrow !== undefined ? tableFromIPC(arrow.record) : null;
  }, [arrow]);

  const mappingsList = useMappingList(metadataMappingFiles);
  const filenamesList = useFilenamesList(table);

  const [storeSortBy = FIELD_FUNCTION_NAME] = useURLState('sort_by');
  const [colorBy, _] = useURLState('color_by');

  // By default, we want delta profiles (CPU) to be relatively compared.
  // For non-delta profiles, like goroutines or memory, we want the profiles to be compared absolutely.
  const compareAbsoluteDefault = profileType?.delta === false ? 'true' : 'false';

  const [compareAbsolute = compareAbsoluteDefault] = useURLState('compare_absolute');
  const isCompareAbsolute = compareAbsolute === 'true';

  const colorByValue = colorBy === undefined || colorBy === '' ? 'binary' : (colorBy as string);

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

  const loadingState =
    !loading && (arrow !== undefined || graph !== undefined) && metadataMappingFiles !== undefined;

  useEffect(() => {
    if (loadingState) {
      setIsLoading(false);
    } else {
      setIsLoading(true);
    }
  }, [loadingState]);

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
          sortBy={storeSortBy as string}
          flamegraphLoading={isLoading}
          isHalfScreen={isHalfScreen}
          mappingsListFromMetadata={mappingsList}
          compareAbsolute={isCompareAbsolute}
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
    storeSortBy,
    isHalfScreen,
    isDarkMode,
    mappingsList,
    isCompareAbsolute,
  ]);

  if (error != null) {
    onError?.(error);

    if (authenticationErrorMessage !== undefined && error.code === 'UNAUTHENTICATED') {
      return <ErrorContent errorMessage={authenticationErrorMessage} />;
    }

    return <ErrorContent errorMessage={capitalizeOnlyFirstLetter(error.message)} />;
  }

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
            compareMode={compareMode}
            mappings={colorByValue === 'binary' ? mappingsList : filenamesList}
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
