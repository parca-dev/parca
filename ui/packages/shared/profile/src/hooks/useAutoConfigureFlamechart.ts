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

import {useCallback} from 'react';

import {useURLState, useURLStateBatch} from '@parca/components';

export const OPTIMAL_LABELS = ['node', 'cpu', 'thread_id'];

export const useAutoConfigureFlamechart = (): (() => void) => {
  const batchUpdates = useURLStateBatch();
  const [existingSumByA] = useURLState<string>('sum_by_a');
  const [_, setSumByA] = useURLState('sum_by_a');

  const [, setSelectionA] = useURLState('selection_a');
  const [expressionA] = useURLState('expression_a');
  const [, setTimeSelectionA] = useURLState('time_selection_a');
  const [, setFromA] = useURLState('from_a');
  const [, setToA] = useURLState('to_a');
  const [, setMergeFromA] = useURLState('merge_from_a');
  const [, setMergeToA] = useURLState('merge_to_a');
  const [, setAutoConfigTsA] = useURLState('autoconfig_ts_a');

  const handleAutoConfigureFlameChart = useCallback(() => {
    const toMs = Date.now();
    const fromMs = toMs - 60000;

    const existing =
      existingSumByA !== undefined && existingSumByA !== ''
        ? existingSumByA.split(',').filter(Boolean)
        : [];
    const toAdd = OPTIMAL_LABELS.filter(label => !existing.includes(label));
    const mergedLabels = [...existing, ...toAdd];

    batchUpdates(() => {
      setFromA(fromMs.toString());
      setToA(toMs.toString());
      setTimeSelectionA('relative:minute|1');
      setMergeFromA((BigInt(fromMs) * 1_000_000n).toString());
      setMergeToA((BigInt(toMs) * 1_000_000n).toString());
      setSumByA(mergedLabels.join(','));

      // Signal to QueryControls that auto-config was triggered
      setAutoConfigTsA(Date.now().toString());

      // Update selection to trigger ProfileSelector refresh
      if (expressionA !== undefined && expressionA !== '') {
        setSelectionA(expressionA);
      }
    });
  }, [
    batchUpdates,
    existingSumByA,
    setFromA,
    setToA,
    setTimeSelectionA,
    setMergeFromA,
    setMergeToA,
    setSumByA,
    setAutoConfigTsA,
    expressionA,
    setSelectionA,
  ]);

  return handleAutoConfigureFlameChart;
};
