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

import {LabelSet} from '@parca/client';

// Determine which label keys have all-numeric values across every label set.
const getNumericKeys = (labelSets: LabelSet[]): Set<string> => {
  const numericKeys = new Set<string>();
  if (labelSets.length === 0) return numericKeys;

  const keyCandidates = new Set(labelSets[0].labels.map(l => l.name));
  for (const key of keyCandidates) {
    const allNumeric = labelSets.every(ls => {
      const label = ls.labels.find(l => l.name === key);
      return label != null && label.value !== '' && !isNaN(Number(label.value));
    });
    if (allNumeric) numericKeys.add(key);
  }
  return numericKeys;
};

// Get key order: text keys first (sorted), then numeric keys (sorted).
const getSortedKeys = (labelSets: LabelSet[], numericKeys: Set<string>): string[] => {
  const allKeys = new Set<string>();
  for (const ls of labelSets) {
    for (const l of ls.labels) allKeys.add(l.name);
  }
  return [...allKeys]
    .filter(k => !numericKeys.has(k))
    .sort()
    .concat([...numericKeys].sort());
};

// Format a LabelSet as a string with keys ordered: text first, then numeric.
export const labelSetToString = (labelSet: LabelSet | undefined, keyOrder?: string[]): string => {
  if (labelSet === undefined) return '{}';

  const labels =
    keyOrder != null
      ? keyOrder
          .map(key => labelSet.labels.find(l => l.name === key))
          .filter((l): l is {name: string; value: string} => l != null)
      : labelSet.labels;

  if (labels.length === 0) return '{}';

  return '{' + labels.map(l => `${l.name}: ${l.value}`).join(', ') + '}';
};

// Build a comparator for LabelSets: text keys first (for grouping), then numeric keys.
// Also returns the key order so labelSetToString can use the same ordering.
export const createLabelSetComparator = (
  labelSets: LabelSet[]
): {compare: (a: LabelSet, b: LabelSet) => number; keyOrder: string[]} => {
  const numericKeys = getNumericKeys(labelSets);
  const keyOrder = getSortedKeys(labelSets, numericKeys);

  const compare = (a: LabelSet, b: LabelSet): number => {
    const aMap = new Map(a.labels.map(l => [l.name, l.value]));
    const bMap = new Map(b.labels.map(l => [l.name, l.value]));
    for (const key of keyOrder) {
      const aVal = aMap.get(key) ?? '';
      const bVal = bMap.get(key) ?? '';
      if (numericKeys.has(key)) {
        const diff = Number(aVal) - Number(bVal);
        if (diff !== 0) return diff;
      } else {
        const cmp = aVal.localeCompare(bVal);
        if (cmp !== 0) return cmp;
      }
    }
    return 0;
  };

  return {compare, keyOrder};
};
