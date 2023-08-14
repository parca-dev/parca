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

import React, {useState} from 'react';

import {Table} from 'apache-arrow';
import {CopyToClipboard} from 'react-copy-to-clipboard';

import {divide, getLastItem, valueFormatter} from '@parca/utilities';

import {
  FIELD_CUMULATIVE,
  FIELD_DIFF,
  FIELD_FUNCTION_FILE_NAME,
  FIELD_FUNCTION_START_LINE,
  FIELD_LABELS,
  FIELD_LOCATION_ADDRESS,
  FIELD_LOCATION_LINE,
  FIELD_MAPPING_BUILD_ID,
  FIELD_MAPPING_FILE,
} from '../ProfileIcicleGraph/IcicleGraphArrow';
import {nodeLabel} from '../ProfileIcicleGraph/IcicleGraphArrow/utils';
import {hexifyAddress, truncateString, truncateStringReverse} from '../utils';
import {ExpandOnHover} from './ExpandOnHoverValue';

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

interface GraphTooltipArrowContentProps {
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
  isFixed: boolean;
}

const NoData = (): React.JSX.Element => {
  return <span className="rounded bg-gray-200 px-2 dark:bg-gray-800">Not available</span>;
};

const GraphTooltipArrowContent = ({
  table,
  unit,
  total,
  totalUnfiltered,
  row,
  level,
  isFixed,
}: GraphTooltipArrowContentProps): React.JSX.Element => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  if (row === null) {
    return <></>;
  }

  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;
  const cumulative: bigint = table.getChild(FIELD_CUMULATIVE)?.get(row) ?? 0n;
  const diff: bigint = table.getChild(FIELD_DIFF)?.get(row) ?? 0n;

  const onCopy = (): void => {
    setIsCopied(true);

    if (timeoutHandle !== null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  const prevValue = cumulative - diff;
  const diffRatio = diff !== 0n ? divide(diff, prevValue) : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;

  const name = nodeLabel(table, row, level, false);
  console.log(level, row, name);

  const getTextForCumulative = (hoveringNodeCumulative: bigint): string => {
    const filtered =
      totalUnfiltered > total
        ? ` / ${(100 * divide(hoveringNodeCumulative, total)).toFixed(2)}% of filtered`
        : '';
    return `${valueFormatter(hoveringNodeCumulative, unit, 2)}
    (${(100 * divide(hoveringNodeCumulative, totalUnfiltered)).toFixed(2)}%${filtered})`;
  };

  return (
    <div className={`flex text-sm ${isFixed ? 'w-full' : ''}`}>
      <div className={`m-auto w-full ${isFixed ? 'w-full' : ''}`}>
        <div className="min-h-52 flex w-[500px] flex-col justify-between rounded-lg border border-gray-300 bg-gray-50 p-3 shadow-lg dark:border-gray-500 dark:bg-gray-900">
          <div className="flex flex-row">
            <div className="mx-2">
              <div className="flex h-10 items-center break-all font-semibold">
                {row === 0 ? (
                  <p>root</p>
                ) : (
                  <>
                    {name !== '' ? (
                      <CopyToClipboard onCopy={onCopy} text={name}>
                        <button className="cursor-pointer text-left">{name}</button>
                      </CopyToClipboard>
                    ) : (
                      <>
                        {locationAddress !== 0n ? (
                          <CopyToClipboard onCopy={onCopy} text={hexifyAddress(locationAddress)}>
                            <button className="cursor-pointer text-left">
                              {hexifyAddress(locationAddress)}
                            </button>
                          </CopyToClipboard>
                        ) : (
                          <p>unknown</p>
                        )}
                      </>
                    )}
                  </>
                )}
              </div>
              <table className="my-2 w-full table-fixed pr-0 text-gray-700 dark:text-gray-300">
                <tbody>
                  <tr>
                    <td className="w-1/4">Cumulative</td>

                    <td className="w-3/4">
                      <CopyToClipboard onCopy={onCopy} text={getTextForCumulative(cumulative)}>
                        <button className="cursor-pointer">
                          {getTextForCumulative(cumulative)}
                        </button>
                      </CopyToClipboard>
                    </td>
                  </tr>
                  {diff !== 0n && (
                    <tr>
                      <td className="w-1/4">Diff</td>
                      <td className="w-3/4">
                        <CopyToClipboard onCopy={onCopy} text={diffText}>
                          <button className="cursor-pointer">{diffText}</button>
                        </CopyToClipboard>
                      </td>
                    </tr>
                  )}
                  <TooltipMetaInfo table={table} row={row} onCopy={onCopy} />
                </tbody>
              </table>
            </div>
          </div>
          <span className="mx-2 block text-xs text-gray-500">
            {isCopied ? 'Copied!' : 'Hold shift and click on a value to copy.'}
          </span>
        </div>
      </div>
    </div>
  );
};

const TooltipMetaInfo = ({
  table,
  // total,
  // totalUnfiltered,
  onCopy,
  row,
}: {
  table: Table<any>;
  row: number;
  onCopy: () => void;
}): React.JSX.Element => {
  const mappingFile: string = table.getChild(FIELD_MAPPING_FILE)?.get(row) ?? '';
  const mappingBuildID: string = table.getChild(FIELD_MAPPING_BUILD_ID)?.get(row) ?? '';
  const locationAddress: bigint = table.getChild(FIELD_LOCATION_ADDRESS)?.get(row) ?? 0n;
  const locationLine: bigint = table.getChild(FIELD_LOCATION_LINE)?.get(row) ?? 0n;
  const functionFilename: string = table.getChild(FIELD_FUNCTION_FILE_NAME)?.get(row) ?? '';
  const functionStartLine: bigint = table.getChild(FIELD_FUNCTION_START_LINE)?.get(row) ?? 0n;
  const labelsString: string = table.getChild(FIELD_LABELS)?.get(row) ?? '{}';

  const getTextForFile = (): string => {
    if (functionFilename === '') return '<unknown>';

    return `${functionFilename} ${
      locationLine !== 0n
        ? ` +${locationLine.toString()}`
        : `${functionStartLine !== 0n ? ` +${functionStartLine}` : ''}`
    }`;
  };
  const file = getTextForFile();

  const labels = Object.entries(JSON.parse(labelsString))
    .sort((a, b) => {
      return a[0].localeCompare(b[0]);
    })
    .map(
      (l): React.JSX.Element => (
        <span
          key={l[0]}
          className="mr-3 inline-block rounded-lg bg-gray-200 px-2 py-1 text-xs font-bold text-gray-700 dark:bg-gray-700 dark:text-gray-400"
        >
          {`${l[0]}="${l[1] as string}"`}
        </span>
      )
    );

  return (
    <>
      <tr>
        <td className="w-1/4">File</td>
        <td className="w-3/4 break-all">
          {functionFilename === '' ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={file}>
              <button className="cursor-pointer whitespace-nowrap text-left">
                <ExpandOnHover value={file} displayValue={truncateStringReverse(file, 40)} />
              </button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Address</td>
        <td className="w-3/4 break-all">
          {locationAddress === 0n ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={hexifyAddress(locationAddress)}>
              <button className="cursor-pointer">{hexifyAddress(locationAddress)}</button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Binary</td>
        <td className="w-3/4 break-all">
          {mappingFile === '' ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={mappingFile}>
              <button className="cursor-pointer">{getLastItem(mappingFile)}</button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Build Id</td>
        <td className="w-3/4 break-all">
          {mappingBuildID === '' ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={mappingBuildID}>
              <button className="cursor-pointer">
                {truncateString(getLastItem(mappingBuildID) as string, 28)}
              </button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
      {labelsString !== '{}' && (
        <tr>
          <td className="w-1/4">Labels</td>
          <td className="w-3/4 break-all">{labels}</td>
        </tr>
      )}
    </>
  );
};

export default GraphTooltipArrowContent;
