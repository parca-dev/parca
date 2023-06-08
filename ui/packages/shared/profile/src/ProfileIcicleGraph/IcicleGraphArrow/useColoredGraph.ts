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

import {useEffect, useMemo} from 'react';

import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {
  Location,
  Mapping,
  Function as ParcaFunction,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {
  selectDarkMode,
  setFeatures,
  useAppDispatch,
  useAppSelector,
  type FeatureType,
  type FeaturesMap,
} from '@parca/store';
import type {ColorProfileName} from '@parca/utilities';

import {extractFeature} from './utils';

export interface ColoredFlamegraphNode extends FlamegraphNode {
  feature?: string;
  children: ColoredFlamegraphNode[];
}

export interface ColoredFlamgraphRootNode extends FlamegraphRootNode {
  feature?: string;
  children: ColoredFlamegraphNode[];
}

export interface ColoredFlamegraph extends Flamegraph {
  root: ColoredFlamgraphRootNode;
}

const colorNodes = (
  nodes: FlamegraphNode[] | undefined,
  strings: string[],
  mappings: Mapping[],
  locations: Location[],
  functions: ParcaFunction[],
  features: {[key: string]: FeatureType}
): ColoredFlamegraphNode[] => {
  if (nodes === undefined) {
    return [];
  }
  return nodes.map<ColoredFlamegraphNode>(node => {
    const coloredNode: ColoredFlamegraphNode = {
      ...node,
    };
    if (node.children != null) {
      coloredNode.children = colorNodes(
        node.children,
        strings,
        mappings,
        locations,
        functions,
        features
      );
    }
    const feature = extractFeature(node, mappings, locations, strings, functions);
    coloredNode.feature = feature.name;
    features[feature.name] = feature.type;
    return coloredNode;
  });
};

const useColoredGraph = (graph: Flamegraph): ColoredFlamegraph => {
  const dispatch = useAppDispatch();
  const [colorProfile] = useUserPreference<ColorProfileName>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isDarkMode = useAppSelector(selectDarkMode);

  const [coloredGraph, features]: [ColoredFlamegraph, FeaturesMap] = useMemo(() => {
    if (graph.root == null) {
      return [graph as ColoredFlamegraph, {}];
    }
    const features: FeaturesMap = {};
    const coloredGraph = {
      ...graph,
      root: {
        ...graph.root,
        children: colorNodes(
          graph.root.children ?? [],
          graph.stringTable,
          graph.mapping,
          graph.locations,
          graph.function,
          features
        ),
      },
    };
    return [coloredGraph, features];
  }, [graph]);

  useEffect(() => {
    dispatch(setFeatures({features, colorProfileName: colorProfile, isDarkMode}));
  }, [features, colorProfile, dispatch, isDarkMode]);

  return coloredGraph;
};

export default useColoredGraph;
