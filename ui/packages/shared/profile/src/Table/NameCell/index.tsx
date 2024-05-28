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

import {useMemo, useState} from 'react';

import {Table as TableComponent} from '@parca/components';
import {valueFormatter} from '@parca/utilities';

import type {ColumnDef, Row} from '../';
import {useProfileViewContext} from '../../ProfileView/ProfileViewContext';

export const NameCell = ({row: {original}}: {row: {original: Row}}) => {
  const [expanded, setExpanded] = useState(false);
  const {profileSource} = useProfileViewContext();

  const columns: ColumnDef[] = useMemo(
    () => [
      {
        id: 'flat',
        accessorKey: 'flat',
        header: 'Flat',
        cell: info =>
          valueFormatter(info.getValue(), profileSource?.ProfileType()?.sampleUnit ?? '', 2),
        size: 80,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'cumulative',
        accessorKey: 'cumulative',
        header: 'Cumulative',
        cell: info =>
          valueFormatter(info.getValue(), profileSource?.ProfileType()?.sampleUnit ?? '', 2),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      },
      {
        id: 'name',
        accessorKey: 'name',
        header: 'Name',
        cell: info => info.getValue(),
      },
    ],
    [profileSource]
  );

  return (
    <div className="flex flex-col gap-2">
      <div onDoubleClick={() => setExpanded(!expanded)}>{original.name}</div>
      {expanded ? (
        <div className="flex">
          <TableComponent
            columns={columns}
            data={original.callers ?? []}
            className="w-1/2 !pr-0"
            title="caller"
            emptyTableMessage="No callers found."
          />
          <div className="border-r border-gray-200 dark:border-gray-800 h-auto w-[1px]" />
          <TableComponent
            columns={columns}
            data={original.callees ?? []}
            className="w-1/2"
            title="called"
            emptyTableMessage="No callees found."
          />
        </div>
      ) : null}
    </div>
  );
};
