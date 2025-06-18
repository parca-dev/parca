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

import {useEffect, useMemo, useState} from 'react';

import {createColumnHelper, type ColumnDef} from '@tanstack/table-core';

import {useURLState} from '@parca/components';
import {valueFormatter} from '@parca/utilities';

import {type Row} from '..';
import {ColorCell} from '../ColorCell';
import MoreDropdown from '../MoreDropdown';
import {addPlusSign, ratioString, type ColumnName} from '../utils/functions';

interface UseTableConfigurationProps {
  unit?: string;
  total: bigint;
  filtered: bigint;
  compareMode: boolean;
  isSandwich?: boolean;
}

interface TableConfiguration {
  columns: Array<ColumnDef<Row>>;
  initialSorting: Array<{id: string; desc: boolean}>;
  columnVisibility: Record<ColumnName, boolean>;
}

export function useTableConfiguration({
  unit = '',
  total,
  filtered,
  compareMode,
}: UseTableConfigurationProps): TableConfiguration {
  const columnHelper = createColumnHelper<Row>();
  const [tableColumns] = useURLState<string[]>('table_columns', {
    alwaysReturnArray: true,
  });
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });

  const [columnVisibility, setColumnVisibility] = useState(() => {
    return {
      color: true,
      flat: true,
      flatPercentage: false,
      flatDiff: compareMode,
      flatDiffPercentage: false,
      cumulative: true,
      cumulativePercentage: false,
      cumulativeDiff: compareMode,
      cumulativeDiffPercentage: false,
      name: true,
      functionSystemName: false,
      functionFileName: false,
      mappingFile: false,
    };
  });

  useEffect(() => {
    if (Array.isArray(tableColumns)) {
      setColumnVisibility(prevState => {
        const newState = {...prevState};
        (Object.keys(newState) as ColumnName[]).forEach(column => {
          newState[column] = tableColumns.includes(column);
        });
        return newState;
      });
    }
  }, [tableColumns]);

  const columns = useMemo<Array<ColumnDef<Row>>>(() => {
    const baseColumns: Array<ColumnDef<Row>> = [
      columnHelper.accessor('colorProperty', {
        id: 'color',
        header: '',
        cell: info => {
          const color = info.getValue() as {color: string; mappingFile: string};
          return <ColorCell color={color.color} mappingFile={color.mappingFile} />;
        },
        size: 10,
        enableSorting: false,
      }),
      columnHelper.accessor('flat', {
        id: 'flat',
        header: 'Flat',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 80,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flat', {
        id: 'flatPercentage',
        header: 'Flat (%)',
        cell: info => {
          return ratioString(info.getValue(), total, filtered);
        },
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flatDiff', {
        id: 'flatDiff',
        header: 'Flat Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('flatDiff', {
        id: 'flatDiffPercentage',
        header: 'Flat Diff (%)',
        cell: info => {
          return ratioString(info.getValue(), total, filtered);
        },
        size: 120,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        id: 'cumulative',
        header: 'Cumulative',
        cell: info => valueFormatter(info.getValue(), unit, 2),
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulative', {
        id: 'cumulativePercentage',
        header: 'Cumulative (%)',
        cell: info => {
          return ratioString(info.getValue(), total, filtered);
        },
        size: 150,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulativeDiff', {
        id: 'cumulativeDiff',
        header: 'Cumulative Diff',
        cell: info => addPlusSign(valueFormatter(info.getValue(), unit, 2)),
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('cumulativeDiff', {
        id: 'cumulativeDiffPercentage',
        header: 'Cumulative Diff (%)',
        cell: info => {
          return ratioString(info.getValue(), total, filtered);
        },
        size: 170,
        meta: {
          align: 'right',
        },
        invertSorting: true,
      }),
      columnHelper.accessor('name', {
        id: 'name',
        header: 'Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('functionSystemName', {
        id: 'functionSystemName',
        header: 'Function System Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('functionFileName', {
        id: 'functionFileName',
        header: 'Function File Name',
        cell: info => info.getValue(),
      }),
      columnHelper.accessor('mappingFile', {
        id: 'mappingFile',
        header: 'Mapping File',
        cell: info => info.getValue(),
      }),
    ];

    if (dashboardItems.length === 1 && dashboardItems[0] === 'table') {
      baseColumns.unshift(
        columnHelper.accessor('moreActions', {
          id: 'moreActions',
          header: '',
          cell: info => {
            return <MoreDropdown functionName={info.row.original.name} />;
          },
          size: 10,
          enableSorting: false,
        })
      );
    }

    return baseColumns;
  }, [unit, total, filtered, columnHelper, dashboardItems]);

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false,
      },
    ];
  }, [compareMode]);

  return {
    columns,
    initialSorting,
    columnVisibility,
  };
}
