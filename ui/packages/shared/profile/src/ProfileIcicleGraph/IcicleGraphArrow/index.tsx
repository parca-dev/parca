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
  FEATURE_TYPES,
  FeaturesMap,
  selectDarkMode,
  setHoveringNode,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {
  ColorProfileName,
  getLastItem,
  scaleLinear,
  selectQueryParam,
  type NavigateFunction,
} from '@parca/utilities';

import GraphTooltip from '../../GraphTooltip';
import ColorStackLegend from './ColorStackLegend';
import {IcicleNode, RowHeight} from './IcicleGraphNodes';

export const FIELD_MAPPING_FILE = 'mapping_file';
export const FIELD_MAPPING_BUILD_ID = 'mapping_build_id';
export const FIELD_LOCATION_ADDRESS = 'location_address';
export const FIELD_FUNCTION_NAME = 'function_name';
export const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
export const FIELD_CHILDREN = 'children';
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
}: IcicleGraphArrowProps): React.JSX.Element {
  const dispatch = useAppDispatch();
  const [colorProfile] = useUserPreference<ColorProfileName>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isDarkMode = useAppSelector(selectDarkMode);

  const [height, setHeight] = useState(0);
  const [sortBy, setSortBy] = useState(FIELD_FUNCTION_NAME);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  const currentSearchString = (selectQueryParam('search_string') as string) ?? '';
  const compareMode: boolean =
    selectQueryParam('compare_a') === 'true' && selectQueryParam('compare_b') === 'true';

  const mappings = useMemo(() => {
    const mappingsDict: Vector<Dictionary> | null = table.getChild(FIELD_MAPPING_FILE);
    return Array.from(mappingsDict?.data.values() ?? [])
      .map((mapping): string[] => {
        const dict = mapping.dictionary;
        const len = dict?.data.length ?? 0;
        const values: string[] = [];
        for (let i = 0; i <= len; i++) {
          // Read the value and only append the binaries last part - binary name
          values.push(getLastItem(dict?.get(i)) ?? '');
        }
        return values;
      })
      .flat()
      .sort((a, b) => a.localeCompare(b));
  }, [table]);

  const mappingFeatures = useMemo(() => {
    const features: FeaturesMap = {
      runtime: FEATURE_TYPES.Runtime,
    };
    mappings.forEach(mapping => {
      features[mapping] = FEATURE_TYPES.Binary;
    });
    return features;
  }, [mappings, colorProfile]);

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

  if (table.numRows === 0 || width === undefined) {
    return <></>;
  }

  return (
    <div onMouseLeave={() => dispatch(setHoveringNode(undefined))}>
      <select
        className="rounded-md border bg-gray-50 px-3 py-2 text-sm shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 dark:bg-gray-900"
        onChange={e => setSortBy(e.target.value)}
      >
        <option value="function_name">Function Name</option>
        <option value="cumulative">Cumulative</option>
        <option value="diff">Diff</option>
      </select>
      <ColorStackLegend
        mappingsFeatures={mappingFeatures}
        navigateTo={navigateTo}
        compareMode={compareMode}
      />
      <GraphTooltip
        table={table}
        unit={sampleUnit}
        total={total}
        totalUnfiltered={total + filtered}
        contextElement={svg.current}
      />
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
              x={0}
              y={0}
              totalWidth={width}
              height={RowHeight}
              setCurPath={setCurPath}
              curPath={curPath}
              total={total}
              xScale={xScale}
              path={[]}
              level={0}
              isRoot={true}
              searchString={currentSearchString}
              sortBy={sortBy}
              compareMode={compareMode}
            />
          </g>
        </g>
      </svg>
    </div>
  );
});

export default IcicleGraphArrow;
