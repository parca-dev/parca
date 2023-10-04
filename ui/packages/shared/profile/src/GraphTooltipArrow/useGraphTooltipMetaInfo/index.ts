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

import {Table} from 'apache-arrow';

import {QueryRequest_ReportType} from '@parca/client';
import {useParcaContext, useURLState} from '@parca/components';
import type {NavigateFunction} from '@parca/utilities';

import {
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_START_LINE,
  FIELD_INLINED,
  FIELD_LOCATION_ADDRESS,
  FIELD_LOCATION_LINE,
  FIELD_MAPPING_BUILD_ID,
  FIELD_MAPPING_FILE,
} from '../../ProfileIcicleGraph/IcicleGraphArrow';
import {arrowToString} from '../../ProfileIcicleGraph/IcicleGraphArrow/utils';
import {ProfileSource} from '../../ProfileSource';
import {useProfileViewContext} from '../../ProfileView/ProfileViewContext';
import {useQuery} from '../../useQuery';

interface Props {
  table: Table<any>;
  row: number;
  navigateTo: NavigateFunction;
}

interface GraphTooltipMetaInfoData {
  labelPairs: Array<[string, string]>;
  functionFilename: string;
  file: string;
  openFile: () => void;
  isSourceAvailable: boolean;
  locationAddress: bigint;
  mappingFile: string | null;
  mappingBuildID: string | null;
  inlined: boolean | null;
}

export const useGraphTooltipMetaInfo = ({
  table,
  row,
  navigateTo,
}: Props): GraphTooltipMetaInfoData => {
  const mappingFile: string | null = arrowToString(table.getChild(FIELD_MAPPING_FILE)?.get(row));
  const mappingBuildID: string | null = arrowToString(
    table.getChild(FIELD_MAPPING_BUILD_ID)?.get(row)
  );
  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;
  const inlined: boolean | null = table.getChild(FIELD_INLINED)?.get(row);
  const locationLine: bigint = table.getChild(FIELD_LOCATION_LINE)?.get(row) ?? 0n;
  const functionFilename: string =
    arrowToString(table.getChild(FIELD_FUNCTION_FILE_NAME)?.get(row)) ?? '';
  const functionStartLine: bigint = table.getChild(FIELD_FUNCTION_START_LINE)?.get(row) ?? 0n;
  const lineNumber =
    locationLine !== 0n ? locationLine : functionStartLine !== 0n ? functionStartLine : undefined;
  const pprofLabelPrefix = 'pprof_labels.';
  const labelColumnNames = table.schema.fields.filter(field =>
    field.name.startsWith(pprofLabelPrefix)
  );

  const {queryServiceClient, enableSourcesView} = useParcaContext();
  const {profileSource} = useProfileViewContext();

  const {isLoading: sourceLoading, response: sourceResponse} = useQuery(
    queryServiceClient,
    profileSource as ProfileSource,
    QueryRequest_ReportType.SOURCE,
    {
      skip:
        enableSourcesView === false ||
        profileSource === undefined ||
        // eslint-disable-next-line no-extra-boolean-cast
        !Boolean(mappingBuildID) ||
        // eslint-disable-next-line no-extra-boolean-cast
        !Boolean(functionFilename),
      sourceBuildID: mappingBuildID !== null ? mappingBuildID : undefined,
      sourceFilename: functionFilename,
      sourceOnly: true,
    }
  );

  const isSourceAvailable = !sourceLoading && sourceResponse?.report != null;

  const getTextForFile = (): string => {
    if (functionFilename === '') return '<unknown>';

    return `${functionFilename} ${lineNumber !== undefined ? ` +${lineNumber.toString()}` : ''}`;
  };
  const file = getTextForFile();

  const labelPairs: Array<[string, string]> = labelColumnNames
    .map((field, i) => [
      labelColumnNames[i].name.slice(pprofLabelPrefix.length),
      arrowToString(table.getChild(field.name)?.get(row)) ?? '',
    ])
    .filter(value => value[1] !== '') as Array<[string, string]>;

  const [dashboardItems, setDashboardItems] = useURLState({
    param: 'dashboard_items',
    navigateTo,
  });

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [unusedBuildId, setSourceBuildId] = useURLState({
    param: 'source_buildid',
    navigateTo,
  });

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [unusedFilename, setSourceFilename] = useURLState({
    param: 'source_filename',
    navigateTo,
  });

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const [unusedLine, setSourceLine] = useURLState({
    param: 'source_line',
    navigateTo,
  });

  const openFile = (): void => {
    setDashboardItems([dashboardItems[0], 'source']);
    if (mappingBuildID != null) {
      setSourceBuildId(mappingBuildID);
    }

    setSourceFilename(functionFilename);
    if (lineNumber !== undefined) {
      setSourceLine(lineNumber.toString());
    }
  };

  return {
    labelPairs,
    functionFilename,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingBuildID,
    mappingFile,
    inlined,
  };
};
