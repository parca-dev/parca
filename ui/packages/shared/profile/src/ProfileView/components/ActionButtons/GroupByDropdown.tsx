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

import GroupByLabelsDropdown from '../GroupByLabelsDropdown';

interface GroupByControlsProps {
  groupBy: string[];
  labels: string[];
  setGroupByLabels: (labels: string[]) => void;
}

const GroupByControls: React.FC<GroupByControlsProps> = ({groupBy, labels, setGroupByLabels}) => {
  return (
    <div className="inline-flex items-start">
      <div className="relative flex gap-3 items-start">
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
