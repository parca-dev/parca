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

import {useCallback, useState} from 'react';

import ReactSelect, {type MenuListProps, type Props as ReactSelectProps} from 'react-select';

import {RefreshButton} from '@parca/components';

export interface SelectWithRefreshProps<Option, IsMulti extends boolean>
  extends ReactSelectProps<Option, IsMulti> {
  onRefresh?: () => Promise<void>;
  refreshTitle?: string;
  refreshTestId?: string;
  menuTestId?: string;
}

export function SelectWithRefresh<Option, IsMulti extends boolean = false>(
  props: SelectWithRefreshProps<Option, IsMulti>
): JSX.Element {
  const {
    onRefresh,
    refreshTitle = 'Refresh label names',
    refreshTestId = 'select-refresh-button',
    menuTestId,
    components,
    ...selectProps
  } = props;

  const [isRefreshing, setIsRefreshing] = useState(false);

  const handleRefetch = useCallback(async () => {
    if (onRefresh == null || isRefreshing) return;

    setIsRefreshing(true);
    try {
      await onRefresh();
    } catch (error) {
      console.error('Error during refresh:', error);
    } finally {
      setIsRefreshing(false);
    }
  }, [onRefresh, isRefreshing]);

  const MenuListWithRefresh = useCallback(
    ({children, innerProps}: MenuListProps<Option, IsMulti>) => {
      const testIdProps = menuTestId != null ? {'data-testid': menuTestId} : {};

      return (
        <div className="flex flex-col" style={{maxHeight: '332px'}}>
          <div
            className="overflow-y-auto flex-1"
            {...innerProps}
            {...testIdProps}
            style={{...innerProps?.style, fontSize: '14px'}}
          >
            {children}
          </div>
          {onRefresh != null && (
            <RefreshButton
              onClick={() => void handleRefetch()}
              disabled={isRefreshing}
              title={refreshTitle}
              testId={refreshTestId}
            />
          )}
        </div>
      );
    },
    [onRefresh, isRefreshing, handleRefetch, refreshTitle, refreshTestId, menuTestId]
  );

  const combinedLoadingState = isRefreshing || (selectProps.isLoading ?? false);

  return (
    <ReactSelect<Option, IsMulti>
      {...selectProps}
      isLoading={combinedLoadingState}
      components={{
        ...components,
        // eslint-disable-next-line react/display-name
        MenuList: MenuListWithRefresh,
      }}
    />
  );
}

export default SelectWithRefresh;
