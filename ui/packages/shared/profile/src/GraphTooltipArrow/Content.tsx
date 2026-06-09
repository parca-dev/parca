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
import {Table} from '@uwdata/flechette';

import {useParcaContext} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {formatDateTimeDownToMS, getLastItem} from '@parca/utilities';

import {getLabelPairs} from '../ProfileFlameGraph/FlameGraphArrow/utils';
import {hexifyAddress, truncateString, truncateStringReverse} from '../utils';
import {ExpandOnHover} from './ExpandOnHoverValue';
import {gpuFrameInfosFromLabels, type GpuFrameInfo} from './gpuFrameDescriptions';
import {openInNewTab} from './openInNewTab';
import {useGraphTooltip} from './useGraphTooltip';
import {useGraphTooltipMetaInfo} from './useGraphTooltipMetaInfo';

interface GraphTooltipArrowContentProps {
  table: Table;
  profileType?: ProfileType;
  unit?: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  isFixed: boolean;
  compareAbsolute: boolean;
  frozen?: boolean;
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
  frozen = false,
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

  const gpuInfos = gpuFrameInfosFromLabels(getLabelPairs(table, rowNumber));

  // Outer card gains a subtle ring when frozen, matching the design's
  // `.is-frozen` treatment.
  const cardClassName = [
    'flex w-auto max-w-[600px] min-w-[300px] flex-col justify-start rounded-lg border bg-gray-50 p-3 shadow-lg dark:bg-gray-900',
    frozen
      ? 'border-indigo-400/60 ring-2 ring-indigo-400/20 dark:border-indigo-400/40'
      : 'border-gray-300 dark:border-gray-500',
  ].join(' ');

  return (
    <div className={`flex text-sm ${isFixed ? 'w-full' : ''}`}>
      <div className={`m-auto w-full ${isFixed ? 'w-full' : ''}`}>
        <div className={cardClassName}>
          <div className="flex flex-row">
            <div className="mx-2">
              <div className="flex min-h-10 items-start justify-between gap-4 break-all font-semibold mb-2">
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
                      <p>{cumulativeText}</p>
                    </td>
                  </tr>
                  <tr>
                    <td className="w-1/4 pt-2">Flat</td>
                    <td className="w-3/4 pt-2">
                      <p>{flatText}</p>
                    </td>
                  </tr>
                  {diff !== 0n && (
                    <tr>
                      <td className="w-1/4 pt-2">Diff</td>
                      <td className="w-3/4 pt-2">
                        <p>{diffText}</p>
                      </td>
                    </tr>
                  )}
                  <TooltipMetaInfo table={table} row={rowNumber} />
                </tbody>
              </table>
            </div>
          </div>
          {gpuInfos.map((info, i) => (
            <GpuDescriptionBlock
              key={i}
              info={info}
              // When both a SASS and a stall block are shown, keep the tooltip
              // compact by trimming the (typically longer) stall description.
              maxSentences={gpuInfos.length > 1 && info.kind === 'stall' ? 2 : undefined}
            />
          ))}
          <ShortcutFooter frozen={frozen} />
        </div>
      </div>
    </div>
  );
};

// Trims a description to at most `max` sentences, appending an ellipsis when
// anything was dropped.
const truncateToSentences = (text: string, max: number): string => {
  const sentences = text.match(/[^.!?]+[.!?]+(\s|$)|[^.!?]+$/g);
  if (sentences === null || sentences.length <= max) return text;
  return `${sentences.slice(0, max).join('').trimEnd()} …`;
};

const GpuDescriptionBlock = ({
  info,
  maxSentences,
}: {
  info: GpuFrameInfo;
  maxSentences?: number;
}): React.JSX.Element => {
  const chipPrefix = info.kind === 'stall' ? 'Stall reason' : 'SASS instruction';
  const description =
    maxSentences === undefined
      ? info.entry.description
      : truncateToSentences(info.entry.description, maxSentences);

  return (
    <div className="mx-2 mt-3 border-t border-gray-200 pt-3 dark:border-gray-700">
      <div className="mb-2 text-xs font-semibold text-gray-700 dark:text-gray-200">
        {chipPrefix} · {info.entry.reasonLabel}
      </div>
      <div className="font-mono text-[10px] uppercase tracking-wider text-gray-500 dark:text-gray-400">
        Description
      </div>
      <p className="mt-1 text-xs leading-relaxed text-gray-600 dark:text-gray-300">
        {description}
      </p>
      <button
        type="button"
        onClick={e => {
          e.preventDefault();
          e.stopPropagation();
          openInNewTab(info.sourceUrl);
        }}
        title={info.sourceLabel}
        className="mt-2 inline-flex cursor-pointer items-center gap-1 self-start text-[11px] text-indigo-600 hover:underline dark:text-indigo-400"
      >
        Docs
        <Icon icon="iconoir:open-new-window" className="opacity-80" width={11} height={11} />
      </button>
    </div>
  );
};

const ShortcutFooter = ({frozen}: {frozen: boolean}): React.JSX.Element => (
  <div className="mx-2 mt-3 flex flex-wrap items-center gap-x-3 gap-y-1 border-t border-gray-200 pt-2 text-[11px] text-gray-500 dark:border-gray-700 dark:text-gray-400">
    <span
      className={`inline-flex items-center gap-1.5 ${
        frozen ? 'text-gray-600 dark:text-gray-300' : ''
      }`}
    >
      <kbd
        className={[
          'inline-flex min-w-[18px] justify-center rounded border border-b-2 px-1 font-mono text-[10px] leading-4',
          frozen
            ? 'border-gray-300 bg-gray-200 text-gray-600 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300'
            : 'border-gray-300 bg-white text-gray-600 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-300',
        ].join(' ')}
      >
        ⇧
      </kbd>
      {frozen ? (
        <span>
          <b className="font-semibold">Frozen</b> · release to resume hover
        </span>
      ) : (
        <span>
          Hold <b className="font-semibold">Shift</b> to freeze · interact
        </span>
      )}
    </span>
    <span className="inline-block h-3 w-px bg-gray-200 dark:bg-gray-700" />
    <span className="inline-flex items-center gap-1.5">
      <Icon icon="iconoir:mouse-button-right" width={12} height={14} />
      <span>
        <b className="font-semibold">Right-click</b> for context menu
      </span>
    </span>
  </div>
);

const TooltipMetaInfo = ({table, row}: {table: Table; row: number}): React.JSX.Element => {
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
      {timestamp != null && timestamp !== 0n && (
        <tr>
          <td className="w-1/4 pt-2">Timestamp</td>
          <td className="w-3/4 pt-2 break-all">
            {formatDateTimeDownToMS(new Date(Number(timestamp / 1000000n)), timezone)}
          </td>
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
                <ExpandOnHover value={file} displayValue={truncateStringReverse(file, 50)} />
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
