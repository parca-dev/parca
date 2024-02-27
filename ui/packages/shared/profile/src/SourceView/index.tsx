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

import React, {MouseEvent, useEffect, useMemo} from 'react';

import {tableFromIPC} from 'apache-arrow';
import {AnimatePresence, motion} from 'framer-motion';
import {Item, Menu, TriggerEvent, useContextMenu} from 'react-contexify';

import {Source} from '@parca/client';
import {SourceSkeleton, useParcaContext, useURLState} from '@parca/components';

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
  const [sourceFileName] = useURLState({param: 'source_filename', navigateTo: () => {}});
  const {isDarkMode, sourceViewContextMenuItems = []} = useParcaContext();

  const sourceCode = useMemo(() => {
    if (data === undefined) {
      return [''];
    }
    return data.source.split('\n');
  }, [data]);

  const {show} = useContextMenu({
    id: MENU_ID,
  });

  const {startLine, endLine} = useLineRange();

  const selectedCode = useMemo(() => {
    if (startLine === -1 && endLine === -1) {
      return '';
    }
    if (startLine === endLine) {
      return sourceCode[startLine - 1];
    }
    let code = '';
    for (let i = startLine - 1; i < endLine; i++) {
      code += sourceCode[i] + '\n';
    }
    return code;
  }, [startLine, endLine, sourceCode]);

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

  const table = tableFromIPC(data.record);
  const cumulative = table.getChild('cumulative');
  const flat = table.getChild('flat');

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
          renderer={profileAwareRenderer(cumulative, flat, total, filtered, onContextMenu)}
        />
        <Menu id={MENU_ID}>
          {sourceViewContextMenuItems.map(item => (
            <Item key={item.id} onClick={() => item.action(selectedCode)}>
              {item.label}
            </Item>
          ))}
        </Menu>
      </motion.div>
    </AnimatePresence>
  );
});

export default SourceView;
