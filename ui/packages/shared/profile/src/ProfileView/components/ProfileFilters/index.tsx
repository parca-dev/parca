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

import {Button, Input, Select, type SelectItem} from '@parca/components';

import {filterPresets, getPresetByKey, isPresetKey} from './filterPresets';
import {useProfileFilters, type ProfileFilter} from './useProfileFilters';

export const isFilterComplete = (filter: ProfileFilter): boolean => {
  // For preset filters, only need type and value
  if (filter.type != null && isPresetKey(filter.type)) {
    return filter.value !== '' && filter.type != null;
  }
  // For regular filters, need all fields
  return (
    filter.value !== '' && filter.type != null && filter.field != null && filter.matchType != null
  );
};

const filterTypeItems: SelectItem[] = [
  {
    key: 'stack',
    element: {
      active: <>Stack Filter</>,
      expanded: (
        <>
          <span>Stack Filter</span>
          <br />
          <span className="text-xs">Filters entire call stacks</span>
        </>
      ),
    },
  },
  {
    key: 'frame',
    element: {
      active: <>Frame Filter</>,
      expanded: (
        <>
          <span>Frame Filter</span>
          <br />
          <span className="text-xs">Filters individual frames</span>
        </>
      ),
    },
  },
  ...filterPresets.map(preset => ({
    key: preset.key,
    element: {
      active: <>{preset.name}</>,
      expanded: (
        <>
          <span>{preset.name}</span>
          <br />
          <span className="text-xs">{preset.description}</span>
        </>
      ),
    },
  })),
];

const fieldItems: SelectItem[] = [
  {
    key: 'function_name',
    element: {
      active: <>Function</>,
      expanded: <>Function Name</>,
    },
  },
  {
    key: 'binary',
    element: {
      active: <>Binary</>,
      expanded: <>Binary/Executable Name</>,
    },
  },
  {
    key: 'system_name',
    element: {
      active: <>System Name</>,
      expanded: <>System Name</>,
    },
  },
  {
    key: 'filename',
    element: {
      active: <>Filename</>,
      expanded: <>Source Filename</>,
    },
  },
  {
    key: 'address',
    element: {
      active: <>Address</>,
      expanded: <>Memory Address</>,
    },
  },
  {
    key: 'line_number',
    element: {
      active: <>Line Number</>,
      expanded: <>Source Line Number</>,
    },
  },
];

const stringMatchTypeItems: SelectItem[] = [
  {
    key: 'equal',
    element: {
      active: <>Equals</>,
      expanded: <>Equals</>,
    },
  },
  {
    key: 'not_equal',
    element: {
      active: <>Not Equals</>,
      expanded: <>Not Equals</>,
    },
  },
  {
    key: 'contains',
    element: {
      active: <>Contains</>,
      expanded: <>Contains</>,
    },
  },
  {
    key: 'not_contains',
    element: {
      active: <>Not Contains</>,
      expanded: <>Not Contains</>,
    },
  },
];

const numberMatchTypeItems: SelectItem[] = [
  {
    key: 'equal',
    element: {
      active: <>Equals</>,
      expanded: <>Equals</>,
    },
  },
  {
    key: 'not_equal',
    element: {
      active: <>Not Equals</>,
      expanded: <>Not Equals</>,
    },
  },
];

const ProfileFilters = (): JSX.Element => {
  const {
    localFilters,
    appliedFilters,
    hasUnsavedChanges,
    onApplyFilters,
    addFilter,
    removeFilter,
    updateFilter,
    resetFilters,
  } = useProfileFilters();

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        if (e.currentTarget.value.trim() === '') {
          return;
        }
        onApplyFilters();
      }
    },
    [onApplyFilters]
  );

  const filtersToRender = localFilters.length > 0 ? localFilters : appliedFilters ?? [];

  return (
    <div className="flex gap-2 w-full">
      <div className="flex-1 flex flex-wrap gap-2">
        {filtersToRender.map(filter => {
          const isNumberField = filter.field === 'address' || filter.field === 'line_number';
          const matchTypeItems = isNumberField ? numberMatchTypeItems : stringMatchTypeItems;
          const isPresetFilter = filter.type != null && isPresetKey(filter.type);

          return (
            <div key={filter.id} className="flex items-center gap-0">
              <Select
                items={filterTypeItems}
                selectedKey={filter.type}
                placeholder="Select Filter"
                onSelection={key => {
                  // Check if this is a preset selection
                  if (isPresetKey(key)) {
                    const preset = getPresetByKey(key);
                    if (preset != null) {
                      updateFilter(filter.id, {
                        type: preset.key,
                        field: undefined,
                        matchType: undefined,
                        value: preset.name,
                      });
                    }
                  } else {
                    const newType = key as 'stack' | 'frame';

                    // Check if we're converting a preset filter to a regular filter
                    if (filter.type != null && isPresetKey(filter.type)) {
                      updateFilter(filter.id, {
                        type: newType,
                        field: 'function_name',
                        matchType: 'contains',
                        value: '',
                      });
                    } else {
                      updateFilter(filter.id, {
                        type: newType,
                        field: filter.field ?? 'function_name',
                        matchType: filter.matchType ?? 'contains',
                      });
                    }
                  }
                }}
                className={cx(
                  'rounded-l-md pr-1 gap-0 focus:z-50 focus:relative focus:outline-1',
                  isPresetFilter ? 'rounded-r-none border-r-0' : 'rounded-r-none',
                  filter.type != null ? 'border-r-0 w-auto' : 'w-32'
                )}
              />

              {filter.type != null && !isPresetFilter && (
                <>
                  <Select
                    items={fieldItems}
                    selectedKey={filter.field ?? ''}
                    onSelection={key => {
                      const newField = key as ProfileFilter['field'];
                      const isNewFieldNumber = newField === 'address' || newField === 'line_number';
                      const isCurrentFieldNumber =
                        filter.field === 'address' || filter.field === 'line_number';

                      if (isNewFieldNumber !== isCurrentFieldNumber) {
                        updateFilter(filter.id, {
                          field: newField,
                          matchType: 'equal',
                        });
                      } else {
                        updateFilter(filter.id, {field: newField});
                      }
                    }}
                    className="rounded-none border-r-0 w-32 pr-1 gap-0 focus:z-50 focus:relative focus:outline-1"
                  />

                  <Select
                    items={matchTypeItems}
                    selectedKey={filter.matchType ?? ''}
                    onSelection={key =>
                      updateFilter(filter.id, {matchType: key as ProfileFilter['matchType']})
                    }
                    className="rounded-none border-r-0 pr-1 gap-0 focus:z-50 focus:relative focus:outline-1"
                  />

                  <Input
                    placeholder="Value"
                    value={filter.value}
                    onChange={e => updateFilter(filter.id, {value: e.target.value})}
                    onKeyDown={handleKeyDown}
                    className="rounded-none w-36 text-sm focus:outline-1"
                  />
                </>
              )}

              <Button
                variant="neutral"
                onClick={() => {
                  // If we're displaying local filters and this is the last one, reset everything
                  if (localFilters.length > 0 && localFilters.length === 1) {
                    resetFilters();
                  }
                  // If we're displaying applied filters and this is the last one, reset everything
                  else if (localFilters.length === 0 && filtersToRender.length === 1) {
                    resetFilters();
                  }
                  // Otherwise, just remove this specific filter
                  else {
                    removeFilter(filter.id);
                  }
                }}
                className={cx(
                  'h-[38px] p-3',
                  filter.type != null ? 'rounded-none rounded-r-md' : 'rounded-l-none rounded-r-md'
                )}
              >
                <Icon icon="mdi:close" className="h-4 w-4" />
              </Button>
            </div>
          );
        })}

        {localFilters.length > 0 && (
          <Button variant="neutral" onClick={addFilter} className="p-3 h-[38px]">
            <Icon icon="mdi:filter-plus-outline" className="h-4 w-4" />
          </Button>
        )}

        {localFilters.length === 0 && (appliedFilters?.length ?? 0) === 0 && (
          <Button variant="neutral" onClick={addFilter} className="flex items-center gap-2">
            <Icon icon="mdi:filter-outline" className="h-4 w-4" />
            <span>Filter</span>
          </Button>
        )}
      </div>

      {localFilters.length > 0 && hasUnsavedChanges && localFilters.some(isFilterComplete) && (
        <Button
          variant="primary"
          onClick={onApplyFilters}
          className={cx('flex items-center gap-2 self-end')}
        >
          <span>Apply</span>
        </Button>
      )}
    </div>
  );
};

export default ProfileFilters;
