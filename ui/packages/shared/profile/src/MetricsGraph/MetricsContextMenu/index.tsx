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
import {Item, Menu, Submenu} from 'react-contexify';

import {useParcaContext} from '@parca/components';
import {testId} from '@parca/test-utils';

import {Series, SeriesPoint} from '../';

export interface ContextMenuItem {
  id: string;
  label: React.ReactNode;
  icon?: string;
  onClick: (closestPoint: SeriesPoint | null, series: Series[]) => void;
  disabled?: (closestPoint: SeriesPoint | null, series: Series[]) => boolean;
}

export interface ContextMenuSubmenu {
  id: string;
  label: React.ReactNode;
  icon?: string;
  items?: ContextMenuItem[];
  createDynamicItems?: (closestPoint: SeriesPoint | null, series: Series[]) => ContextMenuItem[];
}

export type ContextMenuItemOrSubmenu = ContextMenuItem | ContextMenuSubmenu;

interface MetricsContextMenuProps {
  menuId: string;
  closestPoint: SeriesPoint | null;
  series: Series[];
  trackVisibility: (isVisible: boolean) => void;
  menuItems: ContextMenuItemOrSubmenu[];
}

const MetricsContextMenu = ({
  menuId,
  closestPoint,
  series,
  trackVisibility,
  menuItems,
}: MetricsContextMenuProps): JSX.Element => {
  const {isDarkMode} = useParcaContext();

  const renderMenuItem = (item: ContextMenuItemOrSubmenu): React.ReactNode => {
    if ('items' in item || 'createDynamicItems' in item) {
      // This is a submenu
      const submenu = item;
      const items =
        submenu.createDynamicItems != null
          ? submenu.createDynamicItems(closestPoint, series)
          : submenu.items ?? [];

      return (
        <Submenu
          key={submenu.id}
          label={
            <div className="flex w-full items-center gap-2">
              {submenu.icon != null && submenu.icon !== '' && <Icon icon={submenu.icon} />}
              <div>{submenu.label}</div>
            </div>
          }
        >
          <div className="max-h-[300px] overflow-auto">
            {items.map(subItem => (
              <Item
                key={subItem.id}
                id={subItem.id}
                onClick={() => subItem.onClick(closestPoint, series)}
                disabled={subItem.disabled?.(closestPoint, series) ?? false}
                className="max-w-[400px] overflow-hidden"
              >
                <div className="flex w-full items-center gap-2">
                  {subItem.icon != null && subItem.icon !== '' && <Icon icon={subItem.icon} />}
                  <div>{subItem.label}</div>
                </div>
              </Item>
            ))}
          </div>
        </Submenu>
      );
    } else {
      // This is a regular menu item
      const menuItem = item as ContextMenuItem;
      return (
        <Item
          key={menuItem.id}
          id={menuItem.id}
          onClick={() => menuItem.onClick(closestPoint, series)}
          disabled={menuItem.disabled?.(closestPoint, series) ?? false}
        >
          <div className="flex w-full items-center gap-2">
            {menuItem.icon != null && menuItem.icon !== '' && <Icon icon={menuItem.icon} />}
            <div>{menuItem.label}</div>
          </div>
        </Item>
      );
    }
  };

  return (
    <Menu 
      id={menuId} 
      onVisibilityChange={trackVisibility} 
      theme={isDarkMode ? 'dark' : ''}
      {...testId('METRICS_GRAPH_CONTEXT_MENU')}
    >
      {menuItems.map(renderMenuItem)}
    </Menu>
  );
};

export default MetricsContextMenu;
