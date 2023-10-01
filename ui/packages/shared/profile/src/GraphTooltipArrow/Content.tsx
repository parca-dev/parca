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
import cx from 'classnames';
import {CopyToClipboard} from 'react-copy-to-clipboard';
import {Tooltip} from 'react-tooltip';

import {Button, IconButton, useParcaContext} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {getLastItem, type NavigateFunction} from '@parca/utilities';

import {hexifyAddress, truncateString, truncateStringReverse} from '../utils';
import {ExpandOnHover} from './ExpandOnHoverValue';
import {useGraphTooltip} from './useGraphTooltip';
import {useGraphTooltipMetaInfo} from './useGraphTooltipMetaInfo';

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

interface GraphTooltipArrowContentProps {
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
  isFixed: boolean;
  navigateTo: NavigateFunction;
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
  navigateTo,
}: GraphTooltipArrowContentProps): React.JSX.Element => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const graphTooltipData = useGraphTooltip({
    table,
    unit,
    total,
    totalUnfiltered,
    row,
    level,
  });
  const [_, setIsDocked] = useUserPreference(USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key);

  if (graphTooltipData === null) {
    return <></>;
  }

  const onCopy = (): void => {
    setIsCopied(true);

    if (timeoutHandle !== null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  const {name, locationAddress, cumulativeText, diffText, diff, row: rowNumber} = graphTooltipData;

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
                <IconButton
                  onClick={() => setIsDocked(true)}
                  icon="mdi:dock-bottom"
                  title="Dock MetaInfo Panel"
                />
              </div>
              <table className="my-2 w-full table-fixed pr-0 text-gray-700 dark:text-gray-300">
                <tbody>
                  <tr>
                    <td className="w-1/4">Cumulative</td>

                    <td className="w-3/4">
                      <CopyToClipboard onCopy={onCopy} text={cumulativeText}>
                        <button className="cursor-pointer">{cumulativeText}</button>
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
                  <TooltipMetaInfo
                    table={table}
                    row={rowNumber}
                    onCopy={onCopy}
                    navigateTo={navigateTo}
                  />
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
  navigateTo,
}: {
  table: Table<any>;
  row: number;
  onCopy: () => void;
  navigateTo: NavigateFunction;
}): React.JSX.Element => {
  const {
    labelPairs,
    functionFilename,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingFile,
    mappingBuildID,
    inlined,
  } = useGraphTooltipMetaInfo({table, row, navigateTo});
  const {enableSourcesView} = useParcaContext();

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
      <tr>
        <td className="w-1/4">File</td>
        <td className="w-3/4 break-all">
          {functionFilename === '' ? (
            <NoData />
          ) : (
            <div className="flex gap-4">
              <CopyToClipboard onCopy={onCopy} text={file}>
                <button className="cursor-pointer whitespace-nowrap text-left">
                  <ExpandOnHover value={file} displayValue={truncateStringReverse(file, 30)} />
                </button>
              </CopyToClipboard>
              <div className={cx('flex gap-2', {hidden: enableSourcesView === false})}>
                <div
                  data-tooltip-id="open-source-button-help"
                  data-tooltip-content="There is no source code uploaded for this build"
                >
                  <Button
                    variant={'neutral'}
                    onClick={() => openFile()}
                    className="shrink-0"
                    disabled={!isSourceAvailable}
                  >
                    open
                  </Button>
                </div>
                {!isSourceAvailable ? <Tooltip id="open-source-button-help" /> : null}
              </div>
            </div>
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
        <td className="w-1/4">Inlined</td>
        <td className="w-3/4 break-all">
          <CopyToClipboard onCopy={onCopy} text={inlinedText}>
            <button className="cursor-pointer">{inlinedText}</button>
          </CopyToClipboard>
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Binary</td>
        <td className="w-3/4 break-all">
          {mappingFile === null ? (
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
          {!isMappingBuildIDAvailable ? (
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
