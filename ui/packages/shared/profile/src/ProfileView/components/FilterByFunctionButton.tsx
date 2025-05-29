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

import {useCallback, useMemo, useState} from 'react';

import {Icon} from '@iconify/react';

import {Input, Select, useURLState, type SelectItem} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

const FilterByFunctionButton = (): JSX.Element => {
  const [highlightAfterFilteringEnabled] = useUserPreference<boolean>(
    USER_PREFERENCES.HIGHTLIGHT_AFTER_FILTERING.key
  );
  const [storeValue, setStoreValue] = useURLState('filter_by_function');
  const [excludeFunctionStoreValue, setExcludeFunctionStoreValue] = useURLState('exclude_function');
  const [excludeFunction, setExcludeFunction] = useState(excludeFunctionStoreValue === 'true');
  const [_, setSearchString] = useURLState('search_string');
  const [localValue, setLocalValue] = useState(storeValue as string);

  const isClearAction = useMemo(() => {
    return (
      localValue === storeValue &&
      localValue != null &&
      localValue !== '' &&
      (excludeFunction
        ? excludeFunctionStoreValue === 'true'
        : excludeFunctionStoreValue !== 'true')
    );
  }, [localValue, storeValue, excludeFunction, excludeFunctionStoreValue]);

  const onAction = useCallback((): void => {
    if (isClearAction) {
      setLocalValue('');
      setStoreValue('');
      setExcludeFunction(false);
      setExcludeFunctionStoreValue('');
      if (highlightAfterFilteringEnabled) {
        setSearchString('');
      }
    } else {
      setStoreValue(localValue);
      setExcludeFunctionStoreValue(excludeFunction ? 'true' : '');
      if (!excludeFunction && highlightAfterFilteringEnabled) {
        setSearchString(localValue);
      } else {
        setSearchString('');
      }
    }
  }, [
    localValue,
    isClearAction,
    setStoreValue,
    setExcludeFunctionStoreValue,
    excludeFunction,
    highlightAfterFilteringEnabled,
    setSearchString,
  ]);

  const filterModeItems: SelectItem[] = [
    {
      key: 'include',
      element: {
        active: <>Contains</>,
        expanded: <>Contains</>,
      },
    },
    {
      key: 'exclude',
      element: {
        active: <>Not Contains</>,
        expanded: <>Not Contains</>,
      },
    },
  ];

  const handleFilterModeChange = useCallback((key: string): void => {
    setExcludeFunction(key === 'exclude');
  }, []);

  return (
    <div className="relative">
      <div className="absolute inset-y-0 left-0 z-10 flex items-center">
        <div className="h-full">
          <Select
            items={filterModeItems}
            selectedKey={excludeFunction ? 'exclude' : 'include'}
            onSelection={handleFilterModeChange}
            className="h-full rounded-l-md rounded-r-none w-[148px]"
          />
        </div>
      </div>
      <div className="flex">
        <Input
          placeholder="Filter by function"
          className="pl-[152px] text-sm"
          onAction={onAction}
          onChange={e => setLocalValue(e.target.value)}
          value={localValue ?? ''}
          onBlur={() => setLocalValue(storeValue as string)}
          actionIcon={
            isClearAction ? <Icon icon="ep:circle-close" /> : <Icon icon="ep:arrow-right" />
          }
          id="h-filter-by-function"
        />
      </div>
    </div>
  );
};

export default FilterByFunctionButton;
