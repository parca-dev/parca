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

import React, {useCallback, useEffect, useMemo, useState} from 'react';

import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';

import {
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {isSearchMatch} from '@parca/utilities';

import useMappingList, {
  useFilenamesList,
} from '../ProfileIcicleGraph/IcicleGraphArrow/useMappingList';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
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
  currentSearchString?: string;
  setSearchString?: (searchString: string) => void;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  isHalfScreen: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
}

export const Table = React.memo(function Table({
  data,
  total,
  filtered,
  profileType,
  loading,
  currentSearchString,
  setSearchString = () => {},
  isHalfScreen,
  unit,
  metadataMappingFiles,
}: TableProps): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });

  const [colorBy, setColorBy] = useURLState('color_by');
  const {isDarkMode} = useParcaContext();
  const [scrollToIndex, setScrollToIndex] = useState<number | undefined>(undefined);

  const {compareMode} = useProfileViewContext();

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
      setSearchString(span.trim());
    },
    [setSearchString]
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

  const shouldHighlightRow = useCallback(
    (row: Row) => {
      if (!('name' in row)) {
        return false;
      }
      const name = row.name;
      return isSearchMatch(currentSearchString as string, name);
    },
    [currentSearchString]
  );

  const enableHighlighting = useMemo(() => {
    return currentSearchString != null && currentSearchString?.length > 0;
  }, [currentSearchString]);

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

  useEffect(() => {
    setTimeout(() => {
      if (currentSearchString == null || rows.length === 0) return;

      const firstHighlightedRowIndex = rows.findIndex(row => {
        return isSearchMatch(currentSearchString, row.name);
      });

      if (firstHighlightedRowIndex !== -1) {
        setScrollToIndex(firstHighlightedRowIndex);
      }
    }, 1000); // Adding a delay to allow the table to render seems to be the only way to get this to work i.e. scrolling down to the highlighted row
  }, [currentSearchString, rows]);

  if (loading) {
    return (
      <div className="overflow-clip h-[700px] min-h-[700px]">
        <TableSkeleton isHalfScreen={isHalfScreen} isDarkMode={isDarkMode} />
      </div>
    );
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
          <div className="font-robotoMono h-[80vh] w-full">
            <TableComponent
              data={rows}
              columns={columns}
              initialSorting={initialSorting}
              columnVisibility={columnVisibility}
              onRowClick={onRowClick}
              enableHighlighting={enableHighlighting}
              shouldHighlightRow={shouldHighlightRow}
              usePointerCursor={dashboardItems.length > 1}
              scrollToIndex={scrollToIndex}
              estimatedRowHeight={ROW_HEIGHT}
            />
          </div>
        </div>
      </motion.div>
    </AnimatePresence>
  );
});

export default Table;
