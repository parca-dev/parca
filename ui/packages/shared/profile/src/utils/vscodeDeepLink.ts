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

import {isPresetKey} from '../ProfileView/components/ProfileFilters/filterPresets';
import {type ProfileFilter} from '../ProfileView/components/ProfileFilters/useProfileFilters';

// Compact encoding mappings (same as useProfileFiltersUrlState)
const TYPE_MAP: Record<string, string> = {
  stack: 's',
  frame: 'f',
};

const FIELD_MAP: Record<string, string> = {
  function_name: 'fn',
  binary: 'b',
  system_name: 'sn',
  filename: 'f',
  address: 'a',
  line_number: 'ln',
};

const MATCH_MAP: Record<string, string> = {
  equal: '=',
  not_equal: '!=',
  contains: '~',
  not_contains: '!~',
  starts_with: '^',
  not_starts_with: '!^',
};

/**
 * Encode filters to compact string format for URL.
 */
export const encodeProfileFilters = (filters: ProfileFilter[]): string => {
  if (filters.length === 0) return '';

  return filters
    .filter(f => f.value !== '' && f.type != null)
    .map(f => {
      // Handle preset filters differently
      if (isPresetKey(f.type!)) {
        const presetKey = encodeURIComponent(f.type!);
        const value = encodeURIComponent(f.value);
        return `p:${presetKey}:${value}`;
      }

      // Handle regular filters
      const type = TYPE_MAP[f.type!];
      const field = FIELD_MAP[f.field!];
      const match = MATCH_MAP[f.matchType!];
      const value = encodeURIComponent(f.value);
      return `${type}:${field}:${match}:${value}`;
    })
    .join(',');
};

export interface VSCodeDeepLinkParams {
  expression_a?: string;
  time_selection_a?: string;
  from_a?: number; // Absolute timestamp (milliseconds)
  to_a?: number; // Absolute timestamp (milliseconds)
  profileFilters?: ProfileFilter[];
  filename?: string;
  buildId?: string;
  line?: number;
}

/**
 * Build a VS Code deep link URL for opening profiling data in the Polar Signals extension.
 *
 * URL format: vscode://parca.profiler/open?expression=...&time_selection=...
 */
export function buildVSCodeDeepLink(params: VSCodeDeepLinkParams): string {
  const searchParams = new URLSearchParams();

  console.log(params);

  if (params.expression_a != null && params.expression_a !== '') {
    searchParams.set('expression_a', params.expression_a);
  }

  if (params.time_selection_a != null && params.time_selection_a !== '') {
    searchParams.set('time_selection_a', params.time_selection_a);
  }

  if (params.from_a !== undefined) {
    searchParams.set('from_a', params.from_a.toString());
  }

  if (params.to_a !== undefined) {
    searchParams.set('to_a', params.to_a.toString());
  }

  if (params.profileFilters != null && params.profileFilters.length > 0) {
    const encoded = encodeProfileFilters(params.profileFilters);
    if (encoded != null && encoded !== '') {
      searchParams.set('profile_filters', encoded);
    }
  }

  if (params.filename != null && params.filename !== '') {
    searchParams.set('filename', params.filename);
  }

  if (params.buildId != null && params.buildId !== '') {
    searchParams.set('build_id', params.buildId);
  }

  if (params.line != null && params.line > 0) {
    searchParams.set('line', params.line.toString());
  }

  console.log(searchParams.toString());

  return `vscode://parca.parca-profiler/open?${searchParams.toString()}`;
}

/**
 * Attempt to open VS Code with the deep link.
 * Returns true if the link was triggered, false if it failed.
 */
export function openInVSCode(params: VSCodeDeepLinkParams): boolean {
  const url = buildVSCodeDeepLink(params);
  console.log(url);

  try {
    // Attempt to open the VS Code URI
    window.location.href = url;
    return true;
  } catch (error) {
    console.error('Failed to open VS Code deep link:', error);
    return false;
  }
}

/**
 * Copy the VS Code deep link to clipboard.
 */
export async function copyVSCodeDeepLink(params: VSCodeDeepLinkParams): Promise<boolean> {
  const url = buildVSCodeDeepLink(params);

  try {
    await navigator.clipboard.writeText(url);
    return true;
  } catch (error) {
    console.error('Failed to copy VS Code deep link:', error);
    return false;
  }
}
