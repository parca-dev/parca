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

import {ReactNode} from 'react';

import {useParcaContext, useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';

import {ProfileSource} from '../../../ProfileSource';
import Dropdown, {DropdownElement, InnerAction} from './Dropdown';

interface Props {
  profileSource?: ProfileSource;
}

const ViewSelector = ({profileSource}: Props): JSX.Element => {
  const [dashboardItems = ['icicle'], setDashboardItems] = useURLState<string[]>(
    'dashboard_items',
    {
      alwaysReturnArray: true,
    }
  );
  const {enableSourcesView} = useParcaContext();

  const [enableicicleCharts] = useUserPreference<boolean>(USER_PREFERENCES.ENABLE_ICICLECHARTS.key);

  const allItems: Array<{
    key: string;
    label?: string | ReactNode;
    canBeSelected: boolean;
    supportingText?: string;
    disabledText?: string;
  }> = [
    {key: 'table', label: 'Table', canBeSelected: !dashboardItems.includes('table')},
    {key: 'icicle', label: 'icicle', canBeSelected: !dashboardItems.includes('icicle')},
    {key: 'sandwich', label: 'sandwich', canBeSelected: !dashboardItems.includes('sandwich')},
  ];
  if (enableicicleCharts) {
    allItems.push({
      key: 'iciclechart',
      label: (
        <span className="relative">
          Iciclechart
          <span className="absolute top-[-2px] text-xs lowercase text-red-500">&nbsp;alpha</span>
        </span>
      ),
      canBeSelected:
        !dashboardItems.includes('iciclechart') && profileSource?.ProfileType().delta === true,
      disabledText:
        !dashboardItems.includes('iciclechart') && profileSource?.ProfileType().delta !== true
          ? 'Iciclechart is not available for non-delta profiles'
          : undefined,
    });
  }

  if (enableSourcesView === true) {
    allItems.push({key: 'source', label: 'Source', canBeSelected: false});
  }

  const getOption = ({
    label,
    supportingText,
  }: {
    key: string;
    label?: string | ReactNode;
    supportingText?: string;
  }): DropdownElement => {
    const title = (
      <span className="capitalize">
        {typeof label === 'string' ? label.replaceAll('-', ' ') : label}
      </span>
    );

    return {
      active: title,
      expanded: (
        <>
          {title}
          {supportingText !== null && <span className="text-xs">{supportingText}</span>}
        </>
      ),
    };
  };

  const getInnerActionForItem = (item: {
    key: string;
    canBeSelected: boolean;
  }): InnerAction | undefined => {
    if (dashboardItems.length === 1 && item.key === dashboardItems[0]) return undefined;

    // For sandwich view, return a no-op action
    if (item.key === 'sandwich') {
      return {
        text: 'Add Panel',
        onClick: () => {},
        isDisabled: true, // Custom property to control button state
      };
    }

    return {
      text:
        !item.canBeSelected && item.key === 'source'
          ? 'Add Panel'
          : item.canBeSelected
          ? 'Add Panel'
          : 'Close Panel',
      onClick: () => {
        if (item.canBeSelected) {
          setDashboardItems([...dashboardItems, item.key]);
        } else {
          setDashboardItems(dashboardItems.filter(v => v !== item.key));
        }
      },
    };
  };

  const items = allItems.map(item => ({
    key: item.key,
    disabled: !item.canBeSelected,
    disabledText: item.disabledText,
    element: getOption(item),
    innerAction: getInnerActionForItem(item),
  }));

  const onSelection = (value: string): void => {
    const isOnlyChart = dashboardItems.length === 1;
    if (isOnlyChart) {
      setDashboardItems([value]);
      return;
    }

    const newDashboardItems = [dashboardItems[0], value];

    setDashboardItems(newDashboardItems);
  };

  return (
    <Dropdown
      className="h-view-selector"
      items={items}
      selectedKey={dashboardItems.length >= 2 ? 'Multiple' : dashboardItems[0]}
      onSelection={onSelection}
      placeholder={'Select view type...'}
      id="h-view-selector"
    />
  );
};

export default ViewSelector;
