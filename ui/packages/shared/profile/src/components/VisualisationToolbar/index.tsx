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

import {FIELD_FUNCTION_NAME} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {ProfileSource} from '../../ProfileSource';
import GroupByDropdown from '../ActionButtons/GroupByDropdown';
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
}

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

  const clearSelection = useCallback((): void => {
    setSearchString?.('');
  }, [setSearchString]);

  const isTableViz = dashboardItems?.includes('table');
  const isGraphViz = dashboardItems?.includes('icicle');

  return (
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
          {isGraphViz && (
            <>
              <GroupByDropdown groupBy={groupBy} toggleGroupBy={toggleGroupBy} />
              <MultiLevelDropdown profileType={profileType} onSelect={() => {}} />
              <Button
                variant="neutral"
                className="gap-2 w-max"
                onClick={() => setNewCurPath([])}
                disabled={curPath.length === 0}
              >
                Reset graph
                <Icon icon="system-uicons:reset" width={20} />
              </Button>
            </>
          )}

          {isTableViz && (
            <>
              <TableColumnsDropdown profileType={profileType} total={total} filtered={filtered} />
              {dashboardItems.length > 1 && (
                <Button
                  color="neutral"
                  onClick={clearSelection}
                  className="w-auto"
                  variant="neutral"
                  disabled={currentSearchString === undefined || currentSearchString.length === 0}
                >
                  Clear selection
                </Button>
              )}
            </>
          )}

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
  );
};

export default VisualisationToolbar;
