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

import {Input, useURLState} from '@parca/components';
import type {NavigateFunction} from '@parca/utilities';

const FilterByFunctionButton = ({
  navigateTo,
}: {
  navigateTo: NavigateFunction | undefined;
}): JSX.Element => {
  const [storeValue, setStoreValue] = useURLState({param: 'filter_by_function', navigateTo});
  const [localValue, setLocalValue] = useState(storeValue as string);

  const isClearAction = useMemo(() => {
    return localValue === storeValue && localValue != null && localValue !== '';
  }, [localValue, storeValue]);

  const onAction = useCallback((): void => {
    if (isClearAction) {
      setLocalValue('');
      setStoreValue('');
    } else {
      setStoreValue(localValue);
    }
  }, [localValue, isClearAction, setStoreValue]);

  return (
    <Input
      placeholder="Filter by function"
      className="text-sm"
      onAction={onAction}
      onChange={e => setLocalValue(e.target.value)}
      value={localValue ?? ''}
      onBlur={() => setLocalValue(storeValue as string)}
      actionIcon={isClearAction ? <Icon icon="ep:circle-close" /> : <Icon icon="ep:arrow-right" />}
    />
  );
};

export default FilterByFunctionButton;
