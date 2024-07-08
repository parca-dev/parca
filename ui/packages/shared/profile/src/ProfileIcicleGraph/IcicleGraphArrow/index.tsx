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

import React, {memo, useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {Dictionary, Table, Vector, tableFromIPC} from 'apache-arrow';
import {useContextMenu} from 'react-contexify';

import {FlamegraphArrow} from '@parca/client';
import {useURLState, useURLStateNew} from '@parca/components';
import {USER_PREFERENCES, useCurrentColorProfile, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {
  getColorForFeature,
  selectDarkMode,
  setHoveringNode,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {
  getLastItem,
  scaleLinear,
  selectQueryParam,
  type ColorConfig,
  type NavigateFunction,
} from '@parca/utilities';

import GraphTooltipArrow from '../../GraphTooltipArrow';
import GraphTooltipArrowContent from '../../GraphTooltipArrow/Content';
import {DockedGraphTooltip} from '../../GraphTooltipArrow/DockedGraphTooltip';
import {useProfileViewContext} from '../../ProfileView/ProfileViewContext';
import ContextMenu from './ContextMenu';
import {IcicleNode, RowHeight, mappingColors} from './IcicleGraphNodes';
import {arrowToString, extractFeature} from './utils';

export const FIELD_LABELS_ONLY = 'labels_only';
export const FIELD_MAPPING_FILE = 'mapping_file';
export const FIELD_MAPPING_BUILD_ID = 'mapping_build_id';
export const FIELD_LOCATION_ADDRESS = 'location_address';
export const FIELD_LOCATION_LINE = 'location_line';
export const FIELD_INLINED = 'inlined';
export const FIELD_FUNCTION_NAME = 'function_name';
export const FIELD_FUNCTION_SYSTEM_NAME = 'function_system_name';
export const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
export const FIELD_FUNCTION_START_LINE = 'function_startline';
export const FIELD_CHILDREN = 'children';
export const FIELD_LABELS = 'labels';
export const FIELD_CUMULATIVE = 'cumulative';
export const FIELD_CUMULATIVE_PER_SECOND = 'cumulative_per_second';
export const FIELD_FLAT = 'flat';
export const FIELD_FLAT_PER_SECOND = 'flat_per_second';
export const FIELD_DIFF = 'diff';
export const FIELD_DIFF_PER_SECOND = 'diff_per_second';

interface IcicleGraphArrowProps {
  arrow: FlamegraphArrow;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  sortBy: string;
  flamegraphLoading: boolean;
  isHalfScreen: boolean;
  mappingsListFromMetadata: string[];
}

export const getMappingColors = (
  mappingsList: string[],
  isDarkMode: boolean,
  currentColorProfile: ColorConfig
): mappingColors => {
  const mappingFeatures = mappingsList.map(mapping => extractFeature(mapping));

  const colors: mappingColors = {};
  Object.entries(mappingFeatures).forEach(([_, feature]) => {
    colors[feature.name] = getColorForFeature(feature.name, isDarkMode, currentColorProfile.colors);
  });
  return colors;
};

const noop = (): void => {};

export const IcicleGraphArrow = memo(function IcicleGraphArrow({
  arrow,
  total,
  filtered,
  width,
  setCurPath,
  curPath,
  profileType,
  navigateTo,
  sortBy,
  flamegraphLoading,
  mappingsListFromMetadata,
}: IcicleGraphArrowProps): React.JSX.Element {
  const [isContextMenuOpen, setIsContextMenuOpen] = useState<boolean>(false);
  const dispatch = useAppDispatch();
  const [highlightSimilarStacksPreference] = useUserPreference<boolean>(
    USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS.key
  );
  const [dockedMetainfo] = useUserPreference<boolean>(USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key);
  const isDarkMode = useAppSelector(selectDarkMode);

  const table: Table<any> = useMemo(() => {
    return tableFromIPC(arrow.record);
  }, [arrow]);

  const [height, setHeight] = useState(0);
  const [hoveringRow, setHoveringRow] = useState<number | null>(null);
  const [hoveringLevel, setHoveringLevel] = useState<number | null>(null);
  const [hoveringName, setHoveringName] = useState<string | null>(null);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  const [binaryFrameFilter, setBinaryFrameFilter] = useURLState({
    param: 'binary_frame_filter',
    navigateTo,
  });

  const [currentSearchString] = useURLStateNew('search_string');
  const {compareMode} = useProfileViewContext();
  const currentColorProfile = useCurrentColorProfile();
  const colorForSimilarNodes = currentColorProfile.colorForSimilarNodes;

  const mappingsList = useMemo(() => {
    // Read the mappings from the dictionary that contains all mapping strings.
    // This is great, as might only have a dozen or so mappings,
    // and don't need to read through all the rows (potentially thousands).
    const mappingsDict: Vector<Dictionary> | null = table.getChild(FIELD_MAPPING_FILE);
    const mappings =
      mappingsDict?.data
        .map(mapping => {
          if (mapping.dictionary == null) {
            return [];
          }
          const len = mapping.dictionary.length;
          const entries: string[] = [];
          for (let i = 0; i < len; i++) {
            const fn = arrowToString(mapping.dictionary.get(i));
            entries.push(getLastItem(fn) ?? '');
          }
          return entries;
        })
        .flat() ?? [];

    // We add a EVERYTHING ELSE mapping to the list.
    mappings.push('');

    // We sort the mappings alphabetically to make sure that the order is always the same.
    mappings.sort((a, b) => a.localeCompare(b));
    return mappings;
  }, [table]);

  const mappingColors = useMemo(() => {
    const colors = getMappingColors(mappingsList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, mappingsList, currentColorProfile]);

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width, flamegraphLoading]);

  const xScale = useMemo(() => {
    if (total === 0n) {
      return () => 0;
    }

    if (width === undefined) {
      return () => 0;
    }
    return scaleLinear([0n, total], [0, width]);
  }, [total, width]);

  const MENU_ID = 'icicle-graph-context-menu';
  const {show, hideAll} = useContextMenu({
    id: MENU_ID,
  });
  const displayMenu = useCallback(
    (e: React.MouseEvent): void => {
      show({
        event: e,
      });
    },
    [show]
  );

  const trackVisibility = (isVisible: boolean): void => {
    setIsContextMenuOpen(isVisible);
  };

  const hideBinary = (binaryToRemove: string): void => {
    // second/subsequent time filtering out a binary i.e. a binary has already been hidden
    // and we want to hide more binaries, we simply remove the binary from the binaryFrameFilter array in the URL.
    if (Array.isArray(binaryFrameFilter) && binaryFrameFilter.length > 0) {
      const newMappingsList = binaryFrameFilter.filter(mapping => mapping !== binaryToRemove);

      setBinaryFrameFilter(newMappingsList);
      return;
    }

    // first time hiding a binary
    const newMappingsList = mappingsListFromMetadata.filter(mapping => mapping !== binaryToRemove);
    setBinaryFrameFilter(newMappingsList);
  };

  const highlightSimilarStacksName = highlightSimilarStacksPreference ? hoveringName : null;
  const highlightSimilarStacksSetName = useMemo(() => {
    return highlightSimilarStacksPreference ? setHoveringName : noop;
  }, [highlightSimilarStacksPreference]);
  const highlightSimilarStacksSetLevel = useMemo(() => {
    return highlightSimilarStacksPreference ? setHoveringLevel : noop;
  }, [highlightSimilarStacksPreference]);
  const highlightSimilarStacksRow = highlightSimilarStacksPreference ? hoveringRow : null;
  const path = useMemo(() => {
    return [];
  }, []);

  // useMemo for the root graph as it otherwise renders the whole graph if the hoveringRow changes.
  const root = useMemo(() => {
    return (
      <svg
        className="font-robotoMono"
        width={width}
        height={height}
        preserveAspectRatio="xMinYMid"
        ref={svg}
        onContextMenu={displayMenu}
      >
        <g ref={ref}>
          <g transform={'translate(0, 0)'}>
            <IcicleNode
              table={table}
              row={0} // root is always row 0 in the arrow record
              mappingColors={mappingColors}
              x={0}
              y={0}
              totalWidth={width ?? 1}
              height={RowHeight}
              setCurPath={setCurPath}
              curPath={curPath}
              total={total}
              xScale={xScale}
              path={path}
              level={0}
              isRoot={true}
              searchString={(currentSearchString as string) ?? ''}
              setHoveringRow={setHoveringRow}
              setHoveringLevel={highlightSimilarStacksSetLevel}
              sortBy={sortBy}
              darkMode={isDarkMode}
              compareMode={compareMode}
              profileType={profileType}
              isContextMenuOpen={isContextMenuOpen}
              hoveringName={highlightSimilarStacksName}
              setHoveringName={highlightSimilarStacksSetName}
              hoveringRow={highlightSimilarStacksRow}
              colorForSimilarNodes={colorForSimilarNodes}
              highlightSimilarStacksPreference={highlightSimilarStacksPreference}
            />
          </g>
        </g>
      </svg>
    );
  }, [
    width,
    height,
    displayMenu,
    table,
    mappingColors,
    setCurPath,
    curPath,
    total,
    xScale,
    currentSearchString,
    sortBy,
    isDarkMode,
    compareMode,
    profileType,
    isContextMenuOpen,
    highlightSimilarStacksName,
    highlightSimilarStacksRow,
    colorForSimilarNodes,
    highlightSimilarStacksPreference,
    path,
    highlightSimilarStacksSetLevel,
    highlightSimilarStacksSetName,
  ]);

  return (
    <>
      <div onMouseLeave={() => dispatch(setHoveringNode(undefined))}>
        <ContextMenu
          menuId={MENU_ID}
          table={table}
          row={hoveringRow ?? 0}
          level={hoveringLevel ?? 0}
          total={total}
          totalUnfiltered={total + filtered}
          profileType={profileType}
          navigateTo={navigateTo as NavigateFunction}
          trackVisibility={trackVisibility}
          curPath={curPath}
          setCurPath={setCurPath}
          hideMenu={hideAll}
          hideBinary={hideBinary}
          unit={arrow.unit}
        />
        {dockedMetainfo ? (
          <DockedGraphTooltip
            table={table}
            row={hoveringRow}
            level={hoveringLevel ?? 0}
            total={total}
            totalUnfiltered={total + filtered}
            profileType={profileType}
            unit={arrow.unit}
          />
        ) : (
          !isContextMenuOpen && (
            <GraphTooltipArrow contextElement={svg.current} isContextMenuOpen={isContextMenuOpen}>
              <GraphTooltipArrowContent
                table={table}
                row={hoveringRow}
                level={hoveringLevel ?? 0}
                isFixed={false}
                total={total}
                totalUnfiltered={total + filtered}
                profileType={profileType}
                navigateTo={navigateTo as NavigateFunction}
                unit={arrow.unit}
              />
            </GraphTooltipArrow>
          )
        )}
        {root}
      </div>
    </>
  );
});

export default IcicleGraphArrow;
