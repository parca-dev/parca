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

import {Table, tableFromIPC} from '@uwdata/flechette';

import {FlamegraphArrow} from '@parca/client';

import useMappingList, {
  useFilenamesList,
} from '../../ProfileFlameGraph/FlameGraphArrow/useMappingList';
import {alignedUint8Array} from '../../utils';

interface UseProfileMetadataProps {
  flamegraphArrow?: FlamegraphArrow;
  metadataMappingFiles?: string[];
  metadataLoading: boolean;
  colorBy: string;
}

export const useProfileMetadata = ({
  flamegraphArrow,
  metadataMappingFiles,
  metadataLoading,
  colorBy,
}: UseProfileMetadataProps): {
  table: Table | null;
  mappingsList: string[];
  filenamesList: string[];
  colorMappings: string[];
  metadataLoading: boolean;
} => {
  const table: Table | null = useMemo(() => {
    if (flamegraphArrow === undefined) {
      return null;
    }
    return tableFromIPC(alignedUint8Array(flamegraphArrow.record), {useBigInt: true});
  }, [flamegraphArrow]);

  const mappingsList = useMappingList(metadataMappingFiles);
  const filenamesList = useFilenamesList(table);

  const colorMappings = colorBy === 'binary' || colorBy === '' ? mappingsList : filenamesList;

  return {
    table,
    mappingsList,
    filenamesList,
    colorMappings,
    metadataLoading,
  };
};
