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

import {
  createColumnHelper,
  type CellContext,
  type ColumnDef,
  type ExpandedState,
  type Row as RowType,
} from '@tanstack/table-core';
import {Int64, Vector, tableFromIPC, vectorFromArray} from 'apache-arrow';
import cx from 'classnames';
import {AnimatePresence, motion} from 'framer-motion';
import {Tooltip} from 'react-tooltip';

import {QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {
  Table as TableComponent,
  TableSkeleton,
  useParcaContext,
  useURLState,
} from '@parca/components';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {getLastItem, isSearchMatch, valueFormatter} from '@parca/utilities';

import ProfileIcicleGraph from '../ProfileIcicleGraph';
import {getFilenameColors, getMappingColors} from '../ProfileIcicleGraph/IcicleGraphArrow/';
import {colorByColors} from '../ProfileIcicleGraph/IcicleGraphArrow/IcicleGraphNodes';
import useMappingList, {
  useFilenamesList,
} from '../ProfileIcicleGraph/IcicleGraphArrow/useMappingList';
import {ProfileSource} from '../ProfileSource';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {useVisualizationState} from '../ProfileView/hooks/useVisualizationState';
import {
  FIELD_CALLEES,
  FIELD_CALLERS,
  FIELD_CUMULATIVE,
  FIELD_CUMULATIVE_DIFF,
  FIELD_FLAT,
  FIELD_FLAT_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_FUNCTION_SYSTEM_NAME,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
  Row,
  isDummyRow,
} from '../Table';
import {
  ColumnName,
  DataRow,
  ROW_HEIGHT,
  RowName,
  addPlusSign,
  getCalleeRows,
  getCallerRows,
  getRowColor,
  ratioString,
  sizeToBottomStyle,
  sizeToWidthStyle,
} from '../Table/utils/functions';
import {getTopAndBottomExpandedRowModel} from '../Table/utils/topAndBottomExpandedRowModel';
import {useQuery} from '../useQuery';
import CustomRowRenderer from './CustomRenderer';

interface Props {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  loading: boolean;
  currentSearchString?: string;
  setActionButtons?: (buttons: React.JSX.Element) => void;
  isHalfScreen: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
  metadataLoading?: boolean;
  callees?: Uint8Array;
  callers?: Uint8Array;
  queryClient?: QueryServiceClient;
  profileSource?: ProfileSource;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
}

const Sandwich = React.memo(function Sandwich({
  data,
  total,
  filtered,
  profileType,
  loading,
  isHalfScreen,
  unit,
  metadataMappingFiles,
  queryClient,
  profileSource,
  metadataLoading,
}: Props): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });

  const [tableColumns] = useURLState<string[]>('table_columns', {
    alwaysReturnArray: true,
  });
  const {isDarkMode} = useParcaContext();
  const [expanded, setExpanded] = useState<ExpandedState>({});
  const [selectedRow, setSelectedRow] = useState<RowType<Row> | null>(null);
  const callersRef = React.useRef<HTMLDivElement | null>(null);
  const calleesRef = React.useRef<HTMLDivElement | null>(null);

  const {compareMode} = useProfileViewContext();

  const nodeTrimThreshold = useMemo(() => {
    let width =
      // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
      window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    // subtract the padding
    width = width - 12 - 16 - 12;
    return (1 / width) * 100;
  }, []);

  const [selectedFunctionName, setSelectedFunctionName] = useState<string | undefined>();

  const {
    isLoading: callersFlamegraphLoading,
    response: callersFlamegraphResponse,
    error: callersFlamegraphError,
  } = useQuery(
    queryClient as QueryServiceClient,
    profileSource as ProfileSource,
    QueryRequest_ReportType.FLAMEGRAPH_ARROW,
    {
      nodeTrimThreshold,
      groupBy: [FIELD_FUNCTION_NAME],
      invertCallStack: true,
      binaryFrameFilter: [],
      filterByFunction: selectedFunctionName,
      skip: selectedFunctionName === undefined,
    }
  );

  const {
    isLoading: calleesFlamegraphLoading,
    response: calleesFlamegraphResponse,
    error: calleesFlamegraphError,
  } = useQuery(
    queryClient as QueryServiceClient,
    profileSource as ProfileSource,
    QueryRequest_ReportType.FLAMEGRAPH_ARROW,
    {
      nodeTrimThreshold,
      groupBy: [FIELD_FUNCTION_NAME],
      invertCallStack: false,
      binaryFrameFilter: [],
      filterByFunction: selectedFunctionName,
      skip: selectedFunctionName === undefined,
    }
  );

  const {curPath, setCurPath, colorBy, setColorBy, curPathArrow, setCurPathArrow} =
    useVisualizationState();

  const table = useMemo(() => {
    if (loading || data == null) {
      return null;
    }

    return tableFromIPC(data);
  }, [data, loading]);

  const mappingsList = useMappingList(metadataMappingFiles);
  const filenamesList = useFilenamesList(table);
  const colorByValue = colorBy === undefined || colorBy === '' ? 'binary' : colorBy;

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

  const filenameColors = useMemo(() => {
    const colors = getFilenameColors(filenamesList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, filenamesList, currentColorProfile]);

  const mappingColors = useMemo(() => {
    const colors = getMappingColors(mappingsList, isDarkMode, currentColorProfile);
    return colors;
  }, [isDarkMode, mappingsList, currentColorProfile]);

  const colorByList = {
    filename: filenameColors,
    binary: mappingColors,
  };

  type ColorByKey = keyof typeof colorByList;

  const colorByColors: colorByColors = colorByList[colorByValue as ColorByKey];

  const columnHelper = createColumnHelper<Row>();

  unit = useMemo(() => unit ?? profileType?.sampleUnit ?? '', [unit, profileType?.sampleUnit]);

  const columns = useMemo<Array<ColumnDef<Row>>>(() => {
    return [
      columnHelper.accessor('colorProperty', {
        id: 'color',
        header: '',
        cell: info => {
          const color = info.getValue() as {color: string; mappingFile: string};
          return (
            <>
              <div
                className="w-4 h-4 rounded-[4px]"
                style={{backgroundColor: color.color}}
                data-tooltip-id="table-color-tooltip"
                data-tooltip-content={getLastItem(color.mappingFile)}
              />
              <Tooltip id="table-color-tooltip" />
            </>
          );
        },
        size: 10,
      }),
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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
          return ratioString((info as CellContext<DataRow, bigint>).getValue(), total, filtered);
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

  const onRowClick = useCallback((row: RowType<Row>) => {
    if (isDummyRow(row.original)) {
      return;
    }

    setSelectedRow(row);
    setSelectedFunctionName(row.original.name.trim());
  }, []);

  const initialSorting = useMemo(() => {
    return [
      {
        id: compareMode ? 'flatDiff' : 'flat',
        desc: false, // columns sorting are inverted - so this is actually descending
      },
    ];
  }, [compareMode]);

  const enableHighlighting = useMemo(() => {
    return selectedRow != null;
  }, [selectedRow]);

  const shouldHighlightRow = useCallback(
    (row: Row) => {
      if (!('name' in row)) {
        return false;
      }
      const name = row.name;
      // @ts-expect-error
      return isSearchMatch(selectedRow?.original?.name as string, name);
    },
    [selectedRow]
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
    const callersColumn = table.getChild(FIELD_CALLERS);
    const calleesColumn = table.getChild(FIELD_CALLEES);

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
          color: getRowColor(colorByColors, mappingFileColumn, i, functionFileNameColumn, colorBy),
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

    const rows: DataRow[] = [];
    for (let i = 0; i < table.numRows; i++) {
      const row = getRow(i);
      const callerIndices: Vector<Int64> = callersColumn?.get(i) ?? vectorFromArray([]);
      const callers: DataRow[] = Array.from(callerIndices.toArray().values()).map(rowIdx => {
        return getRow(Number(rowIdx));
      });

      const calleeIndices: Vector<Int64> = calleesColumn?.get(i) ?? vectorFromArray([]);
      const callees: DataRow[] = Array.from(calleeIndices.toArray().values()).map(rowIdx => {
        return getRow(Number(rowIdx));
      });

      row.callers = callers;
      row.callees = callees;
      row.subRows = [...getCallerRows(callers), ...getCalleeRows(callees)];

      rows.push(row);
    }

    return rows;
  }, [table, colorByColors, colorBy]);

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

  // console.log(dimensions, ref);

  return (
    <section className="flex flex-row h-full w-full">
      <AnimatePresence>
        <motion.div
          className="h-full w-full"
          key="sandwich-loaded"
          initial={{display: 'none', opacity: 0}}
          animate={{display: 'block', opacity: 1}}
          transition={{duration: 0.5}}
        >
          <div className="relative flex flex-row">
            <div
              className={cx('font-robotoMono h-[80vh] w-full cursor-pointer', {
                'w-[50%]': selectedRow != null,
              })}
            >
              <TableComponent
                data={rows}
                columns={columns}
                initialSorting={initialSorting}
                columnVisibility={columnVisibility}
                usePointerCursor={dashboardItems.length > 1}
                onRowDoubleClick={onRowClick}
                getSubRows={row => (isDummyRow(row) ? [] : row.subRows ?? [])}
                getCustomExpandedRowModel={getTopAndBottomExpandedRowModel}
                expandedState={expanded}
                shouldHighlightRow={shouldHighlightRow}
                enableHighlighting={enableHighlighting}
                onExpandedChange={getNewState => {
                  // We only want the new expanded row so passing the exisitng state as empty
                  // @ts-expect-error
                  let newState = getNewState({});
                  if (Object.keys(newState)[0] === Object.keys(expanded)[0]) {
                    newState = {};
                  }
                  setExpanded(newState);
                }}
                CustomRowRenderer={CustomRowRenderer}
                estimatedRowHeight={ROW_HEIGHT}
                sandwich={true}
              />
            </div>

            {selectedRow != null && (
              <div className="w-[50%] flex flex-col">
                <div className="flex relative flex-row" ref={callersRef}>
                  <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
                    Callers {'->'}
                  </div>
                  <ProfileIcicleGraph
                    curPath={curPath}
                    setNewCurPath={setCurPath}
                    arrow={
                      callersFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
                        ? callersFlamegraphResponse?.report?.flamegraphArrow
                        : undefined
                    }
                    graph={undefined}
                    total={BigInt(callersFlamegraphResponse?.total ?? '0')}
                    filtered={filtered}
                    profileType={profileSource?.ProfileType()}
                    loading={callersFlamegraphLoading}
                    error={callersFlamegraphError}
                    isHalfScreen={true}
                    width={
                      callersRef.current != null
                        ? isHalfScreen
                          ? (callersRef.current.getBoundingClientRect().width - 54) / 2
                          : callersRef.current.getBoundingClientRect().width - 16
                        : 0
                    }
                    metadataMappingFiles={metadataMappingFiles}
                    metadataLoading={metadataLoading}
                    isSandwichIcicleGraph={true}
                    curPathArrow={curPathArrow}
                    setNewCurPathArrow={setCurPathArrow}
                  />
                </div>
                {/* divider space */}
                <div className="h-4" />
                {/* divider space */}
                <div className="flex relative items-start flex-row" ref={calleesRef}>
                  <div className="[writing-mode:vertical-lr] -rotate-180 px-1 uppercase text-[10px] text-left">
                    {'<-'} Callees
                  </div>
                  <ProfileIcicleGraph
                    curPath={curPath}
                    setNewCurPath={setCurPath}
                    arrow={
                      calleesFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
                        ? calleesFlamegraphResponse?.report?.flamegraphArrow
                        : undefined
                    }
                    graph={undefined}
                    total={BigInt(calleesFlamegraphResponse?.total ?? '0')}
                    filtered={filtered}
                    profileType={profileSource?.ProfileType()}
                    loading={calleesFlamegraphLoading}
                    error={calleesFlamegraphError}
                    isHalfScreen={true}
                    width={
                      calleesRef.current != null
                        ? isHalfScreen
                          ? (calleesRef.current.getBoundingClientRect().width - 54) / 2
                          : calleesRef.current.getBoundingClientRect().width - 16
                        : 0
                    }
                    metadataMappingFiles={metadataMappingFiles}
                    metadataLoading={metadataLoading}
                    isSandwichIcicleGraph={true}
                    curPathArrow={curPathArrow}
                    setNewCurPathArrow={setCurPathArrow}
                  />
                </div>
              </div>
            )}
          </div>
        </motion.div>
      </AnimatePresence>
    </section>
  );
});

export default Sandwich;
