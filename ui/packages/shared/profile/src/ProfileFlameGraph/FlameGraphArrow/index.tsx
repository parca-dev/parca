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

import React, {
  memo,
  useCallback,
  useDeferredValue,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';

import {Dictionary, Table, Vector, tableFromIPC} from 'apache-arrow';
import {useContextMenu} from 'react-contexify';

import {FlamegraphArrow} from '@parca/client';
import {useParcaContext, useURLState} from '@parca/components';
import {USER_PREFERENCES, useCurrentColorProfile, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {getColorForFeature, selectDarkMode, useAppSelector} from '@parca/store';
import {getLastItem, type ColorConfig} from '@parca/utilities';

import {ProfileSource} from '../../ProfileSource';
import {useProfileFilters} from '../../ProfileView/components/ProfileFilters/useProfileFilters';
import {useProfileViewContext} from '../../ProfileView/context/ProfileViewContext';
import ContextMenuWrapper, {ContextMenuWrapperRef} from './ContextMenuWrapper';
import {FlameNode, RowHeight, colorByColors} from './FlameGraphNodes';
import {MemoizedTooltip} from './MemoizedTooltip';
import {TooltipProvider} from './TooltipContext';
import {useFilenamesList} from './useMappingList';
import {useScrollViewport} from './useScrollViewport';
import {useVisibleNodes} from './useVisibleNodes';
import {
  CurrentPathFrame,
  arrowToString,
  extractFeature,
  extractFilenameFeature,
  getCurrentPathFrameData,
  isCurrentPathFrameMatch,
} from './utils';

export const FIELD_LABELS_ONLY = 'labels_only';
export const FIELD_MAPPING_FILE = 'mapping_file';
export const FIELD_MAPPING_BUILD_ID = 'mapping_build_id';
export const FIELD_LOCATION_ADDRESS = 'location_address';
export const FIELD_LOCATION_LINE = 'location_line';
export const FIELD_INLINED = 'inlined';
export const FIELD_TIMESTAMP = 'timestamp';
export const FIELD_DURATION = 'duration';
export const FIELD_GROUPBY_METADATA = 'groupby_metadata';
export const FIELD_FUNCTION_NAME = 'function_name';
export const FIELD_FUNCTION_SYSTEM_NAME = 'function_system_name';
export const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
export const FIELD_FUNCTION_START_LINE = 'function_startline';
export const FIELD_CHILDREN = 'children';
export const FIELD_LABELS = 'labels';
export const FIELD_CUMULATIVE = 'cumulative';
export const FIELD_FLAT = 'flat';
export const FIELD_DIFF = 'diff';
export const FIELD_PARENT = 'parent';
export const FIELD_DEPTH = 'depth';
export const FIELD_VALUE_OFFSET = 'value_offset';

interface FlameGraphArrowProps {
  arrow: FlamegraphArrow;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  profileSource: ProfileSource;
  width?: number;
  curPath: CurrentPathFrame[];
  setCurPath: (path: CurrentPathFrame[]) => void;
  isHalfScreen: boolean;
  mappingsListFromMetadata: string[];
  compareAbsolute: boolean;
  isFlameChart?: boolean;
  isRenderedAsFlamegraph?: boolean;
  isInSandwichView?: boolean;
  tooltipId?: string;
  maxFrameCount?: number;
  isExpanded?: boolean;
}

export const getMappingColors = (
  mappingsList: string[],
  isDarkMode: boolean,
  currentColorProfile: ColorConfig
): colorByColors => {
  const mappingFeatures = mappingsList.map(mapping => extractFeature(mapping));

  const colors: colorByColors = {};
  Object.entries(mappingFeatures).forEach(([_, feature]) => {
    colors[feature.name] = getColorForFeature(feature.name, isDarkMode, currentColorProfile.colors);
  });
  return colors;
};

export const getFilenameColors = (
  filenamesList: string[],
  isDarkMode: boolean,
  currentColorProfile: ColorConfig
): colorByColors => {
  const filenameFeatures = filenamesList.map(filename => extractFilenameFeature(filename));

  const colors: colorByColors = {};
  Object.entries(filenameFeatures).forEach(([_, feature]) => {
    colors[feature.name] = getColorForFeature(feature.name, isDarkMode, currentColorProfile.colors);
  });
  return colors;
};

const noop = (): void => {};

function getMaxDepth(depthColumn: Vector<any> | null): number {
  if (depthColumn === null) return 0;

  let max = 0;
  for (const val of depthColumn) {
    const numVal = Number(val);
    if (numVal > max) max = numVal;
  }
  return max;
}

export const FlameGraphArrow = memo(function FlameGraphArrow({
  arrow,
  total,
  filtered,
  width,
  setCurPath,
  curPath,
  profileType,
  profileSource,
  compareAbsolute,
  isFlameChart = false,
  isRenderedAsFlamegraph = false,
  isInSandwichView = false,
  tooltipId = 'default',
  maxFrameCount,
  isExpanded = false,
}: FlameGraphArrowProps): React.JSX.Element {
  const [highlightSimilarStacksPreference] = useUserPreference<boolean>(
    USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS.key
  );
  const [hoveringRow, setHoveringRow] = useState<number | undefined>(undefined);
  const [dockedMetainfo] = useUserPreference<boolean>(USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key);
  const isDarkMode = useAppSelector(selectDarkMode);
  const {perf} = useParcaContext();

  const table: Table<any> = useMemo(() => {
    const result = tableFromIPC(arrow.record);

    if (perf?.setMeasurement != null) {
      perf.setMeasurement('flamegraph.node_count', result.numRows);
    }

    return result;
  }, [arrow, perf]);
  const svg = useRef(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const renderStartTime = useRef<number>(0);

  const [svgElement, setSvgElement] = useState<SVGSVGElement | null>(null);

  const {excludeBinary} = useProfileFilters();

  const {compareMode} = useProfileViewContext();
  const currentColorProfile = useCurrentColorProfile();
  const colorForSimilarNodes = currentColorProfile.colorForSimilarNodes;

  const [colorBy, _] = useURLState('color_by');
  const colorByValue = colorBy === undefined || colorBy === '' ? 'binary' : (colorBy as string);

  const filenamesList = useFilenamesList(table);

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

  const filenameColors = useMemo(() => {
    const colors = getFilenameColors(filenamesList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, filenamesList, currentColorProfile]);

  const mappingColors = useMemo(() => {
    const colors = getMappingColors(mappingsList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, mappingsList, currentColorProfile]);

  const colorByList = {
    filename: filenameColors,
    binary: mappingColors,
  };

  type ColorByKey = keyof typeof colorByList;

  const colorByColors: colorByColors = colorByList[colorByValue as ColorByKey];

  const MENU_ID = 'flame-graph-context-menu';
  const contextMenuRef = useRef<ContextMenuWrapperRef>(null);
  const {show, hideAll} = useContextMenu({
    id: MENU_ID,
  });
  const displayMenu = useCallback(
    (e: React.MouseEvent, row: number): void => {
      e.preventDefault();
      // Race condition fix: Use callback to ensure context menu shows only after
      // row state has been updated and propagated through the hook chain.
      // This prevents empty function names on first click.
      contextMenuRef.current?.setRow(row, () => {
        show({
          event: e,
        });
      });
    },
    [show]
  );

  const hideBinary = (binaryToRemove: string): void => {
    // Add a new frame filter to hide this binary using the new ProfileFilters system
    excludeBinary(binaryToRemove);
  };

  const handleRowClick = (row: number): void => {
    // Walk down the stack starting at row until we reach the root (row 0).
    const path: CurrentPathFrame[] = [];
    let currentRow = row;
    while (currentRow > 0) {
      const frame = getCurrentPathFrameData(table, currentRow);
      path.push(frame);
      currentRow = table.getChild(FIELD_PARENT)?.get(currentRow) ?? 0;
    }

    // Reverse the path so that the root is first.
    path.reverse();
    setCurPath(path);
  };

  const depthColumn = table.getChild(FIELD_DEPTH);
  const maxDepth = getMaxDepth(depthColumn);

  // Apply frame limit if maxFrameCount is provided and not expanded
  const effectiveDepth =
    maxFrameCount !== undefined && !isExpanded ? Math.min(maxDepth, maxFrameCount) : maxDepth;

  // Use deferred value to prevent UI blocking when expanding frames
  const deferredEffectiveDepth = useDeferredValue(effectiveDepth);

  const totalHeight = isInSandwichView
    ? deferredEffectiveDepth * RowHeight
    : (deferredEffectiveDepth + 1) * RowHeight;

  // Get the viewport of the container, this is used to determine which rows are visible.
  const viewport = useScrollViewport(containerRef);

  // To find the selected row, we must walk the current path and look at which
  // children of the current frame matches the path element exactly. Until the
  // end, the row we find at the end is our selected row.
  let currentRow = 0;
  for (const frame of curPath) {
    let childRows: number[] = Array.from(table.getChild(FIELD_CHILDREN)?.get(currentRow) ?? []);
    if (childRows.length === 0) {
      // If there are no children, we can stop here.
      break;
    }
    childRows = childRows.filter(c => isCurrentPathFrameMatch(table, c, frame));
    if (childRows.length === 0) {
      // If there are no children that match the current path frame, we can stop here.
      break;
    }
    if (childRows.length > 1) {
      // If there are multiple children that match the current path frame, we can stop here.
      // This is a case where the path is ambiguous and we cannot determine a single row.
      break;
    }
    // If there is exactly one child that matches the current path frame, we can continue.
    currentRow = childRows[0];
  }
  const selectedRow = currentRow;

  const visibleNodes = useVisibleNodes({
    table,
    viewport,
    total,
    width: width ?? 1,
    selectedRow,
    effectiveDepth: deferredEffectiveDepth,
  });

  useEffect(() => {
    if (perf?.markInteraction != null) {
      renderStartTime.current = performance.now();
    }
  }, [table, width, curPath, perf]);

  useEffect(() => {
    setSvgElement(svg.current);
  }, [tooltipId]);

  return (
    <TooltipProvider
      table={table}
      total={total}
      totalUnfiltered={total + filtered}
      profileType={profileType}
      unit={arrow.unit}
      compareAbsolute={compareAbsolute}
      tooltipId={tooltipId}
    >
      <div className="relative">
        <ContextMenuWrapper
          ref={contextMenuRef}
          menuId={MENU_ID}
          table={table}
          total={total}
          totalUnfiltered={total + filtered}
          compareAbsolute={compareAbsolute}
          resetPath={() => setCurPath([])}
          hideMenu={hideAll}
          hideBinary={hideBinary}
          unit={arrow.unit}
          profileType={profileType}
          isInSandwichView={isInSandwichView}
        />
        <MemoizedTooltip contextElement={svgElement} dockedMetainfo={dockedMetainfo} />
        <div
          ref={containerRef}
          className="overflow-auto scrollbar-thin scrollbar-thumb-gray-400 scrollbar-track-gray-100 dark:scrollbar-thumb-gray-600 dark:scrollbar-track-gray-800 will-change-transform scroll-smooth webkit-overflow-scrolling-touch contain"
          style={{
            width: width ?? '100%',
            contain: 'layout style paint',
          }}
        >
          <svg
            className="font-robotoMono"
            width={width ?? 0}
            height={totalHeight}
            preserveAspectRatio="xMinYMid"
            ref={svg}
          >
            {visibleNodes.map(row => (
              <FlameNode
                key={row}
                table={table}
                row={row}
                colors={colorByColors}
                colorBy={colorByValue}
                totalWidth={width ?? 1}
                height={RowHeight}
                darkMode={isDarkMode}
                compareMode={compareMode}
                colorForSimilarNodes={colorForSimilarNodes}
                selectedRow={selectedRow}
                onClick={() => {
                  if (isFlameChart) {
                    // We don't want to expand in flame charts.
                    return;
                  }
                  handleRowClick(row);
                }}
                onContextMenu={displayMenu}
                hoveringRow={highlightSimilarStacksPreference ? hoveringRow : undefined}
                setHoveringRow={highlightSimilarStacksPreference ? setHoveringRow : noop}
                isFlameChart={isFlameChart}
                profileSource={profileSource}
                isRenderedAsFlamegraph={isRenderedAsFlamegraph}
                isInSandwichView={isInSandwichView}
                maxDepth={maxDepth}
                effectiveDepth={deferredEffectiveDepth}
                tooltipId={tooltipId}
              />
            ))}
          </svg>
        </div>
      </div>
    </TooltipProvider>
  );
});

export default FlameGraphArrow;
