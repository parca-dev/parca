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

import {createParser, useQueryState} from 'nuqs';

interface SelectedLineRange {
  start: number;
  end: number;
}

const lineRangeParser = createParser<SelectedLineRange>({
  parse: (value: string) => {
    const [start, end] = value.split('-');
    const startNum = parseInt(start, 10);
    if (isNaN(startNum)) return null;
    const endNum = end !== undefined ? parseInt(end, 10) : startNum;
    return {start: startNum, end: isNaN(endNum) ? startNum : endNum};
  },
  serialize: (value: SelectedLineRange) => `${value.start}-${value.end}`,
}).withOptions({history: 'replace'});

interface LineRange {
  startLine: number;
  endLine: number;
  setLineRange: (start: number, end: number) => void;
}

const useLineRange = (): LineRange => {
  const [lineRange, setRawLineRange] = useQueryState(
    'source_line',
    lineRangeParser.withDefault({start: -1, end: -1})
  );

  const setLineRange = useCallback(
    (start: number, end: number): void => {
      void setRawLineRange({start, end});
    },
    [setRawLineRange]
  );

  return {
    startLine: lineRange.start,
    endLine: lineRange.end,
    setLineRange,
  };
};

export default useLineRange;
