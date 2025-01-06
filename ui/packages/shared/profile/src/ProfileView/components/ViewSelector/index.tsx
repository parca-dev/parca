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

import {useParcaContext, useURLState} from '@parca/components';

import Dropdown, {DropdownElement, InnerAction} from './Dropdown';

const ViewSelector = (): JSX.Element => {
  const [dashboardItems = ['icicle'], setDashboardItems] = useURLState<string[]>(
    'dashboard_items',
    {
      alwaysReturnArray: true,
    }
  );
  const {enableSourcesView, enableIciclechartView} = useParcaContext();

  const allItems: Array<{key: string; canBeSelected: boolean; supportingText?: string}> = [
    {key: 'table', canBeSelected: !dashboardItems.includes('table')},
    {key: 'icicle', canBeSelected: !dashboardItems.includes('icicle')},
  ];
  if (enableIciclechartView === true) {
    allItems.push({key: 'iciclechart', canBeSelected: !dashboardItems.includes('iciclechart')});
  }

  if (enableSourcesView === true) {
    allItems.push({key: 'source', canBeSelected: false});
  }

  const getOption = ({
    key,
    supportingText,
  }: {
    key: string;
    supportingText?: string;
  }): DropdownElement => {
    const title = <span className="capitalize">{key.replaceAll('-', ' ')}</span>;

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
