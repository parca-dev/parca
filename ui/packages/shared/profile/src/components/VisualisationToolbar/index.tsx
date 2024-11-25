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

import {useCallback} from 'react';

import {Icon} from '@iconify/react';
import cx from 'classnames';

import {QueryRequest, QueryServiceClient} from '@parca/client';
import {Button, UserPreferencesModal, useParcaContext, useURLState} from '@parca/components';
import {ProfileType} from '@parca/parser';

import {FIELD_FUNCTION_NAME, FIELD_LABELS} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {ProfileSource} from '../../ProfileSource';
import GroupByDropdown from '../ActionButtons/GroupByDropdown';
import SortByDropdown from '../ActionButtons/SortByDropdown';
import FilterByFunctionButton from '../FilterByFunctionButton';
import ShareButton from '../ShareButton';
import ViewSelector from '../ViewSelector';
import MultiLevelDropdown from './MultiLevelDropdown';
import TableColumnsDropdown from './TableColumnsDropdown';

interface Props {
  groupBy: string | string[];
  toggleGroupBy: (key: string) => void;
  hasProfileSource: boolean;
  isMultiPanelView: boolean;
  dashboardItems: any;
  profileSource?: ProfileSource;
  queryClient?: QueryServiceClient;
  queryRequest?: QueryRequest;
  onDownloadPProf: () => void;
  pprofdownloading: boolean | undefined;
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
  profileType?: ProfileType;
  total: bigint;
  filtered: bigint;
  setSearchString?: (searchString: string) => void;
  currentSearchString?: string;
  groupByLabels: string[];
}

export const IcicleGraphToolbar = ({
  curPath,
  setNewCurPath,
}: {
  curPath: string[];
  setNewCurPath: (path: string[]) => void;
}): JSX.Element => {
  return (
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
  );
};

export const TableToolbar = ({
  profileType,
  total,
  filtered,
  clearSelection,
  currentSearchString,
}: {
  profileType: ProfileType | undefined;
  total: bigint;
  filtered: bigint;
  clearSelection: () => void;
  currentSearchString: string | undefined;
}): JSX.Element => {
  return (
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
  );
};

const VisualisationToolbar = ({
  hasProfileSource,
  profileSource,
  queryClient,
  onDownloadPProf,
  pprofdownloading,
  curPath,
  setNewCurPath,
  profileType,
  total,
  filtered,
  setSearchString,
  currentSearchString,
  groupByLabels,
}: Props): JSX.Element => {
  const [dashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const {profileViewExternalMainActions, profileViewExternalSubActions, preferencesModal} =
    useParcaContext();

  const [groupBy, setStoreGroupBy] = useURLState<string[]>('group_by', {
    defaultValue: [FIELD_FUNCTION_NAME],
    alwaysReturnArray: true,
  });

  const setGroupBy = useCallback(
    (keys: string[]): void => {
      setStoreGroupBy(keys);
    },
    [setStoreGroupBy]
  );

  const toggleGroupBy = useCallback(
    (key: string): void => {
      groupBy.includes(key)
        ? setGroupBy(groupBy.filter(v => v !== key)) // remove
        : setGroupBy([...groupBy, key]); // add
    },
    [groupBy, setGroupBy]
  );

  const setGroupByLabels = useCallback(
    (labels: string[]): void => {
      setGroupBy(groupBy.filter(l => !l.startsWith(`${FIELD_LABELS}.`)).concat(labels));
    },
    [groupBy, setGroupBy]
  );

  const clearSelection = useCallback((): void => {
    setSearchString?.('');
  }, [setSearchString]);

  const isTableViz = dashboardItems?.includes('table');
  const isGraphViz = dashboardItems?.includes('icicle');

  return (
    <>
      <div
        className={cx(
          'mb-4 flex w-full',
          hasProfileSource || profileViewExternalMainActions != null
            ? 'justify-between'
            : 'justify-end',
          {
            'items-end': !hasProfileSource && profileViewExternalMainActions != null,
            'items-center': hasProfileSource,
          }
        )}
      >
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
            {preferencesModal === true ? <UserPreferencesModal /> : null}
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

export default VisualisationToolbar;
