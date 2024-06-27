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

import {ReactNode, useId, useMemo, useState} from 'react';

import {Icon} from '@iconify/react';
import Draggable from 'react-draggable';
import Select from 'react-select';

import {IconButton} from '@parca/components';

interface Props {
  sumBy: string[];
  setSumBy: (sumBy: string[]) => void;
  labels: string[];
}

export const Toolbar = ({sumBy, setSumBy, labels}: Props): ReactNode => {
  const [collapsed, setCollapsed] = useState<boolean>(false);
  const [isDragging, setIsDragging] = useState<boolean>(false);
  const idWithColon = useId();
  const id = useMemo(() => idWithColon.replace(/[^a-zA-Z0-9]/g, ''), [idWithColon]);

  return (
    <Draggable
      handle={`#${id}`}
      onStart={() => setIsDragging(true)}
      onStop={() => setIsDragging(false)}
      bounds="parent"
      defaultPosition={{x: 96, y: 4}}
    >
      <div className="absolute rounded-full bg-gray-100 dark:bg-gray-800 z-10 text-xs">
        <div className="flex items-center h-14 gap-4 py-1 px-3">
          <Icon
            icon="radix-icons:drag-handle-dots-2"
            height={20}
            className={isDragging ? 'cursor-grabbing' : 'cursor-grab'}
            id={id}
          />
          {!collapsed ? (
            <div className="flex gap-2 items-center mr-4">
              <span>Sum by</span>
              <Select
                defaultValue={[]}
                isMulti
                name="colors"
                options={labels.map(label => ({label, value: label}))}
                className="parca-select-container min-w-60"
                classNamePrefix="parca-select"
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
          <div className="relative">
            <IconButton
              icon={
                <Icon
                  icon={
                    collapsed ? 'iconamoon:arrow-right-2-light' : 'iconamoon:arrow-left-2-light'
                  }
                  height={24}
                />
              }
              onClick={() => setCollapsed(!collapsed)}
              className="hover:bg-gray-200 dark:hover:bg-gray-700 rounded-full p-1"
            />
            {collapsed && sumBy.length > 0 ? (
              <div className="rounded-full bg-indigo-600 dark:bg-indigo-500 absolute text-xs h-4 w-4 flex items-center justify-center -top-3 -right-3 text-white text-[10px]">
                {sumBy.length}
              </div>
            ) : null}
          </div>
        </div>
      </div>
    </Draggable>
  );
};
