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
import {
  Location,
  Mapping,
  Function as ParcaFunction,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {EVERYTHING_ELSE, FEATURE_TYPES, type Feature} from '@parca/store';
import {getLastItem} from '@parca/utilities';

import {hexifyAddress} from '../../utils';

export const getBinaryName = (
  node: FlamegraphNode,
  mappings: Mapping[],
  locations: Location[],
  strings: string[]
): string | undefined => {
  if (node.meta?.locationIndex === undefined || node.meta?.locationIndex === 0) {
    return undefined;
  }
  if (node.meta.locationIndex > locations.length) {
    return undefined;
  }

  const location = locations[node.meta.locationIndex - 1];

  if (location.mappingIndex === undefined || location.mappingIndex === 0) {
    return undefined;
  }
  const mapping = mappings[location.mappingIndex - 1];
  if (mapping == null || mapping.fileStringIndex == null) {
    return undefined;
  }

  const mappingFile = strings[mapping.fileStringIndex];
  return getLastItem(mappingFile);
};

export function nodeLabel(
  node: FlamegraphNode,
  strings: string[],
  mappings: Mapping[],
  locations: Location[],
  functions: ParcaFunction[],
  showBinaryName: boolean
): string {
  if (node.meta?.locationIndex === undefined) return '<unknown>';
  if (node.meta?.locationIndex === 0) return '<unknown>';

  if (node.meta.locationIndex > locations.length) {
    console.info('location index out of bounds', node.meta.locationIndex, locations.length);
    return '<unknown>';
  }

  const location = locations[node.meta.locationIndex - 1];
  if (location === undefined) return '<unknown>';

  let mappingString = '';

  if (showBinaryName) {
    const binary = getBinaryName(node, mappings, locations, strings);
    if (binary != null) mappingString = `[${binary}]`;
  }

  if (location.lines.length > 0) {
    const funcName =
      strings[functions[location.lines[node.meta.lineIndex].functionIndex - 1].nameStringIndex];
    return `${mappingString.length > 0 ? `${mappingString} ` : ''}${funcName}`;
  }

  const address = hexifyAddress(location.address);
  const fallback = `${mappingString}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
}

export const extractFeature = (
  data: FlamegraphNode,
  mappings: Mapping[],
  locations: Location[],
  strings: string[],
  functions: ParcaFunction[]
): Feature => {
  const name = nodeLabel(data, strings, mappings, locations, functions, false).trim();
  if (name.startsWith('runtime') || name === 'root') {
    return {name: 'runtime', type: FEATURE_TYPES.Runtime};
  }

  const binaryName = getBinaryName(data, mappings, locations, strings);
  if (binaryName != null) {
    return {name: binaryName, type: FEATURE_TYPES.Binary};
  }

  return {name: EVERYTHING_ELSE, type: FEATURE_TYPES.Misc};
};
