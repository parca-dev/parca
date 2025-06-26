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

import {Dictionary, Table, Vector} from 'apache-arrow';

import {getLastItem} from '@parca/utilities';

import {FIELD_FUNCTION_FILE_NAME} from './index';
import {arrowToString} from './utils';

const useMappingList = (mappings: string[] | undefined): string[] => {
  const mappingsList = useMemo(() => {
    if (mappings === undefined) {
      return [];
    }
    const list =
      mappings
        ?.map(mapping => {
          return getLastItem(mapping) as string;
        })
        .flat() ?? [];

    // We add a EVERYTHING ELSE mapping to the list.
    list.push('');

    // We sort the mappings alphabetically to make sure that the order is always the same.
    list.sort((a, b) => a.localeCompare(b));

    return list;
  }, [mappings]);

  return mappingsList;
};

export const useFilenamesList = (table: Table | null): string[] => {
  if (table === null) {
    return [];
  }
  const filenamesDict: Vector<Dictionary> | null = table.getChild(FIELD_FUNCTION_FILE_NAME);
  const filenames =
    filenamesDict?.data
      .map(file => {
        if (file.dictionary == null) {
          return [];
        }
        const len = file.dictionary.length;
        const entries: string[] = [];
        for (let i = 0; i < len; i++) {
          const fn = arrowToString(file.dictionary.get(i));
          entries.push(getLastItem(fn) ?? '');
        }
        return entries;
      })
      .flat() ?? [];

  filenames.push('');

  filenames.sort((a, b) => a.localeCompare(b));
  return filenames;
};

export default useMappingList;
