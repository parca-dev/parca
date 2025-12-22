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

import React, {useCallback, useEffect, useMemo, useRef} from 'react';

import {RpcError} from '@protobuf-ts/runtime-rpc';
import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';
import {useContextMenu} from 'react-contexify';

import {
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';

import useMappingList, {
  useFilenamesList,
} from '../ProfileFlameGraph/FlameGraphArrow/useMappingList';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import TableContextMenuWrapper, {TableContextMenuWrapperRef} from './TableContextMenuWrapper';
import {useColorManagement} from './hooks/useColorManagement';
import {useTableConfiguration} from './hooks/useTableConfiguration';
import {DataRow, ROW_HEIGHT, RowName, getRowColor} from './utils/functions';

export const FIELD_MAPPING_FILE = 'mapping_file';
export const FIELD_LOCATION_ADDRESS = 'location_address';
export const FIELD_FUNCTION_NAME = 'function_name';
export const FIELD_FUNCTION_SYSTEM_NAME = 'function_system_name';
export const FIELD_FUNCTION_FILE_NAME = 'function_file_name';
export const FIELD_FLAT = 'flat';
export const FIELD_FLAT_DIFF = 'flat_diff';
export const FIELD_CUMULATIVE = 'cumulative';
export const FIELD_CUMULATIVE_DIFF = 'cumulative_diff';
export const FIELD_CALLERS = 'callers';
export const FIELD_CALLEES = 'callees';

export type Row = DataRow;

export interface TableProps {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  loading: boolean;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  isHalfScreen: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
  error: RpcError | null;
}

export const Table = React.memo(function Table({
  data,
  total,
  filtered,
  profileType,
  loading,
  isHalfScreen,
  unit,
  metadataMappingFiles,
  error,
}: TableProps): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const [_, setSandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const [colorBy, setColorBy] = useURLState('color_by');
  const {isDarkMode} = useParcaContext();
  const {compareMode} = useProfileViewContext();

  const MENU_ID = 'table-context-menu';
  const contextMenuRef = useRef<TableContextMenuWrapperRef>(null);
  const {show} = useContextMenu({
    id: MENU_ID,
  });

  const table = useMemo(() => {
    if (loading || data == null) {
      return null;
    }

    return tableFromIPC(data);
  }, [data, loading]);

  const mappingsList = useMappingList(metadataMappingFiles);
  const filenamesList = useFilenamesList(table);

  const mappingsListCount = useMemo(
    () => mappingsList.filter(m => m !== '').length,
    [mappingsList]
  );

  // If there is only one mapping file, we want to color by filename by default.
  useEffect(() => {
    if (mappingsListCount === 1 && colorBy !== 'filename') {
      setColorBy('filename');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [mappingsListCount]);

  const {colorByColors} = useColorManagement({
    isDarkMode,
    currentColorProfile,
    mappingsList,
    filenamesList,
    colorBy: colorBy as string,
  });

  unit = useMemo(() => unit ?? profileType?.sampleUnit ?? '', [unit, profileType?.sampleUnit]);

  const tableConfig = useTableConfiguration({
    unit,
    total,
    filtered,
    compareMode,
  });

  const {columns, initialSorting, columnVisibility} = tableConfig;

  const selectSpan = useCallback(
    (span: string): void => {
      if (!dashboardItems.includes('flamegraph')) {
        setSandwichFunctionName(span.trim());
      }
    },
    [setSandwichFunctionName, dashboardItems]
  );

  const onRowClick = useCallback(
    (row: Row) => {
      // If there is only one dashboard item, we don't want to select a span
      if (dashboardItems.length <= 1) {
        return;
      }
      selectSpan(row.name);
    },
    [selectSpan, dashboardItems.length]
  );

  const onRowContextMenu = useCallback(
    (e: React.MouseEvent, row: Row) => {
      e.preventDefault();
      contextMenuRef.current?.setRow(row, () => {
        show({
          event: e,
        });
      });
    },
    [show]
  );

  const rows: DataRow[] = useMemo(() => {
    if (table == null || table.numRows === 0) {
      return [];
    }

    const flatColumn = table.getChild(FIELD_FLAT);
    const flatDiffColumn = table.getChild(FIELD_FLAT_DIFF);
    const cumulativeColumn = table.getChild(FIELD_CUMULATIVE);
    const cumulativeDiffColumn = table.getChild(FIELD_CUMULATIVE_DIFF);
    const functionNameColumn = table.getChild(FIELD_FUNCTION_NAME);
    const functionSystemNameColumn = table.getChild(FIELD_FUNCTION_SYSTEM_NAME);
    const functionFileNameColumn = table.getChild(FIELD_FUNCTION_FILE_NAME);
    const mappingFileColumn = table.getChild(FIELD_MAPPING_FILE);
    const locationAddressColumn = table.getChild(FIELD_LOCATION_ADDRESS);

    const getRow = (i: number): DataRow => {
      const flat: bigint = flatColumn?.get(i) ?? 0n;
      const flatDiff: bigint = flatDiffColumn?.get(i) ?? 0n;
      const cumulative: bigint = cumulativeColumn?.get(i) ?? 0n;
      const cumulativeDiff: bigint = cumulativeDiffColumn?.get(i) ?? 0n;
      const functionSystemName: string = functionSystemNameColumn?.get(i) ?? '';
      const functionFileName: string = functionFileNameColumn?.get(i) ?? '';
      const mappingFile: string = mappingFileColumn?.get(i) ?? '';

      return {
        id: i,
        colorProperty: {
          color: getRowColor(
            colorByColors,
            mappingFileColumn,
            i,
            functionFileNameColumn,
            colorBy as string
          ),
          mappingFile,
        },
        name: RowName(mappingFileColumn, locationAddressColumn, functionNameColumn, i),
        flat,
        flatDiff,
        cumulative,
        cumulativeDiff,
        functionSystemName,
        functionFileName,
        mappingFile,
      };
    };

    const rows: DataRow[] = Array.from({length: table.numRows}, (_, i) => getRow(i));

    return rows;
  }, [table, colorByColors, colorBy]);

  if (loading) {
    return (
      <div className="overflow-clip h-[700px] min-h-[700px]">
        <TableSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
      </div>
    );
  }
  if (error != null) {
    return <div className="mx-auto text-center">Error: {error.message}</div>;
  }

  if (rows.length === 0) {
    return <div className="mx-auto text-center">Profile has no samples</div>;
  }

  return (
    <AnimatePresence>
      <motion.div
        className="h-full w-full"
        key="table-loaded"
        initial={{display: 'none', opacity: 0}}
        animate={{display: 'block', opacity: 1}}
        transition={{duration: 0.5}}
      >
        <div className="relative">
          <TableContextMenuWrapper
            ref={contextMenuRef}
            menuId={MENU_ID}
            unit={unit}
            total={total}
            totalUnfiltered={total}
            columnVisibility={columnVisibility}
          />
          <div className="font-robotoMono h-[80vh] w-full">
            <TableComponent
              data={rows}
              columns={columns}
              initialSorting={initialSorting}
              columnVisibility={columnVisibility}
              onRowClick={onRowClick}
              onRowContextMenu={onRowContextMenu}
              usePointerCursor={dashboardItems.length > 1}
              estimatedRowHeight={ROW_HEIGHT}
            />
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
});

export default Table;
