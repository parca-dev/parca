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

import {Icon} from '@iconify/react';
import {Table} from 'apache-arrow';

import {useParcaContext} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {formatDateTimeDownToMS, getLastItem} from '@parca/utilities';

import {hexifyAddress, truncateString, truncateStringReverse} from '../utils';
import {ExpandOnHover} from './ExpandOnHoverValue';
import {useGraphTooltip} from './useGraphTooltip';
import {useGraphTooltipMetaInfo} from './useGraphTooltipMetaInfo';

interface GraphTooltipArrowContentProps {
  table: Table<any>;
  profileType?: ProfileType;
  unit?: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  isFixed: boolean;
  compareAbsolute: boolean;
}

const NoData = (): React.JSX.Element => {
  return <span className="rounded bg-gray-200 px-2 dark:bg-gray-800">Not available</span>;
};

const GraphTooltipArrowContent = ({
  table,
  profileType,
  unit,
  total,
  totalUnfiltered,
  row,
  isFixed,
  compareAbsolute,
}: GraphTooltipArrowContentProps): React.JSX.Element => {
  const graphTooltipData = useGraphTooltip({
    table,
    profileType,
    unit,
    total,
    totalUnfiltered,
    row,
    compareAbsolute,
  });

  if (graphTooltipData === null) {
    return <></>;
  }

  const {
    name,
    locationAddress,
    cumulativeText,
    flatText,
    diffText,
    diff,
    row: rowNumber,
  } = graphTooltipData;

  return (
    <div className={`flex text-sm ${isFixed ? 'w-full' : ''}`}>
      <div className={`m-auto w-full ${isFixed ? 'w-full' : ''}`}>
        <div className="min-h-52 flex w-[500px] flex-col justify-between rounded-lg border border-gray-300 bg-gray-50 p-3 shadow-lg dark:border-gray-500 dark:bg-gray-900">
          <div className="flex flex-row">
            <div className="mx-2">
              <div className="flex h-10 items-start justify-between gap-4 break-all font-semibold">
                {row === 0 ? (
                  <p>root</p>
                ) : (
                  <p>
                    {name !== ''
                      ? name
                      : locationAddress !== 0n
                      ? hexifyAddress(locationAddress)
                      : 'unknown'}
                  </p>
                )}
              </div>
              <table className="my-2 w-full table-fixed pr-0 text-gray-700 dark:text-gray-300">
                <tbody>
                  <tr>
                    <td className="w-1/4">Cumulative</td>
                    <td className="w-3/4">
                      <div>{cumulativeText}</div>
                    </td>
                  </tr>
                  <tr>
                    <td className="w-1/4 pt-2">Flat</td>
                    <td className="w-3/4 pt-2">
                      <div>{flatText}</div>
                    </td>
                  </tr>
                  {diff !== 0n && (
                    <tr>
                      <td className="w-1/4 pt-2">Diff</td>
                      <td className="w-3/4 pt-2">
                        <div>{diffText}</div>
                      </td>
                    </tr>
                  )}
                  <TooltipMetaInfo table={table} row={rowNumber} />
                </tbody>
              </table>
            </div>
          </div>
          <div className="flex w-full items-center gap-1 text-xs text-gray-500">
            <Icon icon="iconoir:mouse-button-right" />
            <div>Right click to show context menu</div>
          </div>
        </div>
      </div>
    </div>
  );
};

const TooltipMetaInfo = ({table, row}: {table: Table<any>; row: number}): React.JSX.Element => {
  const {
    labelPairs,
    functionFilename,
    file,
    locationAddress,
    mappingFile,
    mappingBuildID,
    inlined,
    timestamp,
  } = useGraphTooltipMetaInfo({table, row});
  const {timezone} = useParcaContext();

  const labels = labelPairs.map(
    (l): React.JSX.Element => (
      <span
        key={l[0]}
        className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
      >
        {`${l[0]}="${l[1]}"`}
      </span>
    )
  );

  const isMappingBuildIDAvailable = mappingBuildID !== null && mappingBuildID !== '';
  const inlinedText = inlined === null ? 'merged' : inlined ? 'yes' : 'no';

  return (
    <>
      {timestamp == null || timestamp === 0n ? (
        <div className="pt-2" />
      ) : (
        <tr>
          <td className="w-1/4 pt-2">Timestamp</td>
          <td className="w-3/4 pt-2 break-all">
            {formatDateTimeDownToMS(new Date(Number(timestamp / 1000000n)), timezone)}
          </td>{' '}
        </tr>
      )}
      <tr>
        <td className="w-1/4">File</td>
        <td className="w-3/4 break-all">
          {functionFilename === '' ? (
            <NoData />
          ) : (
            <div className="flex gap-4">
              <div className="whitespace-nowrap text-left">
                <ExpandOnHover value={file} displayValue={truncateStringReverse(file, 30)} />
              </div>
            </div>
          )}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Address</td>
        <td className="w-3/4 break-all">
          {locationAddress === 0n ? <NoData /> : <div>{hexifyAddress(locationAddress)}</div>}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Inlined</td>
        <td className="w-3/4 break-all">{inlinedText}</td>
      </tr>
      <tr>
        <td className="w-1/4">Binary</td>
        <td className="w-3/4 break-all">
          {(mappingFile != null ? getLastItem(mappingFile) : null) ?? <NoData />}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Build Id</td>
        <td className="w-3/4 break-all">
          {isMappingBuildIDAvailable ? <div>{truncateString(mappingBuildID, 28)}</div> : <NoData />}
        </td>
      </tr>
      {labelPairs.length > 0 && (
        <tr>
          <td className="w-1/4">Labels</td>
          <td className="w-3/4 break-all">{labels}</td>
        </tr>
      )}
    </>
  );
};

export default GraphTooltipArrowContent;
