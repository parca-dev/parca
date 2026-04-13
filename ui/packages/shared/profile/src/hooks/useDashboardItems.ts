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

import {useCallback, useMemo} from 'react';

import {parseAsArrayOf, parseAsString, useQueryState} from 'nuqs';

import {useParcaContext} from '@parca/components';

const opts = {history: 'replace' as const};

export const useDashboardItems = (): {
  dashboardItems: string[];
  setDashboardItems: (items: string[]) => void;
} => {
  const {defaultDashboardItems} = useParcaContext();

  const parser = useMemo(
    () =>
      parseAsArrayOf(parseAsString, ',')
        .withDefault(defaultDashboardItems ?? ['flamegraph'])
        .withOptions(opts),
    [defaultDashboardItems]
  );

  const [dashboardItems, setRawDashboardItems] = useQueryState('dashboard_items', parser);

  const setDashboardItems = useCallback(
    (items: string[]) => {
      void setRawDashboardItems(items);
    },
    [setRawDashboardItems]
  );

  return {dashboardItems, setDashboardItems};
};
