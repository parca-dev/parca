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

import * as d3 from 'd3';

// Cache to store color mappings
const colorCache = new Map<string, string>();

// Create a color scale using d3's category10 scheme
const colorScale = d3.scaleOrdinal(d3.schemeCategory10);

/**
 * Generates a consistent color for a series based on its identifying properties
 */
export function getSeriesColor(labels: Array<{name: string; value: string}>): string {
  // Create a key from all labels to ensure unique identification
  const key = labels
    .map(l => `${l.name}=${l.value}`)
    .sort()
    .join(',');

  // Return cached color if exists
  if (colorCache.has(key)) {
    return colorCache.get(key)!;
  }

  // Generate new color and cache it
  const color = colorScale(key);
  colorCache.set(key, color);
  return color;
}
