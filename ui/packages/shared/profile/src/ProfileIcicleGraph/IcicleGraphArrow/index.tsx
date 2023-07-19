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

import React, {memo, useEffect, useMemo, useRef, useState} from 'react';

import {Dictionary, Table, Vector} from 'apache-arrow';

import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
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
  type ColorProfileName,
  type NavigateFunction,
} from '@parca/utilities';

import GraphTooltipArrow from '../../GraphTooltipArrow';
import GraphTooltipArrowContent from '../../GraphTooltipArrow/Content';
import ColorStackLegend from './ColorStackLegend';
import {IcicleNode, RowHeight, mappingColors} from './IcicleGraphNodes';
import {extractFeature} from './utils';

export const FIELD_MAPPING_FILE = 'mapping_file';
export const FIELD_MAPPING_BUILD_ID = 'mapping_build_id';
export const FIELD_LOCATION_ADDRESS = 'location_address';
export const FIELD_LOCATION_LINE = 'location_line';
export const FIELD_FUNCTION_NAME = 'function_name';
export const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
export const FIELD_FUNCTION_START_LINE = 'function_startline';
export const FIELD_CHILDREN = 'children';
export const FIELD_LABELS = 'labels';
export const FIELD_CUMULATIVE = 'cumulative';
export const FIELD_DIFF = 'diff';

interface IcicleGraphArrowProps {
  table: Table<any>;
  total: bigint;
  filtered: bigint;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  navigateTo?: NavigateFunction;
  sortBy: string;
}

export const IcicleGraphArrow = memo(function IcicleGraphArrow({
  table,
  total,
  filtered,
  width,
  setCurPath,
  curPath,
  sampleUnit,
  navigateTo,
  sortBy,
}: IcicleGraphArrowProps): React.JSX.Element {
  const dispatch = useAppDispatch();
  const [colorProfile] = useUserPreference<ColorProfileName>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isDarkMode = useAppSelector(selectDarkMode);

  const [height, setHeight] = useState(0);
  const [hoveringRow, setHoveringRow] = useState<number | null>(null);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const mappings = useMemo(() => {
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
            entries.push(getLastItem(mapping.dictionary.get(i)) ?? '');
          }
          return entries;
        })
        .flat() ?? [];

    // We add a EVERYTHING ELSE mapping to the list.
    mappings.push('');

    // We look through the function names to find out if there's a runtime function.
    // Again, we only read through the dictionary, which is much faster than reading through all the rows.
    // We stop as soon as we find a runtime function.
    const functionNamesDict: Vector<Dictionary> | null = table.getChild(FIELD_FUNCTION_NAME);
    functionNamesDict?.data.forEach(fn => {
      if (fn.dictionary == null) {
        return;
      }
      const len = fn.dictionary.length;
      for (let i = 0; i < len; i++) {
        const fn: string | null = functionNamesDict?.get(i);
        if (fn?.startsWith('runtime') === true) {
          mappings.push('runtime');
          break;
        }
      }
    });

    // We sort the mappings alphabetically to make sure that the order is always the same.
    mappings.sort((a, b) => a.localeCompare(b));
    return mappings;
  }, [table]);

  // TODO: Somehow figure out how to add runtime to this, if stacks are present.
  // Potentially read the function name dictionary and check if it contains strings starting with runtime.
  const mappingFeatures = useMemo(() => {
    return mappings.map(mapping => extractFeature(mapping));
  }, [mappings]);

  // TODO: Unify with mappingFeatures
  const mappingColors = useMemo(() => {
    const colors: mappingColors = {};
    Object.entries(mappingFeatures).forEach(([_, feature]) => {
      colors[feature.name] = getColorForFeature(feature.name, isDarkMode, colorProfile);
    });
    return colors;
  }, [colorProfile, isDarkMode, mappingFeatures]);

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

  // useMemo for the root graph as it otherwise renders the whole graph if the hoveringRow changes.
  const root = useMemo(() => {
    return (
      <svg
        className="font-robotoMono"
        width={width}
        height={height}
        preserveAspectRatio="xMinYMid"
        ref={svg}
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
              sortBy={sortBy}
              darkMode={isDarkMode}
              compareMode={compareMode}
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
    mappingColors,
    setCurPath,
    sortBy,
    table,
    total,
    width,
    xScale,
  ]);

  if (table.numRows === 0 || width === undefined) {
    return <></>;
  }

  return (
    <div onMouseLeave={() => dispatch(setHoveringNode(undefined))}>
      <ColorStackLegend
        mappingColors={mappingColors}
        navigateTo={navigateTo}
        compareMode={compareMode}
      />
      <GraphTooltipArrow contextElement={svg.current}>
        <GraphTooltipArrowContent
          table={table}
          row={hoveringRow}
          isFixed={false}
          total={total}
          totalUnfiltered={total + filtered}
          unit={sampleUnit}
        />
      </GraphTooltipArrow>
      {root}
    </div>
  );
});

export default IcicleGraphArrow;
