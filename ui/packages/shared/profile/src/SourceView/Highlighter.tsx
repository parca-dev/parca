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

import {useId} from 'react';

import {Vector} from 'apache-arrow';
import cx from 'classnames';
import {scaleLinear} from 'd3-scale';
import SyntaxHighlighter, {createElement, type createElementProps} from 'react-syntax-highlighter';
import {atomOneDark, atomOneLight} from 'react-syntax-highlighter/dist/esm/styles/hljs';
import {Tooltip} from 'react-tooltip';

import {useURLState} from '@parca/components';
import {selectDarkMode, useAppSelector} from '@parca/store';

import {useProfileViewContext} from '../ProfileView/ProfileViewContext';
import {LineNo} from './LineNo';

interface RendererProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  rows: any[];
  stylesheet: createElementProps['stylesheet'];
  useInlineStyles: createElementProps['useInlineStyles'];
}

type Renderer = ({rows, stylesheet, useInlineStyles}: RendererProps) => JSX.Element;

interface HighlighterProps {
  content: string;
  language?: string;
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
  const {sampleUnit} = useProfileViewContext();
  if (value === 0n) {
    return <div className={cx(commonClasses)} />;
  }
  const unfilteredPercent = (Number(value) / Number(total + filtered)) * 100;
  const filteredPercent = (Number(value) / Number(total)) * 100;

  const valueString = value.toString();
  const valueWithUnit = `${valueString}${sampleUnit}`;

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
        {valueString}
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
  filtered: bigint
): Renderer => {
  return function ProfileAwareRenderer({
    rows,
    stylesheet,
    useInlineStyles,
  }: RendererProps): JSX.Element {
    const lineNumberWidth = charsToWidth(rows.length.toString().length);
    const [sourceLine, setSourceLine] = useURLState({param: 'source_line', navigateTo: () => {}});
    return (
      <>
        {rows.map((node, i) => {
          const lineNumber: number = node.children[0].children[0].value as number;
          const isCurrentLine = sourceLine === lineNumber.toString();
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
                  setCurrentLine={() => setSourceLine(lineNumber.toString())}
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
                  'w-11/12 flex-grow-0 border-l border-gray-200 pl-1 dark:border-gray-700',
                  {
                    'bg-yellow-200 dark:bg-yellow-700': isCurrentLine,
                  }
                )}
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

export const Highlighter = ({content, language, renderer}: HighlighterProps): JSX.Element => {
  const isDarkMode = useAppSelector(selectDarkMode);

  return (
    <div className="relative">
      <div className="flex gap-2 text-xs">
        <div
          className={cx('text-right', charsToWidth(content.split('\n').length.toString().length))}
        >
          Line
        </div>
        <div className="flex gap-3">
          <div>Cumulative</div>
          <div>Flat</div>
          <div>Source</div>
        </div>
      </div>
      <div className="text-xs">
        <SyntaxHighlighter
          language={language}
          style={isDarkMode ? atomOneDark : atomOneLight}
          showLineNumbers
          renderer={renderer}
          customStyle={{padding: 0, height: '80vh'}}
        >
          {content}
        </SyntaxHighlighter>
      </div>
    </div>
  );
};
