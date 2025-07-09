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

import React, {useEffect, useMemo, useRef, useState} from 'react';

import {type Row as TableRow} from '@tanstack/table-core';
import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';

import {QueryRequest_ReportType, QueryServiceClient} from '@parca/client';
import {useParcaContext, useURLState} from '@parca/components';
import {useCurrentColorProfile} from '@parca/hooks';
import {ProfileType} from '@parca/parser';

import useMappingList, {
  useFilenamesList,
} from '../ProfileFlameGraph/FlameGraphArrow/useMappingList';
import {ProfileSource} from '../ProfileSource';
import {useDashboard} from '../ProfileView/context/DashboardContext';
import {useVisualizationState} from '../ProfileView/hooks/useVisualizationState';
import {FIELD_FUNCTION_NAME, Row} from '../Table';
import {useColorManagement} from '../Table/hooks/useColorManagement';
import {useQuery} from '../useQuery';
import {CalleesSection} from './components/CalleesSection';
import {CallersSection} from './components/CallersSection';
import {processRowData} from './utils/processRowData';

interface Props {
  data?: Uint8Array;
  total: bigint;
  filtered: bigint;
  profileType?: ProfileType;
  loading: boolean;
  unit?: string;
  metadataMappingFiles?: string[];
  queryClient?: QueryServiceClient;
  profileSource: ProfileSource;
}

const Sandwich = React.memo(function Sandwich({
  data,
  filtered,
  profileType,
  loading,
  unit,
  metadataMappingFiles,
  queryClient,
  profileSource,
}: Props): React.JSX.Element {
  const currentColorProfile = useCurrentColorProfile();
  const {dashboardItems} = useDashboard();
  const [sandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');

  const {isDarkMode} = useParcaContext();
  const [selectedRow, setSelectedRow] = useState<TableRow<Row> | null>(null);
  const callersRef = React.useRef<HTMLDivElement | null>(null);
  const calleesRef = React.useRef<HTMLDivElement | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const defaultMaxFrames = 10;

  const callersCalleesContainerRef = useRef<HTMLDivElement | null>(null);

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
            {sandwichFunctionName !== undefined ? (
              <div className="w-full flex flex-col" ref={callersCalleesContainerRef}>
                <CallersSection
                  callersRef={callersRef}
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
                  isExpanded={isExpanded}
                  setIsExpanded={setIsExpanded}
                  defaultMaxFrames={defaultMaxFrames}
                />
                <div className="h-4" />
                <CalleesSection
                  calleesRef={calleesRef}
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
            ) : (
              <div className="items-center justify-center flex h-full w-full">
                <p className="text-sm">
                  {dashboardItems.includes('table')
                    ? 'Please select a function to view its callers and callees.'
                    : 'Use the right-click menu on the flame graph to choose a function to view its callers and callees.'}
                </p>
              </div>
            )}
          </div>
        </motion.div>
      </AnimatePresence>
    </section>
  );
});

export default Sandwich;
