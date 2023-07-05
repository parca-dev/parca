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

import {useEffect, useMemo} from 'react';

import {debounce} from 'lodash';

import {useUIFeatureFlag} from '@parca/hooks';
import type {NavigateFunction} from '@parca/utilities';

import {Input, useURLState} from '../';

interface Props {
  navigateTo?: NavigateFunction;
}

const SearchNodes = ({navigateTo}: Props): JSX.Element => {
  const [currentSearchString, setSearchString] = useURLState({param: 'search_string', navigateTo});
  const [filterByFunctionEnabled] = useUIFeatureFlag('filterByFunction');

  useEffect(() => {
    return () => {
      debouncedSearch.cancel();
    };
  });

  const debouncedSearch = useMemo(() => {
    const handleChange = (event: React.ChangeEvent<HTMLInputElement>): void => {
      const searchString = event.target.value;
      setSearchString(searchString);
    };

    return debounce(handleChange, 300);
  }, [setSearchString]);

  return (
    <div>
      <Input
        className="text-sm"
        placeholder={filterByFunctionEnabled ? 'Highlight nodes...' : 'Search nodes...'}
        onChange={debouncedSearch}
        defaultValue={currentSearchString as string}
      />
    </div>
  );
};

export default SearchNodes;
