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

import {Icon} from '@iconify/react';
import cx from 'classnames';
import {Item, Menu} from 'react-contexify';

import 'react-contexify/dist/ReactContexify.css';

import {useParcaContext, useURLState} from '@parca/components';

import {type Row} from '.';

interface TableContextMenuProps {
  menuId: string;
  row: Row | null;
}

const TableContextMenu = ({menuId, row}: TableContextMenuProps): React.JSX.Element => {
  const [_, setSandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const [dashboardItems, setDashboardItems] = useURLState<string[]>('dashboard_items', {
    alwaysReturnArray: true,
  });
  const {enableSandwichView, isDarkMode} = useParcaContext();

  const onSandwichViewSelect = (): void => {
    if (row?.name != null && row.name.length > 0) {
      setSandwichFunctionName(row.name.trim());
      if (!dashboardItems.includes('sandwich')) {
        setDashboardItems([...dashboardItems, 'sandwich']);
      }
    }
  };

  const isMenuDisabled = row === null || enableSandwichView !== true;

  return (
    <Menu
      id={menuId}
      theme={isDarkMode ? 'dark' : ''}
      className={cx(
        dashboardItems.includes('sandwich') ? 'min-w-[350px] w-[350px]' : 'min-w-[260px] w-[260px]'
      )}
    >
      <Item id="sandwich-view" onClick={onSandwichViewSelect} disabled={isMenuDisabled}>
        <div className="flex w-full items-center gap-2">
          <Icon icon="tdesign:sandwich-filled" />
          <div className="relative">
            {dashboardItems.includes('sandwich')
              ? 'Focus sandwich on this frame.'
              : 'Show in sandwich'}
            <span className="absolute top-[-2px] text-xs lowercase text-red-500">&nbsp;alpha</span>
          </div>
        </div>
      </Item>
    </Menu>
  );
};

export default TableContextMenu;
