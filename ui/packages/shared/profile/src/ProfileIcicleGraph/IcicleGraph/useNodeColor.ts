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

import {FlamegraphNode} from '@parca/client';
import {Mapping, Function, Location} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {COLOR_PROFILES, diffColor} from '@parca/functions';
import useUserPreference, {USER_PREFERENCES} from '@parca/functions/useUserPreference';
import {
  selectDarkMode,
  selectStackColors,
  useAppDispatch,
  useAppSelector,
  generateColorForFeature,
} from '@parca/store';
import {memo, useEffect, useMemo} from 'react';
import {getBinaryName, nodeLabel} from './utils';

const extractFeature = (
  data: FlamegraphNode,
  mappings: Mapping[],
  locations: Location[],
  strings: string[],
  functions: Function[]
): string => {
  const name = nodeLabel(data, strings, mappings, locations, functions).trim();
  if (name.startsWith('runtime') || name === 'root') {
    return 'runtime';
  }

  const binaryName = getBinaryName(data, mappings, locations, strings);
  if (binaryName != null) {
    return binaryName;
  }

  return 'NA';
};

interface Props {
  data: FlamegraphNode;
  strings: string[];
  mappings: Mapping[];
  locations: Location[];
  functions: Function[];
}

const useNodeColor = ({data, strings, mappings, locations, functions}: Props): string => {
  const colors = useAppSelector(selectStackColors);
  const [colorProfile] = useUserPreference<string>(USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key);
  const dispatch = useAppDispatch();
  const isDarkMode = useAppSelector(selectDarkMode);
  const name = nodeLabel(data, strings, mappings, locations, functions).trim();
  const feature = useMemo(
    function extractFeatureMemo() {
      return extractFeature(data, mappings, locations, strings, functions);
    },
    [data, strings, mappings, locations, functions]
  );

  useEffect(
    function useNodeColorEffect() {
      if (colors[feature] == null) {
        dispatch(generateColorForFeature({feature, colorProfileName: colorProfile}));
      }
    },
    [colors, feature, colorProfile, dispatch]
  );

  const color: string = useMemo(() => {
    const diff = parseFloat(data.diff);
    const cumulative = parseFloat(data.cumulative);
    // eslint-disable-next-line no-constant-condition
    if (Math.abs(diff) > 0) {
      const featureColor = diffColor(diff, cumulative, isDarkMode, name, colorProfile);
      return featureColor.color;
    }

    const color = colors[feature];
    return color;
  }, [colors, colorProfile, data, feature, isDarkMode, name]);

  return color;
};

export default useNodeColor;
