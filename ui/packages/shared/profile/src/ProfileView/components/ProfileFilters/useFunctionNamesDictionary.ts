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

import {Column, Table} from '@uwdata/flechette';

import {FIELD_FUNCTION_NAME} from '../../../ProfileFlameGraph/FlameGraphArrow/index';
import {arrowToString} from '../../../ProfileFlameGraph/FlameGraphArrow/utils';

const useFunctionNamesDictionary = (table: Table | null | undefined): string[] | undefined => {
  return useMemo(() => {
    if (table == null) return undefined;

    const column: Column<string> | null = table.getChild(FIELD_FUNCTION_NAME);
    if (column === null) return undefined;

    // Access dictionary directly instead of iterating all rows
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const dictionary: Column<string> | undefined = (column.data[0] as any)?.dictionary;
    if (dictionary == null) return undefined;

    return Array.from(dictionary.toArray()).map(value => arrowToString(value) ?? '');
  }, [table]);
};

export default useFunctionNamesDictionary;
