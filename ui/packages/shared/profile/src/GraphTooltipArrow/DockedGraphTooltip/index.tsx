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

import {Icon} from '@iconify/react';
import {Table} from 'apache-arrow';
import cx from 'classnames';
import {useWindowSize} from 'react-use';

import {useParcaContext} from '@parca/components';
import {ProfileType} from '@parca/parser';
import {getLastItem} from '@parca/utilities';

import {hexifyAddress, truncateString, truncateStringReverse} from '../../utils';
import {useGraphTooltip} from '../useGraphTooltip';
import {useGraphTooltipMetaInfo} from '../useGraphTooltipMetaInfo';

interface Props {
  table: Table<any>;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  profileType?: ProfileType;
  unit?: string;
  compareAbsolute: boolean;
}

const InfoSection = ({
  title,
  value,
  minWidth = '',
}: {
  title: string;
  value: string | JSX.Element;
  minWidth?: string;
}): JSX.Element => {
  return (
    <div className={cx('flex shrink-0 flex-col gap-1 p-2', {[minWidth]: minWidth != null})}>
      <p className="text-sm font-medium leading-5 text-gray-500 dark:text-gray-400">{title}</p>
      <div className="text-lg font-normal text-gray-900 dark:text-gray-50">{value}</div>
    </div>
  );
};

const NoData = (): React.JSX.Element => {
  return <span className="rounded bg-gray-200 px-2 dark:bg-gray-800">Not available</span>;
};

export const DockedGraphTooltip = ({
  table,
  total,
  totalUnfiltered,
  row,
  profileType,
  unit,
  compareAbsolute,
}: Props): JSX.Element => {
  let {width} = useWindowSize();
  const {profileExplorer} = useParcaContext();
  const {PaddingX} = profileExplorer ?? {PaddingX: 0};
  width = width - PaddingX - 24;

  const graphTooltipData = useGraphTooltip({
    table,
    profileType,
    unit,
    total,
    totalUnfiltered,
    row,
    compareAbsolute,
  });

  const {
    labelPairs,
    functionFilename,
    file,
    locationAddress,
    mappingFile,
    mappingBuildID,
    inlined,
  } = useGraphTooltipMetaInfo({table, row: row ?? 0});

  if (graphTooltipData === null) {
    return <></>;
  }

  const {name, cumulativeText, flatText, diffText, diff} = graphTooltipData;

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
  const addressText = locationAddress !== 0n ? hexifyAddress(locationAddress) : <NoData />;

  return (
    <div
      className="fixed bottom-0 z-20 overflow-hidden rounded-t-lg border-l border-r border-t border-gray-400 bg-white bg-opacity-90 px-8 py-3 dark:border-gray-600 dark:bg-black dark:bg-opacity-80"
      style={{width}}
    >
      <div className="flex flex-col gap-4">
        <div className="flex justify-between gap-4">
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
        <div className="flex justify-between gap-3">
          <InfoSection title="Cumulative" value={cumulativeText} minWidth="w-44" />
          <InfoSection title="Flat" value={flatText} minWidth="w-44" />
          {diff !== 0n ? <InfoSection title="Diff" value={diffText} minWidth="w-44" /> : null}
          <InfoSection
            title="File"
            value={functionFilename !== '' ? truncateStringReverse(file, 45) : <NoData />}
            minWidth={'w-[460px]'}
          />
          <InfoSection title="Address" value={addressText} minWidth="w-44" />
          <InfoSection title="Inlined" value={inlinedText} minWidth="w-44" />
          <InfoSection
            title="Binary"
            value={(mappingFile != null ? getLastItem(mappingFile) : null) ?? <NoData />}
            minWidth="w-44"
          />
          <InfoSection
            title="Build ID"
            value={
              isMappingBuildIDAvailable ? (
                <div>{truncateString(mappingBuildID, 28)}</div>
              ) : (
                <NoData />
              )
            }
          />
        </div>
        <div>
          <div className="flex h-5 gap-1">{labels}</div>
        </div>
      </div>
      <div className="flex w-full items-center gap-1 text-xs text-gray-500">
        <Icon icon="iconoir:mouse-button-right" />
        <div>Right click to show context menu</div>
      </div>
    </div>
  );
};
