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

import React, {useEffect, useRef, useState} from 'react';

import {Transition} from '@headlessui/react';
import {Icon} from '@iconify/react';
import Select from 'react-select';

import {Button} from '@parca/components';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../../ProfileIcicleGraph/IcicleGraphArrow';

interface LabelSelectorProps {
  labels: string[];
  groupBy: string[];
  setGroupByLabels: (labels: string[]) => void;
  isOpen: boolean;
  labelsButtonRef: React.RefObject<HTMLDivElement>;
  setIsLabelSelectorOpen: (isOpen: boolean) => void;
}

interface LabelSelectorProps {
  labels: string[];
  groupBy: string[];
  setGroupByLabels: (labels: string[]) => void;
}

interface LabelOption {
  label: string;
  value: string;
}

interface GroupByDropdownProps {
  groupBy: string[];
  toggleGroupBy: (key: string) => void;
  onLabelClick: () => void;
  labelsButtonRef: React.RefObject<HTMLDivElement>;
}

const groupByOptions = [
  {
    value: FIELD_FUNCTION_NAME,
    label: 'Function Name',
    description: 'Stacktraces are grouped by function names.',
    disabled: true,
  },
  {
    value: FIELD_FUNCTION_FILE_NAME,
    label: 'Filename',
    description: 'Stacktraces are grouped by filenames.',
    disabled: false,
  },
  {
    value: FIELD_LOCATION_ADDRESS,
    label: 'Address',
    description: 'Stacktraces are grouped by addresses.',
    disabled: false,
  },
  {
    value: FIELD_MAPPING_FILE,
    label: 'Binary',
    description: 'Stacktraces are grouped by binaries.',
    disabled: false,
  },
];

const LabelSelector: React.FC<LabelSelectorProps> = ({
  labels,
  groupBy,
  setGroupByLabels,
  isOpen,
  labelsButtonRef,
  setIsLabelSelectorOpen,
}) => {
  const [position, setPosition] = useState({top: 0, left: 0});

  useEffect(() => {
    if (isOpen && labelsButtonRef.current !== null) {
      const rect = labelsButtonRef.current.getBoundingClientRect();
      const parentRect = labelsButtonRef.current.offsetParent?.getBoundingClientRect() ?? {
        top: 0,
        left: 0,
      };

      setPosition({
        top: rect.bottom - parentRect.top,
        left: rect.right - parentRect.left + 4,
      });
    }
  }, [isOpen, labelsButtonRef]);

  if (!isOpen) return null;

  return (
    <div
      className="absolute w-64 ml-4 z-20"
      style={{
        top: `${position.top}px`,
        left: `${position.left}px`,
      }}
    >
      <Select<LabelOption, true>
        isMulti
        name="labels"
        options={labels.map(label => ({label, value: `${FIELD_LABELS}.${label}`}))}
        className="parca-select-container text-sm w-full border-gray-300 border rounded-md"
        classNamePrefix="parca-select"
        value={groupBy
          .filter(l => l.startsWith(FIELD_LABELS))
          .map(l => ({value: l, label: l.slice(FIELD_LABELS.length + 1)}))}
        onChange={newValue => {
          setGroupByLabels(newValue.map(option => option.value));
          setIsLabelSelectorOpen(false);
        }}
        placeholder="Select labels..."
        styles={{
          menu: provided => ({
            ...provided,
            position: 'relative',
            marginBottom: 0,
            boxShadow: 'none',
            marginTop: 0,
          }),
          control: provided => ({
            ...provided,
            boxShadow: 'none',
            borderBottom: '1px solid #e2e8f0',
            borderRight: 0,
            borderLeft: 0,
            borderTop: 0,
            borderBottomLeftRadius: 0,
            borderBottomRightRadius: 0,
            ':hover': {
              borderColor: '#e2e8f0',
              borderBottomLeftRadius: 0,
              borderBottomRightRadius: 0,
            },
          }),
        }}
        menuIsOpen={true}
      />
    </div>
  );
};

const GroupByDropdown: React.FC<GroupByDropdownProps> = ({
  groupBy,
  toggleGroupBy,
  onLabelClick,
  labelsButtonRef,
}) => {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent): void => {
      if (
        isDropdownOpen &&
        dropdownRef.current != null &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsDropdownOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isDropdownOpen]);

  const label =
    groupBy.length === 0
      ? 'Nothing'
      : groupBy.length === 1
      ? groupByOptions.find(option => option.value === groupBy[0])?.label
      : 'Multiple';

  const selectedLabels = groupBy
    .filter(l => l.startsWith(FIELD_LABELS))
    .map(l => l.slice(FIELD_LABELS.length + 1));

  return (
    <div className="relative" ref={dropdownRef}>
      <label className="text-sm">Group by</label>
      <div className="relative text-left" id="h-group-by-filter">
        <Button
          variant="neutral"
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
          className="relative w-max cursor-default rounded-md border bg-white py-2 pl-3 pr-[1.7rem] text-left text-sm shadow-sm dark:border-gray-600 dark:bg-gray-900 sm:text-sm"
        >
          <span className="block overflow-x-hidden text-ellipsis">{label}</span>
          <span className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2 text-gray-400">
            <Icon icon="heroicons:chevron-down-20-solid" aria-hidden="true" />
          </span>
        </Button>

        <Transition
          as="div"
          leave="transition ease-in duration-100"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
          show={isDropdownOpen}
        >
          <div className="absolute left-0 z-10 mt-1 min-w-[400px] overflow-auto rounded-md bg-gray-50 py-1 text-base shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none dark:border-gray-600 dark:bg-gray-900 dark:ring-white dark:ring-opacity-20 sm:text-sm">
            <div className="p-4">
              <fieldset>
                <div className="space-y-5">
                  {groupByOptions.map(({value, label, description, disabled}) => (
                    <div key={value} className="relative flex items-start">
                      <div className="flex h-6 items-center">
                        <input
                          id={value}
                          name={value}
                          type="checkbox"
                          disabled={disabled}
                          className="h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600"
                          checked={groupBy.includes(value)}
                          onChange={() => toggleGroupBy(value)}
                        />
                      </div>
                      <div className="ml-3 text-sm leading-6">
                        <label
                          htmlFor={value}
                          className="font-medium text-gray-900 dark:text-gray-200"
                        >
                          {label}
                        </label>
                        <p className="text-gray-500 dark:text-gray-400">{description}</p>
                      </div>
                    </div>
                  ))}
                  <div
                    className="ml-7 flex flex-col items-start text-sm leading-6 cursor-pointer"
                    onClick={onLabelClick}
                    ref={labelsButtonRef}
                  >
                    <div className="flex justify-between w-full items-center">
                      <div>
                        <span className="font-medium text-gray-900 dark:text-gray-200">Labels</span>
                        <p className="text-gray-500 dark:text-gray-400">
                          Stacktraces are grouped by labels.
                        </p>
                      </div>

                      <Icon icon="flowbite:caret-right-solid" className="h-[14px] w-[14px]" />
                    </div>

                    {selectedLabels.length > 0 && (
                      <div className="flex gap-2 flex-wrap">
                        <span className="text-gray-500 dark:text-gray-200">Selected labels:</span>

                        <div className="flex flex-wrap gap-3">
                          {selectedLabels.map(label => (
                            <span
                              key={label}
                              className="mr-2 px-3 py-1 text-xs text-gray-700 dark:text-gray-200 bg-gray-200 rounded-md dark:bg-gray-800"
                            >
                              {label}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              </fieldset>
            </div>
          </div>
        </Transition>
      </div>
    </div>
  );
};

interface GroupByControlsProps {
  groupBy: string[];
  labels: string[];
  toggleGroupBy: (key: string) => void;
  setGroupByLabels: (labels: string[]) => void;
}

const GroupByControls: React.FC<GroupByControlsProps> = ({
  groupBy,
  labels,
  toggleGroupBy,
  setGroupByLabels,
}) => {
  const [isLabelSelectorOpen, setIsLabelSelectorOpen] = useState(false);

  const labelsButton = useRef<HTMLDivElement>(null);
  const labelSelectorRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent): void => {
      if (
        isLabelSelectorOpen &&
        labelSelectorRef.current !== null &&
        !labelSelectorRef.current.contains(event.target as Node) &&
        labelsButton.current !== null &&
        !labelsButton.current.contains(event.target as Node)
      ) {
        setIsLabelSelectorOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isLabelSelectorOpen]);

  return (
    <div className="inline-flex items-start">
      <div className="relative flex items-start">
        <GroupByDropdown
          groupBy={groupBy}
          toggleGroupBy={toggleGroupBy}
          onLabelClick={() => setIsLabelSelectorOpen(!isLabelSelectorOpen)}
          labelsButtonRef={labelsButton}
        />
        <div ref={labelSelectorRef}>
          <LabelSelector
            labels={labels}
            groupBy={groupBy}
            setGroupByLabels={setGroupByLabels}
            isOpen={isLabelSelectorOpen}
            labelsButtonRef={labelsButton}
            setIsLabelSelectorOpen={setIsLabelSelectorOpen}
          />
        </div>
      </div>
    </div>
  );
};

export default GroupByControls;
