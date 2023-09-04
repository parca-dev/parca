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

import {useState} from 'react';

import {Table} from 'apache-arrow';
import cx from 'classnames';
import {CopyToClipboard} from 'react-copy-to-clipboard';
import {Tooltip} from 'react-tooltip';
import {useWindowSize} from 'react-use';

import {Button, IconButton, useParcaContext} from '@parca/components';
import {USER_PREFERENCES, useUserPreference} from '@parca/hooks';
import {getLastItem} from '@parca/utilities';

import {hexifyAddress, truncateString, truncateStringReverse} from '../../utils';
import {ExpandOnHover} from '../ExpandOnHoverValue';
import {useGraphTooltip} from '../useGraphTooltip';
import {useGraphTooltipMetaInfo} from '../useGraphTooltipMetaInfo';

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

interface Props {
  table: Table<any>;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  row: number | null;
  level: number;
}

const InfoSection = ({
  title,
  value,
  onCopy,
  copyText,
  minWidth = '',
}: {
  title: string;
  value: string | JSX.Element;
  copyText: string;
  onCopy: () => void;
  minWidth?: string;
}): JSX.Element => {
  return (
    <div className={cx('flex shrink-0 flex-col gap-1 p-2', {[minWidth]: minWidth != null})}>
      <p className="text-sm font-medium leading-5 text-gray-500 dark:text-gray-400">{title}</p>
      <div className="text-lg font-normal text-gray-900 dark:text-gray-50">
        <CopyToClipboard onCopy={onCopy} text={copyText}>
          <button>{value}</button>
        </CopyToClipboard>
      </div>
    </div>
  );
};

export const DockedGraphTooltip = ({
  table,
  unit,
  total,
  totalUnfiltered,
  row,
  level,
}: Props): JSX.Element => {
  let {width} = useWindowSize();
  const {profileExplorer, navigateTo, enableSourcesView} = useParcaContext();
  const {PaddingX} = profileExplorer ?? {PaddingX: 0};
  width = width - PaddingX - 24;
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const onCopy = (): void => {
    setIsCopied(true);

    if (timeoutHandle !== null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  const graphTooltipData = useGraphTooltip({
    table,
    unit,
    total,
    totalUnfiltered,
    row,
    level,
  });

  const {
    labelPairs,
    functionFilename,
    file,
    openFile,
    isSourceAvailable,
    locationAddress,
    mappingFile,
    mappingBuildID,
  } = useGraphTooltipMetaInfo({table, row: row ?? 0, navigateTo});

  const [_, setIsDocked] = useUserPreference(USER_PREFERENCES.GRAPH_METAINFO_DOCKED.key);

  if (graphTooltipData === null) {
    return <></>;
  }

  const {name, cumulativeText, diffText, diff} = graphTooltipData;

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

  const addressText = locationAddress !== 0n ? hexifyAddress(locationAddress) : 'unknown';
  const fileText = functionFilename !== '' ? file : 'Not available';

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
            onClick={() => setIsDocked(false)}
            icon="mdi:dock-window"
            title="Undock MetaInfo Panel"
          />
        </div>
        <div className="flex justify-between gap-3">
          <InfoSection
            title="Cumulative"
            value={cumulativeText}
            onCopy={onCopy}
            copyText={cumulativeText}
            minWidth="w-44"
          />
          {diff !== 0n ? (
            <InfoSection
              title="Diff"
              value={diffText}
              onCopy={onCopy}
              copyText={diffText}
              minWidth="w-44"
            />
          ) : null}
          <InfoSection
            title="File"
            value={
              <div className="flex gap-2">
                <ExpandOnHover
                  value={fileText}
                  displayValue={truncateStringReverse(fileText, 45)}
                />
                <div
                  className={cx('flex items-center gap-2', {
                    hidden: enableSourcesView === false || functionFilename === '',
                  })}
                >
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
            }
            onCopy={onCopy}
            copyText={file}
            minWidth={'w-[460px]'}
          />
          <InfoSection
            title="Address"
            value={addressText}
            onCopy={onCopy}
            copyText={addressText}
            minWidth="w-44"
          />
          <InfoSection
            title="Binary"
            value={(mappingFile != null ? getLastItem(mappingFile) : null) ?? 'Not available'}
            onCopy={onCopy}
            copyText={mappingFile ?? 'Not available'}
            minWidth="w-44"
          />
          <InfoSection
            title="Build ID"
            value={truncateString(getLastItem(mappingBuildID) ?? 'Not available', 28)}
            onCopy={onCopy}
            copyText={mappingBuildID ?? 'Not available'}
          />
        </div>
        <div>
          <div className="flex h-5 gap-1">{labels}</div>
        </div>
        <span className="mx-2 block text-xs text-gray-500">
          {isCopied ? 'Copied!' : 'Hold shift and click on a value to copy.'}
        </span>
      </div>
    </div>
  );
};
