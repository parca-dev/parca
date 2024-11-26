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

import {FC} from 'react';

import {Icon} from '@iconify/react';

import {QueryServiceClient} from '@parca/client';
import {Button, UserPreferencesModal} from '@parca/components';
import {ProfileType} from '@parca/parser';

import {FIELD_FUNCTION_NAME, FIELD_LABELS} from '../../../ProfileIcicleGraph/IcicleGraphArrow';
import {ProfileSource} from '../../../ProfileSource';
import {useDashboard} from '../../context/DashboardContext';
import GroupByDropdown from '../ActionButtons/GroupByDropdown';
import SortByDropdown from '../ActionButtons/SortByDropdown';
import FilterByFunctionButton from '../FilterByFunctionButton';
import ShareButton from '../ShareButton';
import ViewSelector from '../ViewSelector';
import MultiLevelDropdown from '../VisualisationToolbar/MultiLevelDropdown';
import TableColumnsDropdown from '../VisualisationToolbar/TableColumnsDropdown';

export interface VisualisationToolbarProps {
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
  hasProfileSource: boolean;
  pprofdownloading?: boolean;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  onDownloadPProf: () => void;
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
  profileType?: ProfileType;
  total: bigint;
  filtered: bigint;
  currentSearchString?: string;
  setSearchString?: (value: string) => void;
  groupByLabels: string[];
  preferencesModal?: boolean;
  profileViewExternalSubActions?: React.ReactNode;
  clearSelection: () => void;
  setGroupByLabels: (labels: string[]) => void;
}

export interface TableToolbarProps {
  profileType?: ProfileType;
  total: bigint;
  filtered: bigint;
  clearSelection: () => void;
  currentSearchString?: string;
}

export interface IcicleGraphToolbarProps {
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
}

export const TableToolbar: FC<TableToolbarProps> = ({
  profileType,
  total,
  filtered,
  clearSelection,
  currentSearchString,
}) => {
  return (
    <>
      <div className="flex w-full gap-2 items-end">
        <TableColumnsDropdown profileType={profileType} total={total} filtered={filtered} />
        <Button
          color="neutral"
          onClick={clearSelection}
          className="w-auto"
          variant="neutral"
          disabled={currentSearchString === undefined || currentSearchString.length === 0}
        >
          Clear selection
        </Button>
      </div>
    </>
  );
};

export const IcicleGraphToolbar: FC<IcicleGraphToolbarProps> = ({curPath, setNewCurPath}) => {
  return (
    <>
      <div className="flex w-full gap-2 items-end">
        <SortByDropdown />
        <Button
          variant="neutral"
          className="gap-2 w-max h-fit"
          onClick={() => setNewCurPath([])}
          disabled={curPath.length === 0}
        >
          Reset graph
          <Icon icon="system-uicons:reset" width={20} />
        </Button>
      </div>
    </>
  );
};

export const VisualisationToolbar: FC<VisualisationToolbarProps> = ({
  groupBy,
  toggleGroupBy,
  groupByLabels,
  setGroupByLabels,
  profileType,
  preferencesModal,
  profileSource,
  queryClient,
  onDownloadPProf,
  pprofdownloading,
  profileViewExternalSubActions,
  curPath,
  setNewCurPath,
  total,
  filtered,
  currentSearchString,
  clearSelection,
}) => {
  const {dashboardItems} = useDashboard();

  const isTableViz = dashboardItems?.includes('table');
  const isGraphViz = dashboardItems?.includes('icicle');

  return (
    <>
      <div className="flex w-full justify-between items-end">
        <div className="flex gap-3 items-end">
          <>
            <GroupByDropdown
              groupBy={groupBy}
              toggleGroupBy={toggleGroupBy}
              labels={groupByLabels}
              setGroupByLabels={setGroupByLabels}
            />
            <MultiLevelDropdown profileType={profileType} onSelect={() => {}} />
          </>

          <FilterByFunctionButton />

          {profileViewExternalSubActions != null ? profileViewExternalSubActions : null}
        </div>
        <div className="flex gap-3">
          {preferencesModal != null ? <UserPreferencesModal /> : null}
          <ShareButton
            profileSource={profileSource}
            queryClient={queryClient}
            queryRequest={profileSource?.QueryRequest() ?? undefined}
            onDownloadPProf={onDownloadPProf}
            pprofdownloading={pprofdownloading ?? false}
            profileViewExternalSubActions={profileViewExternalSubActions}
          />
          <ViewSelector />
        </div>
      </div>
      {isGraphViz && !isTableViz && (
        <>
          <div className="border-t border-gray-200 dark:border-gray-700 h-[1px] w-full pb-4"></div>
          <IcicleGraphToolbar curPath={curPath} setNewCurPath={setNewCurPath} />
        </>
      )}
      {isTableViz && !isGraphViz && (
        <>
          <div className="border-t border-gray-200 dark:border-gray-700 h-[1px] w-full pb-4"></div>
          <TableToolbar
            profileType={profileType}
            total={total}
            filtered={filtered}
            clearSelection={clearSelection}
            currentSearchString={currentSearchString}
          />
        </>
      )}
    </>
  );
};
