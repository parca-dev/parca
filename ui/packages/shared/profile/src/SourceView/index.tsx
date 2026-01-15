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

import React, {useCallback, useEffect, useMemo} from 'react';

import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';
import {Item, Menu, useContextMenu} from 'react-contexify';

import {Source} from '@parca/client';
import {SourceSkeleton, useParcaContext, useURLState, type ProfileData} from '@parca/components';

import {ExpandOnHover} from '../GraphTooltipArrow/ExpandOnHoverValue';
import {truncateStringReverse} from '../utils';
import {Highlighter, profileAwareRenderer} from './Highlighter';
import useLineRange from './useSelectedLineRange';

interface SourceViewProps {
  loading: boolean;
  data?: Source;
  total: bigint;
  filtered: bigint;
  setActionButtons?: (buttons: JSX.Element) => void;
}

const MENU_ID = 'source-view-context-menu';

export const SourceView = React.memo(function SourceView({
  data,
  loading,
  total,
  filtered,
  setActionButtons,
}: SourceViewProps): JSX.Element {
  const [sourceFileName] = useURLState<string | undefined>('source_filename');
  const {isDarkMode, sourceViewContextMenuItems = []} = useParcaContext();

  const sourceCode = useMemo(() => {
    if (data === undefined) {
      return [''];
    }
    // To use the array index as line number
    return ['', ...data.source.split('\n')];
  }, [data]);

  const {show} = useContextMenu({
    id: MENU_ID,
  });

  const {startLine, endLine} = useLineRange();

  const lineMetrics = useMemo(() => {
    if (data === undefined) {
      return new Map<number, {cumulative: bigint; flat: bigint}>();
    }
    const table = tableFromIPC(data.record);
    const lineNumbers = table.getChild('line_number');
    const cumulative = table.getChild('cumulative');
    const flat = table.getChild('flat');

    const metrics = new Map<number, {cumulative: bigint; flat: bigint}>();
    for (let i = 0; i < table.numRows; i++) {
      metrics.set(Number(lineNumbers?.get(i)), {
        cumulative: (cumulative?.get(i) as bigint) ?? 0n,
        flat: (flat?.get(i) as bigint) ?? 0n,
      });
    }
    return metrics;
  }, [data]);

  const getProfileDataForLine = useCallback(
    (line: number, newLine: number): ProfileData | undefined => {
      const metrics = lineMetrics.get(line);
      if (metrics === undefined) {
        return undefined;
      }
      if (metrics.cumulative === 0n && metrics.flat === 0n) {
        return undefined;
      }
      return {
        line: newLine,
        cumulative: Number(metrics.cumulative),
        flat: Number(metrics.flat),
      };
    },
    [lineMetrics]
  );

  const [selectedCode, profileData] = useMemo(() => {
    if (startLine === -1 && endLine === -1) {
      return ['', []];
    }
    if (startLine === endLine) {
      const profileData: ProfileData[] = [];
      const profileDataForLine = getProfileDataForLine(startLine, 1);
      if (profileDataForLine != null) {
        profileData.push(profileDataForLine);
      }

      return [sourceCode[startLine - 1], profileData];
    }
    let code = '';
    let line = 1;
    const profileData: ProfileData[] = [];
    for (let i = startLine; i <= endLine; i++) {
      code += sourceCode[i] + '\n';
      const profileDataForLine = getProfileDataForLine(i, line);
      if (profileDataForLine != null) {
        profileData.push(profileDataForLine);
      }
      line++;
    }
    return [code, profileData];
  }, [startLine, endLine, sourceCode, getProfileDataForLine]);

  useEffect(() => {
    setActionButtons?.(
      <div className="px-2">
        <ExpandOnHover
          value={sourceFileName as string}
          displayValue={truncateStringReverse(sourceFileName as string, 50)}
        />
      </div>
    );
  }, [sourceFileName, setActionButtons]);

  if (loading) {
    return (
      <div className="h-auto overflow-clip">
        <SourceSkeleton isDarkMode={isDarkMode} />
      </div>
    );
  }

  if (data === undefined) {
    return <>Source code not uploaded for this build.</>;
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const onContextMenu = (event: any): void => {
    show({
      event,
    });
  };

  return (
    <AnimatePresence>
      <motion.div
        className="h-full w-full"
        key="source-view-loaded"
        initial={{display: 'none', opacity: 0}}
        animate={{display: 'block', opacity: 1}}
        transition={{duration: 0.5}}
      >
        <Highlighter
          file={sourceFileName as string}
          content={data.source}
          renderer={profileAwareRenderer(lineMetrics, total, filtered, onContextMenu)}
        />
        {sourceViewContextMenuItems.length > 0 ? (
          <Menu id={MENU_ID}>
            {sourceViewContextMenuItems.map(item => (
              <Item key={item.id} onClick={() => item.action(selectedCode, profileData)}>
                {item.label}
              </Item>
            ))}
          </Menu>
        ) : null}
      </motion.div>
    </AnimatePresence>
  );
});

export default SourceView;
