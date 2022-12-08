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

import {Input} from '@parca/components';
import {
  selectFilterByFunction,
  setFilterByFunction,
  useAppDispatch,
  useAppSelector,
} from '@parca/store';
import {Icon} from '@iconify/react';
import {useCallback, useMemo, useState} from 'react';

const FilterByFunctionButton = (): JSX.Element => {
  const dispatch = useAppDispatch();
  const storeVal = useAppSelector(selectFilterByFunction);
  const [value, setValue] = useState<string>(storeVal ?? '');

  const isClearAction = useMemo(() => {
    return value === storeVal && value != null;
  }, [value, storeVal]);

  const onAction = useCallback((): void => {
    dispatch(setFilterByFunction(isClearAction ? undefined : value));
    if (isClearAction) {
      setValue('');
    }
  }, [dispatch, isClearAction, value]);

  return (
    <Input
      placeholder="Filter by function"
      className="text-sm"
      onAction={onAction}
      onChange={e => setValue(e.target.value)}
      value={value}
      onBlur={() => setValue(storeVal ?? '')}
      actionIcon={isClearAction ? <Icon icon="ep:circle-close" /> : <Icon icon="ep:arrow-right" />}
    />
  );
};

export default FilterByFunctionButton;
