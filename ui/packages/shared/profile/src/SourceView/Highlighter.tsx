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

import {MouseEventHandler, useId, useMemo} from 'react';

import {Vector} from 'apache-arrow';
import cx from 'classnames';
import {scaleLinear} from 'd3-scale';
import {type createElementProps} from 'react-syntax-highlighter';
import createElement from 'react-syntax-highlighter/dist/cjs/create-element';
import SyntaxHighlighter from 'react-syntax-highlighter/dist/cjs/default-highlight';
import {atomOneDark, atomOneLight} from 'react-syntax-highlighter/dist/cjs/styles/hljs';
import {Tooltip} from 'react-tooltip';

import {useParcaContext} from '@parca/components';
import {valueFormatter} from '@parca/utilities';

import {useProfileViewContext} from '../ProfileView/context/ProfileViewContext';
import {LineNo} from './LineNo';
import {langaugeFromFile} from './lang-detector';
import useLineRange from './useSelectedLineRange';

interface RendererProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  rows: any[];
  stylesheet: createElementProps['stylesheet'];
  useInlineStyles: createElementProps['useInlineStyles'];
}

type Renderer = ({rows, stylesheet, useInlineStyles}: RendererProps) => JSX.Element;

interface HighlighterProps {
  file: string;
  content: string;
  renderer?: Renderer;
}

// cannot make this a function on the number as we need the classes to be static for tailwind
const charsToWidthMap: {[key: number]: string} = {
  1: 'w-3',
  2: 'w-5',
  3: 'w-7',
  4: 'w-9',
  5: 'w-11',
  6: 'w-[52px]',
  7: 'w-[60px]]',
  8: 'w-[68px]',
  9: 'w-[76px]',
  10: 'w-[84px]',
  11: 'w-[92px]',
  12: 'w-[100px]',
  13: 'w-[108px]',
  14: 'w-[116px]',
};

const intensityScale = scaleLinear().domain([0, 99]).range([0.05, 0.75]);

const LineProfileMetadata = ({
  value,
  total,
  filtered,
}: {
  value: bigint;
  total: bigint;
  filtered: bigint;
}): JSX.Element => {
  const commonClasses = 'w-[52px] shrink-0';
  const id = useId();
  const {profileSource} = useProfileViewContext();
  if (value === 0n) {
    return <div className={cx(commonClasses)} />;
  }
  const unfilteredPercent = (Number(value) / Number(total + filtered)) * 100;
  const filteredPercent = (Number(value) / Number(total)) * 100;

  const valueWithUnit = valueFormatter(
    value,
    profileSource?.ProfileType().sampleUnit ?? '',
    1,
    true
  );

  return (
    <>
      <p
        className={cx(
          'w- flex justify-end overflow-hidden text-ellipsis whitespace-nowrap',
          commonClasses
        )}
        style={{backgroundColor: `rgba(236, 151, 6, ${intensityScale(unfilteredPercent)})`}}
        data-tooltip-id={id}
        data-tooltip-content={`${valueWithUnit} (${unfilteredPercent.toFixed(2)}%${
          filtered > 0n ? ` / ${filteredPercent.toFixed(2)}%` : ''
        })`}
      >
        {valueWithUnit}
      </p>
      <Tooltip id={id} />
    </>
  );
};

const charsToWidth = (chars: number): string => {
  return charsToWidthMap[chars];
};

export const profileAwareRenderer = (
  cumulative: Vector | null,
  flat: Vector | null,
  total: bigint,
  filtered: bigint,
  onContextMenu: MouseEventHandler<HTMLDivElement>
): Renderer => {
  return function ProfileAwareRenderer({
    rows,
    stylesheet,
    useInlineStyles,
  }: RendererProps): JSX.Element {
    const lineNumberWidth = charsToWidth(rows.length.toString().length);
    const {startLine, endLine, setLineRange} = useLineRange();

    return (
      <>
        {rows.map((node, i) => {
          const lineNumber: number = node.children[0].children[0].value as number;
          const isCurrentLine = lineNumber >= startLine && lineNumber <= endLine;
          node.children = node.children.slice(1);
          return (
            <div className="flex gap-1" key={`${i}`}>
              <div
                className={cx(
                  'shrink-0 overflow-hidden border-r border-r-gray-200 text-right dark:border-r-gray-700',
                  lineNumberWidth
                )}
              >
                <LineNo
                  value={lineNumber}
                  isCurrent={isCurrentLine}
                  selectLine={(isShiftDown = false) => {
                    if (!isShiftDown) {
                      setLineRange(lineNumber, lineNumber);
                    }
                    if (isShiftDown && startLine != null) {
                      if (startLine > lineNumber) {
                        setLineRange(lineNumber, startLine);
                      } else {
                        setLineRange(startLine, lineNumber);
                      }
                    }
                  }}
                />
              </div>
              <LineProfileMetadata
                value={cumulative?.get(i) ?? 0n}
                total={total}
                filtered={filtered}
              />
              <LineProfileMetadata value={flat?.get(i) ?? 0n} total={total} filtered={filtered} />
              <div
                className={cx(
                  'w-full flex-grow-0 border-l border-gray-200 pl-1 dark:border-gray-700',
                  {
                    'bg-yellow-200 dark:bg-yellow-700': isCurrentLine,
                  }
                )}
                onContextMenu={onContextMenu}
              >
                {createElement({
                  key: `source-line-${i}`,
                  node,
                  stylesheet,
                  useInlineStyles,
                })}
              </div>
            </div>
          );
        })}
      </>
    );
  };
};

export const Highlighter = ({file, content, renderer}: HighlighterProps): JSX.Element => {
  const {isDarkMode} = useParcaContext();
  const language = useMemo(() => langaugeFromFile(file), [file]);

  return (
    <div className="relative">
      <div className="flex gap-2 text-xs">
        <div
          className={cx(
            'text-right',
            charsToWidth(
              content
                .split(
                  // prettier-ignore
                  '\n'
                )
                .length.toString().length
            )
          )}
        >
          Line
        </div>
        <div className="flex gap-3">
          <div>Cumulative</div>
          <div>Flat</div>
          <div>Source</div>
        </div>
      </div>
      <div className="text-xs overflow-auto" style={{maxHeight: 'calc(100vh - 200px)'}}>
        <SyntaxHighlighter
          language={language}
          style={isDarkMode ? atomOneDark : atomOneLight}
          showLineNumbers
          renderer={renderer}
          customStyle={{padding: 0}}
        >
          {content}
        </SyntaxHighlighter>
      </div>
    </div>
  );
};
