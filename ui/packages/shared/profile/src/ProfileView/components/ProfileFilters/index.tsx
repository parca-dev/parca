// Copyright 2025 The Parca Authors
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

import { useCallback } from 'react';
import { Icon } from '@iconify/react';
import { Button, Input, Select, type SelectItem } from '@parca/components';
import { useProfileFilters, type ProfileFilter } from './useProfileFilters';

interface ProfileFiltersProps {
    onFiltersChange?: (filters: ProfileFilter[]) => void;
}

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
            active: <>=</>,
            expanded: <>= Equals</>,
        },
    },
    {
        key: 'not_equal',
        element: {
            active: <>!=</>,
            expanded: <>!= Not Equals</>,
        },
    },
    {
        key: 'contains',
        element: {
            active: <>=~</>,
            expanded: <>=~ Contains</>,
        },
    },
    {
        key: 'not_contains',
        element: {
            active: <>!~</>,
            expanded: <>!~ Not Contains</>,
        },
    },
];

const numberMatchTypeItems: SelectItem[] = [
    {
        key: 'equal',
        element: {
            active: <>=</>,
            expanded: <>= Equals</>,
        },
    },
    {
        key: 'not_equal',
        element: {
            active: <>!=</>,
            expanded: <>!= Not Equals</>,
        },
    },
];


const ProfileFilters = ({ onFiltersChange }: ProfileFiltersProps): JSX.Element => {
    const {
        localFilters,
        appliedFilters,
        hasUnsavedChanges,
        isClearAction,
        onApplyFilters,
        addFilter,
        removeFilter,
        updateFilter,
        resetFilters,
    } = useProfileFilters({ onFiltersChange });

    const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            if (e.currentTarget.value.trim() === '') {
                return;
            }
            onApplyFilters();
        }
    }, [onApplyFilters]);

    return (
        <div className="flex gap-2 w-full">
            <div className="flex-1 flex flex-wrap gap-2">
                {localFilters.map((filter) => {
                    const isNumberField = filter.field === 'address' || filter.field === 'line_number';
                    const matchTypeItems = isNumberField ? numberMatchTypeItems : stringMatchTypeItems;

                    return (
                        <div key={filter.id} className="flex items-center gap-0">
                            <Select
                                items={filterTypeItems}
                                selectedKey={filter.type}
                                onSelection={(key) => updateFilter(filter.id, { type: key as 'stack' | 'frame' })}
                                className="rounded-l-md rounded-r-none border-r-0 w-28 pr-1 gap-0"
                            />

                            <Select
                                items={fieldItems}
                                selectedKey={filter.field}
                                onSelection={(key) => {
                                    const newField = key as ProfileFilter['field'];
                                    const isNewFieldNumber = newField === 'address' || newField === 'line_number';
                                    const isCurrentFieldNumber = filter.field === 'address' || filter.field === 'line_number';

                                    if (isNewFieldNumber !== isCurrentFieldNumber) {
                                        updateFilter(filter.id, {
                                            field: newField,
                                            matchType: 'equal'
                                        });
                                    } else {
                                        updateFilter(filter.id, { field: newField });
                                    }
                                }}
                                className="rounded-none border-r-0 w-32 pr-1 gap-0"
                            />

                            <Select
                                items={matchTypeItems}
                                selectedKey={filter.matchType}
                                onSelection={(key) => updateFilter(filter.id, { matchType: key as ProfileFilter['matchType'] })}
                                className="rounded-none border-r-0 w-16 pr-1 gap-0"
                            />

                            <Input
                                placeholder={filter.field === 'address' || filter.field === 'line_number' ? 'Number' : 'Value'}
                                value={filter.value}
                                onChange={(e) => updateFilter(filter.id, { value: e.target.value })}
                                onKeyDown={handleKeyDown}
                                className="rounded-r-md w-36 text-sm"
                            />

                            <Button
                                variant="neutral"
                                onClick={() => {
                                    if (localFilters.length === 1) {
                                        resetFilters();
                                    } else {
                                        removeFilter(filter.id);
                                    }
                                }}
                                className="ml-2"
                            >
                                <Icon icon="mdi:close" className="h-4 w-4" />
                            </Button>

                        </div>
                    );
                })}

                {localFilters.length > 0 && (
                    <Button
                        variant="neutral"
                        onClick={addFilter}
                        className=""
                    >
                        <Icon icon="mdi:plus" className="h-4 w-4" />
                    </Button>
                )}

                {localFilters.length === 0 && !appliedFilters?.length && (
                    <Button
                        variant="secondary"
                        onClick={addFilter}
                        className="flex items-center gap-2"
                    >
                        <Icon icon="mdi:filter-plus" className="h-4 w-4" />
                        <span>Add Filter</span>
                    </Button>
                )}
            </div>

            {(localFilters.length > 0 || (appliedFilters && appliedFilters.length > 0)) && (
                <Button
                    variant={isClearAction ? "secondary" : "primary"}
                    onClick={onApplyFilters}
                    className="flex items-center gap-2 self-end"
                    disabled={!isClearAction && !hasUnsavedChanges}
                >
                    <Icon
                        icon={isClearAction ? "ep:circle-close" : "ep:arrow-right"}
                        className="h-4 w-4"
                    />
                    <span>{isClearAction ? 'Clear Filters' : 'Apply Filters'}</span>
                </Button>
            )}
        </div>
    );
};

export default ProfileFilters;
