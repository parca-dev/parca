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

import {Table, tableFromIPC} from 'apache-arrow';
import {useContextMenu} from 'react-contexify';

import {FlamegraphArrow} from '@parca/client';
import {useURLState} from '@parca/components';
import {USER_PREFERENCES, useCurrentColorProfile, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {
  getColorForFeature,
  selectDarkMode,
  setHoveringNode,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {getLastItem, scaleLinear, selectQueryParam, type NavigateFunction} from '@parca/utilities';

import GraphTooltipArrow from '../../GraphTooltipArrow';
import GraphTooltipArrowContent from '../../GraphTooltipArrow/Content';
import {DockedGraphTooltip} from '../../GraphTooltipArrow/DockedGraphTooltip';
import {useProfileViewContext} from '../../ProfileView/ProfileViewContext';
import ColorStackLegend from './ColorStackLegend';
import ContextMenu from './ContextMenu';
import {IcicleNode, RowHeight, mappingColors} from './IcicleGraphNodes';
import {extractFeature} from './utils';

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
  mappings?: string[];
}

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
  mappings,
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

  const [, setBinaryFrameFilter] = useURLState({
    param: 'binary_frame_filter',
    navigateTo,
  });

  const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  const {compareMode} = useProfileViewContext();
  const isColorStackLegendEnabled = selectQueryParam('color_stack_legend') === 'true';
  const currentColorProfile = useCurrentColorProfile();
  const colorForSimilarNodes = currentColorProfile.colorForSimilarNodes;

  const mappingsList = useMemo(() => {
    const list =
      mappings
        ?.map(mapping => {
          return getLastItem(mapping) as string;
        })
        .flat() ?? [];

    // We add a EVERYTHING ELSE mapping to the list.
    list.push('');

    // We sort the mappings alphabetically to make sure that the order is always the same.
    list.sort((a, b) => a.localeCompare(b));

    return list;
  }, [mappings]);

  const mappingColors = useMemo(() => {
    const mappingFeatures = mappingsList.map(mapping => extractFeature(mapping));

    const colors: mappingColors = {};

    Object.entries(mappingFeatures).forEach(([_, feature]) => {
      colors[feature.name] = getColorForFeature(
        feature.name,
        isDarkMode,
        currentColorProfile.colors
      );
    });

    return colors;
  }, [mappingsList, isDarkMode, currentColorProfile]);

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width]);

  const xScale = useMemo(() => {
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
    const newMappingsList = mappingsList.filter(mapping => mapping !== binaryToRemove);

    setBinaryFrameFilter(newMappingsList);
  };

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
              path={[]}
              level={0}
              isRoot={true}
              searchString={currentSearchString}
              setHoveringRow={setHoveringRow}
              setHoveringLevel={setHoveringLevel}
              sortBy={sortBy}
              darkMode={isDarkMode}
              compareMode={compareMode}
              profileType={profileType}
              isContextMenuOpen={isContextMenuOpen}
              hoveringName={hoveringName}
              setHoveringName={setHoveringName}
              hoveringRow={hoveringRow}
              colorForSimilarNodes={colorForSimilarNodes}
              highlightSimilarStacksPreference={highlightSimilarStacksPreference}
            />
          </g>
        </g>
      </svg>
    );
  }, [
    compareMode,
    curPath,
    currentSearchString,
    height,
    isDarkMode,
    profileType,
    mappingColors,
    setCurPath,
    sortBy,
    table,
    total,
    width,
    xScale,
    isContextMenuOpen,
    displayMenu,
    colorForSimilarNodes,
    highlightSimilarStacksPreference,
    hoveringName,
    hoveringRow,
  ]);

  if (table.numRows === 0 || width === undefined) {
    return <></>;
  }

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
        />
        {isColorStackLegendEnabled && (
          <ColorStackLegend
            mappingColors={mappingColors}
            navigateTo={navigateTo}
            compareMode={compareMode}
          />
        )}
        {dockedMetainfo ? (
          <DockedGraphTooltip
            table={table}
            row={hoveringRow}
            level={hoveringLevel ?? 0}
            total={total}
            totalUnfiltered={total + filtered}
            profileType={profileType}
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
