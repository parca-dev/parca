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

import {useParcaContext, useURLState, useURLStateBatch} from '@parca/components';

import {ProfileSource} from '../../../ProfileSource';
import Dropdown, {DropdownElement, InnerAction} from './Dropdown';

interface Props {
  profileSource?: ProfileSource;
}

const ViewSelector = ({profileSource}: Props): JSX.Element => {
  const [dashboardItems = ['flamegraph'], setDashboardItems] = useURLState<string[]>(
    'dashboard_items',
    {
      alwaysReturnArray: true,
    }
  );
  const [, setSandwichFunctionName] = useURLState<string | undefined>('sandwich_function_name');
  const {enableSourcesView, enableSandwichView} = useParcaContext();
  const batchUpdates = useURLStateBatch();

  const allItems: Array<{
    key: string;
    label?: string | ReactNode;
    canBeSelected: boolean;
    supportingText?: string;
    disabledText?: string;
  }> = [
    {
      key: 'flamegraph',
      label: 'Flame Graph',
      canBeSelected: !dashboardItems.includes('flamegraph'),
    },
    {key: 'table', label: 'Table', canBeSelected: !dashboardItems.includes('table')},
    {
      key: 'flamechart',
      label: (
        <span className="relative">
          Flame Chart
          <span className="absolute top-[-2px] text-xs lowercase text-red-500">&nbsp;alpha</span>
        </span>
      ),
      canBeSelected:
        !dashboardItems.includes('flamechart') && profileSource?.ProfileType().delta === true,
      disabledText:
        !dashboardItems.includes('flamechart') && profileSource?.ProfileType().delta !== true
          ? 'Flamechart is not available for non-delta profiles'
          : undefined,
    },
  ];

  if (enableSandwichView === true) {
    allItems.push({
      key: 'sandwich',
      label: (
        <span className="relative">
          Sandwich
          <span className="absolute top-[-2px] text-xs lowercase text-red-500">&nbsp;alpha</span>
        </span>
      ),
      canBeSelected: !dashboardItems.includes('sandwich'),
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
      <span className="capitalize whitespace-nowrap">
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

    // If we already have 2 panels and this item isn't selected, don't show any action
    if (dashboardItems.length >= 2 && !dashboardItems.includes(item.key)) return undefined;

    return {
      text:
        !item.canBeSelected && item.key === 'source'
          ? 'Add Panel'
          : item.canBeSelected
          ? 'Add Panel'
          : dashboardItems.includes(item.key)
          ? 'Close Panel'
          : 'Add Panel',
      onClick: () => {
        if (item.canBeSelected) {
          setDashboardItems([...dashboardItems, item.key]);
        } else {
          const newDashboardItems = dashboardItems.filter(v => v !== item.key);

          // Batch updates when removing sandwich panel to combine both URL changes
          if (item.key === 'sandwich') {
            batchUpdates(() => {
              setDashboardItems(newDashboardItems);
              setSandwichFunctionName(undefined);
            });
          } else {
            setDashboardItems(newDashboardItems);
          }
        }
      },
      isDisabled: dashboardItems.length === 1 && dashboardItems.includes('sandwich'),
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

    if (isOnlyChart && value === 'sandwich') {
      setDashboardItems([...dashboardItems, value]);
      return;
    }

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
      optionsClassName="min-w-[260px]"
    />
  );
};

export default ViewSelector;
