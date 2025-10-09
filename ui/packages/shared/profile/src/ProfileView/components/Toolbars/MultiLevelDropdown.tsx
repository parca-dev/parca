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

import React, {useCallback, useEffect, useRef, useState} from 'react';

import {Menu} from '@headlessui/react';
import {Icon} from '@iconify/react';
import cx from 'classnames';

import {useURLState} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {ProfileType} from '@parca/parser';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../../ProfileFlameGraph/FlameGraphArrow';
import {useProfileViewContext} from '../../context/ProfileViewContext';
import SwitchMenuItem from './SwitchMenuItem';

interface MenuItemType {
  label: string;
  items?: MenuItemType[];
  onclick?: () => void;
  hide?: boolean;
  id?: string;
  disabled?: boolean;
  active?: boolean;
  value?: string;
  icon?: string;
  customSubmenu?: React.ReactNode;
  renderAsDiv?: boolean;
}

type MenuItemProps = MenuItemType & {
  onSelect: (path: string[]) => void;
  path?: string[];
  closeDropdown: () => void;
  isNested?: boolean;
  activeValueForSortBy?: string;
  activeValueForColorBy?: string;
  activeValuesForLevel?: string[];
  icon?: string;
};

const MenuItem: React.FC<MenuItemProps> = ({
  label,
  items,
  onclick,
  onSelect,
  path = [],
  id,
  closeDropdown,
  isNested = false,
  activeValueForSortBy,
  activeValueForColorBy,
  activeValuesForLevel,
  value,
  disabled = false,
  icon,
  customSubmenu,
  renderAsDiv = false,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);
  const [shouldOpenLeft, setShouldOpenLeft] = useState(false);

  useEffect(() => {
    if (items !== undefined && menuRef.current !== null) {
      const rect = menuRef.current.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const menuWidth = 224; // w-56 = 14rem = 224px
      const spaceOnRight = viewportWidth - rect.right;
      const spaceOnLeft = rect.left;

      // Open to the left if there's not enough space on the right but enough on the left
      setShouldOpenLeft(spaceOnRight < menuWidth && spaceOnLeft >= menuWidth);
    }
  }, [items]);
  let isActive = false;

  if (isNested) {
    if (activeValueForSortBy !== undefined && value === activeValueForSortBy) {
      isActive = true;
    }
    if (activeValueForColorBy !== undefined && value === activeValueForColorBy) {
      isActive = true;
    }
    if (activeValuesForLevel?.includes(value ?? '') ?? false) {
      isActive = true;
    }
  }

  const handleSelect = (): void => {
    if (items === undefined) {
      if (onclick !== undefined) {
        onclick();
        closeDropdown();
      } else {
        onSelect([...path, label]);
        closeDropdown();
      }
    }
  };

  return (
    <div className="relative" ref={menuRef}>
      <Menu>
        {({close}) => (
          <>
            <Menu.Button
              as={renderAsDiv ? 'div' : 'button'}
              className={`w-full text-left px-4 py-2 text-sm ${
                disabled
                  ? 'text-gray-400'
                  : isActive
                  ? 'text-white bg-indigo-400 hover:text-white'
                  : 'text-white-600 hover:bg-indigo-600 hover:text-white'
              } flex justify-between items-center`}
              onClick={handleSelect}
              id={id}
              disabled={disabled}
            >
              {customSubmenu !== undefined ? (
                customSubmenu
              ) : (
                <span className="flex items-center">
                  <div className="flex items-center gap-2">
                    {icon !== undefined && <Icon icon={icon} className="h-4 w-4" />}
                    <span>{label}</span>
                  </div>
                  {isActive && <Icon icon="heroicons-solid:check" className="ml-2 h-4 w-4" />}
                </span>
              )}
              {items !== undefined && (
                <Icon icon="flowbite:caret-right-solid" className="h-[14px] w-[14px]" />
              )}
            </Menu.Button>
            {items !== undefined && (
              <Menu.Items
                className={`absolute top-0 w-56 mt-0 bg-white border border-gray-200 rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:bg-gray-900 dark:border-gray-600 ${
                  shouldOpenLeft
                    ? 'right-full mr-1 origin-top-left'
                    : 'left-full ml-1 origin-top-right'
                }`}
              >
                {items?.map((item, index) => (
                  <MenuItem
                    key={index}
                    {...item}
                    onSelect={selectedPath => {
                      onSelect([...path, ...selectedPath]);
                      close();
                      closeDropdown();
                    }}
                    path={[...path, label]}
                    closeDropdown={closeDropdown}
                    isNested={true}
                    activeValueForSortBy={activeValueForSortBy}
                    activeValueForColorBy={activeValueForColorBy}
                    activeValuesForLevel={activeValuesForLevel}
                  />
                ))}
              </Menu.Items>
            )}
          </>
        )}
      </Menu>
    </div>
  );
};

interface MultiLevelDropdownProps {
  onSelect: (path: string[]) => void;
  profileType?: ProfileType;
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
  isTableVizOnly: boolean;
  alignFunctionName: string;
  setAlignFunctionName: (align: string) => void;
  colorBy: string;
  setColorBy: (colorBy: string) => void;
}

const MultiLevelDropdown: React.FC<MultiLevelDropdownProps> = ({
  onSelect,
  profileType,
  groupBy,
  toggleGroupBy,
  isTableVizOnly,
  alignFunctionName,
  setAlignFunctionName,
  colorBy,
  setColorBy,
}) => {
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [shouldOpenLeft, setShouldOpenLeft] = useState(false);
  const [storeSortBy] = useURLState('sort_by', {
    defaultValue: FIELD_FUNCTION_NAME,
  });
  const [colorStackLegend, setStoreColorStackLegend] = useURLState('color_stack_legend');
  const [hiddenBinaries, setHiddenBinaries] = useURLState('hidden_binaries', {
    defaultValue: [],
    alwaysReturnArray: true,
  });
  const {compareMode} = useProfileViewContext();
  const [colorProfileName] = useUserPreference<string>(
    USER_PREFERENCES.FLAMEGRAPH_COLOR_PROFILE.key
  );
  const isColorStackLegendEnabled = colorStackLegend === 'true';
  const isLeftAligned = alignFunctionName === 'left';

  // By default, we want delta profiles (CPU) to be relatively compared.
  // For non-delta profiles, like goroutines or memory, we want the profiles to be compared absolutely.
  const compareAbsoluteDefault = profileType?.delta === false ? 'true' : 'false';

  const [compareAbsolute = compareAbsoluteDefault, setCompareAbsolute] =
    useURLState('compare_absolute');
  const isCompareAbsolute = compareAbsolute === 'true';

  useEffect(() => {
    const checkOverflow = (): void => {
      if (dropdownRef.current !== null) {
        const rect = dropdownRef.current.getBoundingClientRect();
        const viewportWidth = window.innerWidth;
        const menuWidth = isTableVizOnly ? 256 : 320; // w-64 = 256px, w-80 = 320px
        const spaceOnRight = viewportWidth - rect.right;
        const spaceOnLeft = rect.left;

        setShouldOpenLeft(spaceOnRight < menuWidth && spaceOnLeft >= menuWidth);
      }
    };

    checkOverflow();
    window.addEventListener('resize', checkOverflow);
    return () => window.removeEventListener('resize', checkOverflow);
  }, [isTableVizOnly]);

  const handleBinaryToggle = (index: number): void => {
    const updatedBinaries = [...(hiddenBinaries as string[])];
    updatedBinaries.splice(index, 1);
    setHiddenBinaries(updatedBinaries);
  };

  const setColorStackLegend = useCallback(
    (value: string): void => {
      setStoreColorStackLegend(value);
    },
    [setStoreColorStackLegend]
  );

  const resetLegend = (): void => {
    setHiddenBinaries([]);
  };

  const menuItems: MenuItemType[] = [
    {
      label: 'Levels',
      id: 'h-levels-filter',
      items: [
        {
          label: 'Function',
          onclick: () => toggleGroupBy(FIELD_FUNCTION_NAME),
          value: FIELD_FUNCTION_NAME,
        },
        {
          label: 'Binary',
          onclick: () => toggleGroupBy(FIELD_MAPPING_FILE),
          value: FIELD_MAPPING_FILE,
        },
        {
          label: 'Code',
          onclick: () => toggleGroupBy(FIELD_FUNCTION_FILE_NAME),
          value: FIELD_FUNCTION_FILE_NAME,
        },
        {
          label: 'Address',
          onclick: () => toggleGroupBy(FIELD_LOCATION_ADDRESS),
          value: FIELD_LOCATION_ADDRESS,
        },
      ],
      hide: !!isTableVizOnly,
      icon: 'heroicons-solid:bars-3',
    },
    {
      label: 'Color by',
      id: 'h-color-by-filter',
      items: [
        {
          label: 'Binary',
          onclick: () => setColorBy('binary'),
          value: 'binary',
        },
        {
          label: 'Filename',
          onclick: () => setColorBy('filename'),
          value: 'filename',
        },
      ],
      hide: false,
      icon: 'carbon:color-palette',
    },
    {
      label: isColorStackLegendEnabled ? 'Hide legend' : 'Show legend',
      onclick: () => setColorStackLegend(isColorStackLegendEnabled ? 'false' : 'true'),
      hide: compareMode || colorProfileName === 'default',
      id: 'h-show-legend-button',
      icon: isColorStackLegendEnabled ? 'ph:eye-closed' : 'ph:eye',
    },
    {
      label: isLeftAligned ? 'Right-align function names' : 'Left-align function names',
      onclick: () => setAlignFunctionName(isLeftAligned ? 'right' : 'left'),
      id: 'h-align-function-names',
      hide: !!isTableVizOnly,
      icon: isLeftAligned
        ? 'ic:outline-align-horizontal-right'
        : 'ic:outline-align-horizontal-left',
    },
    {
      label: isCompareAbsolute ? 'Compare Relative' : 'Compare Absolute',
      onclick: () => setCompareAbsolute(isCompareAbsolute ? 'false' : 'true'),
      hide: !compareMode,
      icon: isCompareAbsolute ? 'fluent-mdl2:compare' : 'fluent-mdl2:compare-uneven',
    },
    {
      label: 'Dock Graph MetaInfo',
      hide: !!isTableVizOnly,
      customSubmenu: (
        <SwitchMenuItem
          label="Dock graph tooltip"
          id="h-dock-graph-meta-info"
          userPreferenceDetails={USER_PREFERENCES.GRAPH_METAINFO_DOCKED}
        />
      ),
      renderAsDiv: true,
    },
    {
      label: 'Highlight similar stacks when hovering over a node',
      hide: !!isTableVizOnly,
      customSubmenu: (
        <SwitchMenuItem
          label="Highlight similar stacks when hovering over a node"
          id="h-highlight-similar-stacks"
          userPreferenceDetails={USER_PREFERENCES.HIGHLIGHT_SIMILAR_STACKS}
        />
      ),
      renderAsDiv: true,
    },
    {
      label: 'Reset Legend',
      hide: hiddenBinaries === undefined || hiddenBinaries.length === 0,
      onclick: () => resetLegend(),
      id: 'h-reset-legend-button',
      icon: 'system-uicons:reset',
    },
    {
      label: 'Hidden Binaries',
      id: 'h-hidden-binaries',
      items: (hiddenBinaries as string[])?.map((binary, index) => ({
        label: binary,
        customSubmenu: (
          <div className="flex items-center gap-2 w-full">
            <input
              id={binary}
              name={binary}
              type="checkbox"
              className="h-4 w-4 rounded-md border-2 border-gray-300 text-indigo-600 focus:ring-indigo-600 focus:ring-offset-0 checked:bg-indigo-600 checked:border-indigo-600"
              checked={hiddenBinaries?.includes(binary)}
              onChange={() => handleBinaryToggle(index)}
            />
            <span>{binary}</span>
          </div>
        ),
      })),
      hide: hiddenBinaries === undefined || hiddenBinaries.length === 0,
      icon: 'ph:eye-closed',
    },
  ];

  return (
    <div
      className="relative inline-block text-left"
      id="h-visualisation-toolbar-actions"
      ref={dropdownRef}
    >
      <Menu>
        {({open, close}) => (
          <>
            <Menu.Button className="flex dark:bg-gray-900 dark:border-gray-600 justify-center w-full px-4 py-2 text-sm font-normal text-gray-600 dark:text-gray-200 bg-white rounded-md focus:outline-none focus-visible:ring-2 focus-visible:ring-white focus-visible:ring-opacity-75 border border-gray-200 pr-[1.7rem]">
              <div className="flex items-center gap-2">
                <Icon icon="pajamas:preferences" className="w-4 h-4" />

                <span>Preferences</span>
              </div>

              <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2 text-gray-400">
                <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
              </span>
            </Menu.Button>
            {open && (
              <Menu.Items
                className={cx(
                  isTableVizOnly ? 'w-64' : 'w-80',
                  'absolute z-30 mt-2 py-2 bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none border dark:bg-gray-900 dark:border-gray-600',
                  shouldOpenLeft ? 'right-0 origin-top-right' : 'left-0 origin-top-left'
                )}
              >
                {menuItems
                  .filter(item => item.hide !== undefined && !item.hide)
                  .map((item, index) => (
                    <MenuItem
                      key={index}
                      {...item}
                      onSelect={onSelect}
                      closeDropdown={close}
                      activeValueForSortBy={storeSortBy as string}
                      activeValueForColorBy={
                        colorBy === undefined || colorBy === '' ? 'binary' : colorBy
                      }
                      activeValuesForLevel={groupBy}
                      renderAsDiv={item.renderAsDiv}
                    />
                  ))}
              </Menu.Items>
            )}
          </>
        )}
      </Menu>
    </div>
  );
};

export default MultiLevelDropdown;
