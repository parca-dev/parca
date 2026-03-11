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

import {describe, expect, it} from 'vitest';

import {LabelSet} from '@parca/client';

import {createLabelSetComparator, labelSetToString} from './labelSetUtils';

const ls = (labels: Record<string, string>): LabelSet => ({
  labels: Object.entries(labels).map(([name, value]) => ({name, value})),
});

describe('createLabelSetComparator', () => {
  it('sorts numeric cpu values numerically', () => {
    const sets = [ls({cpu: '10'}), ls({cpu: '2'}), ls({cpu: '1'})];
    const {compare} = createLabelSetComparator(sets);
    const sorted = [...sets].sort(compare);
    expect(sorted.map(s => s.labels[0].value)).toEqual(['1', '2', '10']);
  });

  it('sorts text labels lexicographically', () => {
    const sets = [ls({node: 'charlie'}), ls({node: 'alpha'}), ls({node: 'bravo'})];
    const {compare} = createLabelSetComparator(sets);
    const sorted = [...sets].sort(compare);
    expect(sorted.map(s => s.labels[0].value)).toEqual(['alpha', 'bravo', 'charlie']);
  });

  it('sorts text labels before numeric labels', () => {
    const sets = [ls({node: 'b', cpu: '1'}), ls({node: 'a', cpu: '2'}), ls({node: 'a', cpu: '1'})];
    const {compare} = createLabelSetComparator(sets);
    const sorted = [...sets].sort(compare);
    expect(sorted.map(s => [s.labels[0].value, s.labels[1].value])).toEqual([
      ['a', '1'],
      ['a', '2'],
      ['b', '1'],
    ]);
  });

  it('treats key as text if any value is non-numeric', () => {
    const sets = [ls({cpu: '10'}), ls({cpu: '2'}), ls({cpu: 'all'})];
    const {compare} = createLabelSetComparator(sets);
    const sorted = [...sets].sort(compare);
    // Lexicographic: '10' < '2' < 'all'
    expect(sorted.map(s => s.labels[0].value)).toEqual(['10', '2', 'all']);
  });
});

describe('labelSetToString', () => {
  it('formats labels in key order: text first, numeric last', () => {
    const sets = [ls({cpu: '0', node: 'alpha'}), ls({cpu: '1', node: 'beta'})];
    const {keyOrder} = createLabelSetComparator(sets);
    expect(labelSetToString(sets[0], keyOrder)).toBe('{node: alpha, cpu: 0}');
  });

  it('formats without keyOrder using original label order', () => {
    expect(labelSetToString(ls({cpu: '0', node: 'alpha'}))).toBe('{cpu: 0, node: alpha}');
  });

  it('returns {} for undefined', () => {
    expect(labelSetToString(undefined)).toBe('{}');
  });
});
