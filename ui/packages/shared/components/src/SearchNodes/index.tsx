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
import {useAppDispatch, setSearchNodeString} from '@parca/store';

const SearchNodes = () => {
  const dispatch = useAppDispatch();

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    dispatch(setSearchNodeString(event.target.value));
  };

  return (
    <div>
      <Input className="text-sm" placeholder="Search nodes..." onChange={handleChange}></Input>
    </div>
  );
};

export default SearchNodes;
