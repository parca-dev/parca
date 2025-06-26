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

import React, {useCallback, useEffect, useMemo, useRef, useState} from 'react';

import {type Row as TableRow} from '@tanstack/table-core';
import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';

import {QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {TableSkeleton, useParcaContext, useURLState} from '@parca/components';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';
import {isSearchMatch} from '@parca/utilities';

import useMappingList, {
  useFilenamesList,
} from '../ProfileFlameGraph/FlameGraphArrow/useMappingList';
import {ProfileSource} from '../ProfileSource';
import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {useVisualizationState} from '../ProfileView/hooks/useVisualizationState';
import {FIELD_FUNCTION_NAME, Row} from '../Table';
import {useColorManagement} from '../Table/hooks/useColorManagement';
import {useTableConfiguration} from '../Table/hooks/useTableConfiguration';
import {type DataRow} from '../Table/utils/functions';
import {useQuery} from '../useQuery';
import {CalleesSection} from './components/CalleesSection';
import {CallersSection} from './components/CallersSection';
import {TableSection} from './components/TableSection';
import {processRowData} from './utils/processRowData';

interface Props {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  loading: boolean;
  isHalfScreen: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
  queryClient?: QueryServiceClient;
  profileSource: ProfileSource;
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
}: Props): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();

  const [sandwichFunctionName, setSandwichFunctionName] = useURLState<string | undefined>(
    'sandwich_function_name'
  );
  const {isDarkMode} = useParcaContext();
  const [selectedRow, setSelectedRow] = useState<TableRow<Row> | null>(null);
  const callersRef = React.useRef<HTMLDivElement | null>(null);
  const calleesRef = React.useRef<HTMLDivElement | null>(null);

  const callersCalleesContainerRef = useRef<HTMLDivElement | null>(null);
  const [tableHeight, setTableHeight] = useState<number | undefined>(undefined);

  const {compareMode} = useProfileViewContext();

  const {colorBy, setColorBy, curPathArrow, setCurPathArrow} = useVisualizationState();

  const nodeTrimThreshold = useMemo(() => {
    let width =
      // eslint-disable-next-line @typescript-eslint/strict-boolean-expressions
      window.innerWidth || document.documentElement.clientWidth || document.body.clientWidth;
    // subtract the padding
    width = width - 12 - 16 - 12;
    return (1 / width) * 100;
  }, []);

  const {
    isLoading: callersFlamegraphLoading,
    response: callersFlamegraphResponse,
    error: callersFlamegraphError,
  } = useQuery(
    queryClient as QueryServiceClient,
    profileSource,
    QueryRequest_ReportType.FLAMEGRAPH_ARROW,
    {
      nodeTrimThreshold,
      groupBy: [FIELD_FUNCTION_NAME],
      invertCallStack: true,
      binaryFrameFilter: [],
      sandwichByFunction: sandwichFunctionName,
      skip: sandwichFunctionName === undefined,
    }
  );

  const {
    isLoading: calleesFlamegraphLoading,
    response: calleesFlamegraphResponse,
    error: calleesFlamegraphError,
  } = useQuery(
    queryClient as QueryServiceClient,
    profileSource,
    QueryRequest_ReportType.FLAMEGRAPH_ARROW,
    {
      nodeTrimThreshold,
      groupBy: [FIELD_FUNCTION_NAME],
      invertCallStack: false,
      binaryFrameFilter: [],
      sandwichByFunction: sandwichFunctionName,
      skip: sandwichFunctionName === undefined,
    }
  );

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

  const {colorByColors, colorByValue} = useColorManagement({
    isDarkMode,
    currentColorProfile,
    mappingsList,
    filenamesList,
    colorBy,
  });

  unit = useMemo(() => unit ?? profileType?.sampleUnit ?? '', [unit, profileType?.sampleUnit]);

  const tableConfig = useTableConfiguration({
    unit,
    total,
    filtered,
    compareMode,
  });

  const {columns, initialSorting, columnVisibility} = tableConfig;

  const rows = useMemo(() => {
    if (table == null || table.numRows === 0) {
      return [];
    }

    return processRowData({
      table,
      colorByColors,
      colorBy: colorByValue,
    });
  }, [table, colorByColors, colorByValue]);

  useEffect(() => {
    if (sandwichFunctionName !== undefined && selectedRow == null) {
      // find the row with the sandwichFunctionName
      const row = rows.find(row => {
        return row.name.trim() === sandwichFunctionName.trim();
      });

      if (row != null) {
        setSelectedRow(row as unknown as TableRow<Row>);
      }
    }
  }, [sandwichFunctionName, rows, selectedRow]);

  // Update table height based on callers/callees container height
  useEffect(() => {
    const updateTableHeight = (): void => {
      if (callersCalleesContainerRef.current != null) {
        const containerHeight = callersCalleesContainerRef.current.getBoundingClientRect().height;
        setTableHeight(containerHeight);
      }
    };

    // Initial measurement
    updateTableHeight();

    // Update on window resize
    window.addEventListener('resize', updateTableHeight);

    // Use ResizeObserver if available for more accurate updates
    let resizeObserver: ResizeObserver | null = null;
    if (callersCalleesContainerRef.current != null && 'ResizeObserver' in window) {
      resizeObserver = new ResizeObserver(updateTableHeight);
      resizeObserver.observe(callersCalleesContainerRef.current);
    }

    return () => {
      window.removeEventListener('resize', updateTableHeight);
      if (resizeObserver != null) {
        resizeObserver.disconnect();
      }
    };
  }, [sandwichFunctionName, callersFlamegraphResponse, calleesFlamegraphResponse]);

  const onRowClick = useCallback(
    (row: DataRow) => {
      setSelectedRow(row as unknown as TableRow<Row>);
      setSandwichFunctionName(row.name.trim());
    },
    [setSandwichFunctionName]
  );

  const enableHighlighting = useMemo(() => {
    return sandwichFunctionName != null && sandwichFunctionName?.length > 0;
  }, [sandwichFunctionName]);

  const shouldHighlightRow = useCallback(
    (row: Row) => {
      if (!('name' in row)) {
        return false;
      }
      const name = row.name;
      return isSearchMatch(sandwichFunctionName as string, name);
    },
    [sandwichFunctionName]
  );

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
            <TableSection
              rows={rows}
              columns={columns}
              initialSorting={initialSorting}
              columnVisibility={columnVisibility}
              selectedRow={selectedRow}
              onRowClick={onRowClick}
              shouldHighlightRow={shouldHighlightRow}
              enableHighlighting={enableHighlighting}
              height={tableHeight}
              sandwichFunctionName={sandwichFunctionName}
            />

            {sandwichFunctionName !== undefined && (
              <div className="w-[50%] flex flex-col" ref={callersCalleesContainerRef}>
                <CallersSection
                  callersRef={callersRef}
                  isHalfScreen={isHalfScreen}
                  callersFlamegraphResponse={
                    callersFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
                      ? {
                          report: {
                            oneofKind: 'flamegraphArrow',
                            flamegraphArrow: callersFlamegraphResponse.report.flamegraphArrow,
                          },
                          total: callersFlamegraphResponse.total?.toString() ?? '0',
                        }
                      : undefined
                  }
                  callersFlamegraphLoading={callersFlamegraphLoading}
                  callersFlamegraphError={callersFlamegraphError}
                  filtered={filtered}
                  profileSource={profileSource}
                  curPathArrow={curPathArrow}
                  setCurPathArrow={setCurPathArrow}
                  metadataMappingFiles={metadataMappingFiles}
                />
                <div className="h-4" />
                <CalleesSection
                  calleesRef={calleesRef}
                  isHalfScreen={isHalfScreen}
                  calleesFlamegraphResponse={
                    calleesFlamegraphResponse?.report.oneofKind === 'flamegraphArrow'
                      ? {
                          report: {
                            oneofKind: 'flamegraphArrow',
                            flamegraphArrow: calleesFlamegraphResponse.report.flamegraphArrow,
                          },
                          total: calleesFlamegraphResponse.total?.toString() ?? '0',
                        }
                      : undefined
                  }
                  calleesFlamegraphLoading={calleesFlamegraphLoading}
                  calleesFlamegraphError={calleesFlamegraphError}
                  filtered={filtered}
                  profileSource={profileSource}
                  curPathArrow={curPathArrow}
                  setCurPathArrow={setCurPathArrow}
                  metadataMappingFiles={metadataMappingFiles}
                />
              </div>
            )}
          </div>
        </motion.div>
      </AnimatePresence>
    </section>
  );
});

export default Sandwich;
