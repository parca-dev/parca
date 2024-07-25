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

import {useMemo} from 'react';

import {useURLState} from '@parca/components';

interface LineRange {
  startLine: number;
  endLine: number;
  setLineRange: (start: number, end: number) => void;
}

const useLineRange = (): LineRange => {
  const [sourceLine, setSourceLine] = useURLState<string | undefined>('source_line');
  const [startLine, endLine] = useMemo(() => {
    if (sourceLine == null) {
      return [-1, -1];
    }
    const [start, end] = sourceLine.split('-');

    if (end === undefined) {
      return [parseInt(start, 10), parseInt(start, 10)];
    }
    return [parseInt(start, 10), parseInt(end, 10)];
  }, [sourceLine]);

  const setLineRange = (start: number, end: number): void => {
    setSourceLine(`${start}-${end}`);
  };

  return {startLine, endLine, setLineRange};
};

export default useLineRange;
