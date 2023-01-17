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

import {getLastItem} from '@parca/functions';
import {FlamegraphNode} from '@parca/client';
import {Mapping, Function, Location} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {hexifyAddress} from '../../utils';

export const getBinaryName = (
  location: Location,
  mappings: Mapping[],
  strings: string[]
): string | undefined => {
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
  functions: Function[]
): string {
  if (node.meta?.locationIndex === undefined) return '<unknown>';
  if (node.meta?.locationIndex === 0) return '<unknown>';

  const location = locations[node.meta.locationIndex - 1];

  const binary = getBinaryName(location, mappings, strings);

  const mappingString: string = binary != null ? `[${binary}]` : '';

  if (location.lines.length > 0) {
    const funcName =
      strings[functions[location.lines[node.meta.lineIndex].functionIndex - 1].nameStringIndex];
    return `${mappingString} ${funcName}`;
  }

  const address = hexifyAddress(location.address);
  const fallback = `${mappingString}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
}
