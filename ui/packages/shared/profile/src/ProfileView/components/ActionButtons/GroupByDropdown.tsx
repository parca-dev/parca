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

import React from 'react';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_NAME,
  FIELD_LOCATION_ADDRESS,
  FIELD_MAPPING_FILE,
} from '../../../ProfileIcicleGraph/IcicleGraphArrow';
import GroupByLabelsDropdown from '../GroupByLabelsDropdown';
import LevelsDropdownSelect from '../LevelsDropdownSelect';

export const groupByOptions = [
  {
    value: FIELD_FUNCTION_NAME,
    label: 'Function',
    disabled: true,
  },
  {
    value: FIELD_MAPPING_FILE,
    label: 'Binary',
    disabled: false,
  },
  {
    value: FIELD_FUNCTION_FILE_NAME,
    label: 'Code',
    disabled: false,
  },
  {
    value: FIELD_LOCATION_ADDRESS,
    label: 'Address',
    disabled: false,
  },
];

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
  return (
    <div className="inline-flex items-start">
      <div className="relative flex gap-3 items-start">
        <LevelsDropdownSelect
          groupBy={groupBy}
          toggleGroupBy={toggleGroupBy}
          groupByOptions={groupByOptions}
        />
        <GroupByLabelsDropdown
          labels={labels}
          groupBy={groupBy}
          setGroupByLabels={setGroupByLabels}
        />
      </div>
    </div>
  );
};

export default GroupByControls;
