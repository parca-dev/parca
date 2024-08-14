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

import React, {useEffect, useState} from 'react';

import CustomSelect, {SelectItem} from '../SimpleMatchers/Select';

interface Props {
  labelNames: string[];
  onSelectionChange: (selections: Record<string, string>) => void;
  labelValues: string[];
}
const ViewMatchers: React.FC<Props> = ({labelNames, onSelectionChange, labelValues}) => {
  const [selections, setSelections] = useState<Record<string, string>>({});

  useEffect(() => {
    onSelectionChange(selections);
  }, [selections, onSelectionChange]);

  const handleSelection = (labelName: string, value: string): void => {
    setSelections(prev => ({
      ...prev,
      [labelName]: value,
    }));
  };

  const transformValuesForSelect = (values: string[]): SelectItem[] => {
    return values.map(value => ({
      key: value,
      element: {active: <>{value}</>, expanded: <>{value}</>},
    }));
  };

  return (
    <div className="flex flex-wrap gap-2">
      {labelNames.map(labelName => (
        <div key={labelName} className="flex items-center">
          <div className="bg-gray-100 dark:bg-gray-700 px-3 py-2 rounded-l-md border border-r-0 border-gray-300 dark:border-gray-600">
            {labelName}
          </div>
          <CustomSelect
            placeholder="Select value"
            items={transformValuesForSelect(labelValues)}
            onSelection={value => handleSelection(labelName, value)}
            selectedKey={selections[labelName]}
            className="rounded-l-none border-l-0"
          />
        </div>
      ))}
    </div>
  );
};

export default ViewMatchers;
