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

import {ReactNode, useState} from 'react';

import {Icon} from '@iconify/react';
import Select from 'react-select';

import {IconButton} from '@parca/components';

interface Props {
  sumBy: string[];
  setSumBy: (sumBy: string[]) => void;
  labels: string[];
}

export const Toolbar = ({sumBy, setSumBy, labels}: Props): ReactNode => {
  const [collapsed, setCollapsed] = useState<boolean>(false);
  return (
    <div className="absolute top-1 left-24 rounded-full bg-gray-100 dark:bg-gray-800 z-10 text-xs">
      <div className="flex items-center h-14 gap-4 py-1 px-3">
        <Icon icon="quill:hamburger" height={20} />
        {!collapsed ? (
          <div className="flex gap-2 items-center mr-4">
            <span>Sum by:</span>
            <Select
              defaultValue={[]}
              isMulti
              name="colors"
              options={labels.map(label => ({label, value: label}))}
              className="basic-multi-select min-w-60"
              classNamePrefix="select"
              value={sumBy.map(sumBy => ({label: sumBy, value: sumBy}))}
              onChange={selectedOptions => {
                setSumBy(selectedOptions.map(option => option.value));
              }}
              placeholder="Labels..."
              styles={{
                indicatorSeparator: () => ({display: 'none'}),
              }}
            />
          </div>
        ) : null}
        <IconButton
          icon={
            <Icon
              icon={collapsed ? 'iconamoon:arrow-right-2-light' : 'iconamoon:arrow-left-2-light'}
              height={24}
            />
          }
          onClick={() => setCollapsed(!collapsed)}
          className="hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full p-1"
        />
      </div>
    </div>
  );
};
