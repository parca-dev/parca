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

import {Input} from '../';
import {useEffect, useMemo} from 'react';
import {useAppDispatch, setSearchNodeString} from '@parca/store';
import useUIFeatureFlag from '@parca/functions/useUIFeatureFlag';
import {debounce} from 'lodash';

const SearchNodes = (): JSX.Element => {
  const dispatch = useAppDispatch();
  const [filterByFunctionEnabled] = useUIFeatureFlag('filterByFunction');

  useEffect(() => {
    return () => {
      debouncedSearch.cancel();
    };
  });

  const debouncedSearch = useMemo(() => {
    const handleChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
      dispatch(setSearchNodeString(event.target.value));
    };

    return debounce(handleChange, 300);
  }, [dispatch]);

  return (
    <div>
      <Input
        className="text-sm"
        placeholder={filterByFunctionEnabled ? 'Highlight nodes...' : 'Search nodes...'}
        onChange={debouncedSearch}
      ></Input>
    </div>
  );
};

export default SearchNodes;
