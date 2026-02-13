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

import {useWindowSize} from 'react-use';

import {useParcaContext} from '@parca/components';

interface MetricsGraphDimensions {
  width: number;
  height: number;
  heightStyle: string;
  margin: number;
}

const MAX_HEIGHT = 402;
const MINI_VARIANT_HEIGHT = 180;
const margin = 50;

export const useMetricsGraphDimensions = (
  comparing: boolean,
  isMini = false
): MetricsGraphDimensions => {
  const maxHeight = isMini ? MINI_VARIANT_HEIGHT : MAX_HEIGHT;
  let {width} = useWindowSize();
  const {profileExplorer} = useParcaContext();
  if (profileExplorer == null) {
    return {
      width: 0,
      height: 0,
      heightStyle: '0',
      margin: 0,
    };
  }
  width = width - profileExplorer.PaddingX;
  if (comparing) {
    width = width / 2 - 32;
  }
  const height = isMini ? MINI_VARIANT_HEIGHT : Math.min(width / 2.5, maxHeight);
  const heightStyle = isMini
    ? `${MINI_VARIANT_HEIGHT + margin}px`
    : `min(${maxHeight + margin}px, ${
        comparing
          ? profileExplorer.metricsGraph.maxHeightStyle.compareMode
          : profileExplorer.metricsGraph.maxHeightStyle.default
      })`;
  return {
    width,
    height,
    heightStyle,
    margin,
  };
};
