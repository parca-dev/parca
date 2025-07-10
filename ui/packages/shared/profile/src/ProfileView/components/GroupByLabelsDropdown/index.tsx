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

import Select from 'react-select';

import {FIELD_LABELS} from '../../../ProfileFlameGraph/FlameGraphArrow';

interface LabelOption {
  label: string;
  value: string;
}

interface Props {
  labels: string[];
  groupBy: string[];
  setGroupByLabels: (labels: string[]) => void;
}

const GroupByLabelsDropdown = ({labels, groupBy, setGroupByLabels}: Props): JSX.Element => {
  return (
    <div>
      <div className="flex items-center justify-between">
        <label className="text-sm">Group by</label>
      </div>

      <Select<LabelOption, true>
        id="h-group-by-labels-selector"
        isMulti
        defaultMenuIsOpen={false}
        defaultValue={undefined}
        name="labels"
        options={labels.map(label => ({label, value: `${FIELD_LABELS}.${label}`}))}
        className="parca-select-container text-sm w-full rounded-md bg-white"
        classNamePrefix="parca-select"
        value={groupBy
          .filter(l => l.startsWith(FIELD_LABELS))
          .map(l => ({value: l, label: l.slice(FIELD_LABELS.length + 1)}))}
        onChange={newValue => {
          setGroupByLabels(newValue.map(option => option.value));
        }}
        placeholder="Select labels..."
        styles={{
          menu: provided => ({
            ...provided,
            marginBottom: 0,
            boxShadow: 'none',
            marginTop: 0,
            zIndex: 1000,
            minWidth: '320px',
          }),
          control: provided => ({
            ...provided,
            position: 'relative',
            boxShadow: 'none',
            borderBottom: '1px solid #e2e8f0',
            borderRight: '1px solid #e2e8f0',
            borderLeft: '1px solid #e2e8f0',
            borderTop: '1px solid #e2e8f0',
            ':hover': {
              borderColor: '#e2e8f0',
              borderBottomLeftRadius: 0,
              borderBottomRightRadius: 0,
            },
          }),
          option: provided => ({
            ...provided,
            ':hover': {
              backgroundColor: '#4f46e5',
              color: '#ffffff',
            },
            ':focus': {
              backgroundColor: '#4f46e5',
              color: '#ffffff',
            },
          }),
        }}
      />
    </div>
  );
};

export default GroupByLabelsDropdown;
