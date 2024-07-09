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

import {Select, useParcaContext, useURLStateNew, type SelectElement} from '@parca/components';
import {useUIFeatureFlag} from '@parca/hooks';

interface Props {
  position: number;
  defaultValue: string;
  placeholderText?: string;
  primary?: boolean;
  addView?: boolean;
  disabled?: boolean;
  icon?: JSX.Element;
  id?: string;
}

const ViewSelector = ({
  defaultValue,
  position,
  placeholderText,
  primary = false,
  addView = false,
  disabled = false,
  icon,
  id,
}: Props): JSX.Element => {
  const [callgraphEnabled] = useUIFeatureFlag('callgraph');
  const [dashboardItems = ['icicle'], setDashboardItems] = useURLStateNew<string[]>(
    'dashboard_items',
    {
      alwaysReturnArray: true,
    }
  );
  const {enableSourcesView} = useParcaContext();

  const allItems: Array<{key: string; canBeSelected: boolean; supportingText?: string}> = [
    {key: 'table', canBeSelected: dashboardItems.includes('table') === false},
    {key: 'icicle', canBeSelected: dashboardItems.includes('icicle') === false},
  ];
  if (enableSourcesView === true) {
    allItems.push({key: 'source', canBeSelected: false});
  }
  if (callgraphEnabled) {
    allItems.push({
      key: 'callgraph',
      canBeSelected: dashboardItems.includes('callgraph') === false,
    });
  }

  const getOption = ({
    key,
    supportingText,
  }: {
    key: string;
    supportingText?: string;
  }): SelectElement => {
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

  const items = allItems.map(item => ({
    key: item.key,
    disabled: !item.canBeSelected,
    element: getOption(item),
  }));

  const onSelection = (value: string): void => {
    if (addView) {
      setDashboardItems([dashboardItems[0], value]);
      return;
    }

    const isOnlyChart = dashboardItems.length === 1;
    if (isOnlyChart) {
      setDashboardItems([value]);
      return;
    }

    // Note: this will need to be updated if we ever have more more than 2 panels
    const isFirstChart = position === 0;
    const newDashboardItems = isFirstChart
      ? [value, dashboardItems[1]]
      : [dashboardItems[0], value];

    setDashboardItems(newDashboardItems);
  };

  return (
    <Select
      className="h-view-selector"
      items={items}
      selectedKey={defaultValue}
      onSelection={onSelection}
      placeholder={placeholderText ?? 'Select view type...'}
      primary={primary}
      disabled={disabled}
      icon={icon}
      id={id}
    />
  );
};

export default ViewSelector;
