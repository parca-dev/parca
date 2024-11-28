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

import {createColumnHelper, type CellContext, type ColumnDef} from '@tanstack/table-core';

import {useURLState} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {valueFormatter} from '@parca/utilities';

import {Row, isDummyRow} from '../../../Table';
import ColumnsVisibility from '../../../Table/ColumnsVisibility';
import {ColumnName, DataRow, addPlusSign, getRatioString} from '../../../Table/utils/functions';
import {useProfileViewContext} from '../../context/ProfileViewContext';

interface Props {
  profileType?: ProfileType;
  total: bigint;
  filtered: bigint;
}

const TableColumnsDropdown = ({profileType, total, filtered}: Props): JSX.Element => {
  const {compareMode} = useProfileViewContext();
  const [tableColumns, setTableColumns] = useURLState<string[]>('table_columns', {
    alwaysReturnArray: true,
  });

  const columnHelper = createColumnHelper<Row>();

  const unit: string = useMemo(() => profileType?.sampleUnit ?? '', [profileType?.sampleUnit]);

  const columns = useMemo<Array<ColumnDef<Row>>>(() => {
    return [
      columnHelper.accessor('flat', {
        id: 'flat',
        header: 'Flat',
        cell: info => valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2),
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
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return getRatioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
        cell: info =>
          addPlusSign(valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2)),
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
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return getRatioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
        cell: info => valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2),
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
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return getRatioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
        cell: info =>
          addPlusSign(valueFormatter((info as CellContext<DataRow, bigint>).getValue(), unit, 2)),
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
          if (isDummyRow(info.row.original)) {
            return '';
          }
          return getRatioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileType, unit]);

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

  const updateColumnVisibility = (column: string, isVisible: boolean): void => {
    const updatedColumns = {...columnVisibility, [column]: isVisible};

    const newTableColumns = (Object.keys(updatedColumns) as ColumnName[]).filter(
      col => updatedColumns[col]
    );
    setTableColumns(newTableColumns);
  };

  return (
    <div className="flex flex-col gap-1">
      <label className="text-sm">Table Columns</label>
      <ColumnsVisibility
        columns={columns}
        visibility={columnVisibility}
        setVisibility={(id, visible) => {
          updateColumnVisibility(id, visible);
        }}
      />
    </div>
  );
};

export default TableColumnsDropdown;
