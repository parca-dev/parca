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

import {CopyToClipboard} from 'react-copy-to-clipboard';
import {useState, useEffect} from 'react';
import {usePopper} from 'react-popper';

import {CallgraphNode, FlamegraphNode, FlamegraphNodeMeta, FlamegraphRootNode} from '@parca/client';
import {getLastItem, valueFormatter} from '@parca/functions';
import {hexifyAddress, truncateString, truncateStringReverse} from '../';
import {useKeyDown} from '@parca/components';
import {
  Function as ParcaFunction,
  Location,
  Mapping,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {ExpandOnHover} from './ExpandOnHoverValue';

interface GraphTooltipProps {
  x: number;
  y: number;
  unit: string;
  total: number;
  hoveringNode: HoveringNode;
  contextElement: Element | null;
  isFixed?: boolean;
  virtualContextElement?: boolean;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
}

const virtualElement = {
  getBoundingClientRect: () =>
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    ({
      width: 0,
      height: 0,
      top: 0,
      left: 0,
      right: 0,
      bottom: 0,
    } as DOMRect),
};

function generateGetBoundingClientRect(contextElement: Element, x = 0, y = 0): () => DOMRect {
  const domRect = contextElement.getBoundingClientRect();
  return () =>
    // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
    ({
      width: 0,
      height: 0,
      top: domRect.y + y,
      left: domRect.x + x,
      right: domRect.x + x,
      bottom: domRect.y + y,
    } as DOMRect);
}

const TooltipMetaInfo = ({
  hoveringNode,
  onCopy,
  strings,
  mappings,
  locations,
  functions,
}: {
  hoveringNode: FlamegraphNode;
  onCopy: () => void;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
}): JSX.Element => {
  if (hoveringNode.meta === undefined) return <></>;

  // populate meta from the flamegraph metadata tables
  if (
    locations !== undefined &&
    hoveringNode.meta.locationIndex !== undefined &&
    hoveringNode.meta.locationIndex !== 0
  ) {
    const location = locations[hoveringNode.meta.locationIndex - 1];
    hoveringNode.meta.location = location;

    if (location !== undefined) {
      if (
        mappings !== undefined &&
        location.mappingIndex !== undefined &&
        location.mappingIndex !== 0
      ) {
        const mapping = mappings[location.mappingIndex - 1];
        if (strings !== undefined && mapping !== undefined) {
          mapping.file =
            mapping?.fileStringIndex !== undefined ? strings[mapping.fileStringIndex] : '';
          mapping.buildId =
            mapping?.buildIdStringIndex !== undefined ? strings[mapping.buildIdStringIndex] : '';
        }
        hoveringNode.meta.mapping = mapping;
      }

      if (
        functions !== undefined &&
        location.lines !== undefined &&
        hoveringNode.meta.lineIndex !== undefined &&
        hoveringNode.meta.lineIndex < location.lines.length
      ) {
        const func = functions[location.lines[hoveringNode.meta.lineIndex].functionIndex - 1];
        if (strings !== undefined) {
          func.name = strings[func.nameStringIndex];
          func.systemName = strings[func.systemNameStringIndex];
          func.filename = strings[func.filenameStringIndex];
        }
        hoveringNode.meta.function = func;
      }
    }
  }

  const getTextForFile = (hoveringNode: FlamegraphNode): string => {
    if (hoveringNode.meta?.function == null) return '<unknown>';

    return `${hoveringNode.meta.function.filename} ${
      hoveringNode.meta.line?.line !== undefined && hoveringNode.meta.line?.line !== '0'
        ? ` +${hoveringNode.meta.line.line.toString()}`
        : `${
            hoveringNode.meta.function?.startLine !== undefined &&
            hoveringNode.meta.function?.startLine !== '0'
              ? ` +${hoveringNode.meta.function.startLine}`
              : ''
          }`
    }`;
  };
  const file = getTextForFile(hoveringNode);

  return (
    <>
      {hoveringNode.meta.function?.filename !== undefined &&
        hoveringNode.meta.function?.filename !== '' && (
          <tr>
            <td className="w-1/4">File</td>
            <td className="w-3/4 break-all">
              <CopyToClipboard onCopy={onCopy} text={file}>
                <button className="cursor-pointer text-left whitespace-nowrap">
                  <ExpandOnHover value={file} displayValue={truncateStringReverse(file, 40)} />
                </button>
              </CopyToClipboard>
            </td>
          </tr>
        )}
      {hoveringNode.meta.location?.address !== undefined &&
        hoveringNode.meta.location?.address !== '0' && (
          <tr>
            <td className="w-1/4">Address</td>
            <td className="w-3/4 break-all">
              <CopyToClipboard
                onCopy={onCopy}
                text={hexifyAddress(hoveringNode.meta.location.address)}
              >
                <button className="cursor-pointer">
                  {hexifyAddress(hoveringNode.meta.location.address)}
                </button>
              </CopyToClipboard>
            </td>
          </tr>
        )}
      {hoveringNode.meta.mapping !== undefined && hoveringNode.meta.mapping.file !== '' && (
        <tr>
          <td className="w-1/4">Binary</td>
          <td className="w-3/4 break-all">
            <CopyToClipboard onCopy={onCopy} text={hoveringNode.meta.mapping.file}>
              <button className="cursor-pointer">
                {getLastItem(hoveringNode.meta.mapping.file)}
              </button>
            </CopyToClipboard>
          </td>
        </tr>
      )}
      {hoveringNode.meta.mapping !== undefined && hoveringNode.meta.mapping.buildId !== '' && (
        <tr>
          <td className="w-1/4">Build Id</td>
          <td className="w-3/4 break-all">
            <CopyToClipboard onCopy={onCopy} text={hoveringNode.meta.mapping.buildId}>
              <button className="cursor-pointer">
                {truncateString(getLastItem(hoveringNode.meta.mapping.buildId) as string, 28)}
              </button>
            </CopyToClipboard>
          </td>
        </tr>
      )}
    </>
  );
};

// @ts-expect-error
export interface HoveringNode extends CallgraphNode, FlamegraphRootNode, FlamegraphNode {
  diff: string;
  meta?: FlamegraphNodeMeta | {[key: string]: any};
}

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

const GraphTooltipContent = ({
  hoveringNode,
  unit,
  total,
  isFixed,
  strings,
  mappings,
  locations,
  functions,
}: {
  hoveringNode: HoveringNode;
  unit: string;
  total: number;
  isFixed: boolean;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
}): JSX.Element => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const onCopy = (): void => {
    setIsCopied(true);

    if (timeoutHandle !== null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  const hoveringNodeCumulative = parseFloat(hoveringNode.cumulative);
  const diff = hoveringNode.diff === undefined ? 0 : parseFloat(hoveringNode.diff);
  const prevValue = hoveringNodeCumulative - diff;
  const diffRatio = Math.abs(diff) > 0 ? diff / prevValue : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;
  const metaRows =
    hoveringNode.meta === undefined ? (
      <></>
    ) : (
      <TooltipMetaInfo
        onCopy={onCopy}
        // @ts-expect-error
        hoveringNode={hoveringNode}
        strings={strings}
        mappings={mappings}
        locations={locations}
        functions={functions}
      />
    );

  const getTextForCumulative = (hoveringNodeCumulative: number): string => {
    return `${valueFormatter(hoveringNodeCumulative, unit, 2)} (
      ${((hoveringNodeCumulative * 100) / total).toFixed(2)}%)`;
  };

  return (
    <div className={`text-sm flex ${isFixed ? 'w-full' : ''}`}>
      <div className={`m-auto w-full ${isFixed ? 'w-full' : ''}`}>
        <div className="border border-gray-300 dark:border-gray-500 bg-gray-50 dark:bg-gray-900 rounded-lg p-3 shadow-lg min-h-52 w-[500px] flex justify-between flex-col">
          <div className="flex flex-row">
            <div className="mx-2">
              <div className="font-semibold break-all h-10 flex items-center">
                {hoveringNode.meta === undefined ? (
                  <p>root</p>
                ) : (
                  <>
                    {hoveringNode.meta.function !== undefined &&
                    hoveringNode.meta.function.name !== '' ? (
                      <CopyToClipboard onCopy={onCopy} text={hoveringNode.meta.function.name}>
                        <button className="cursor-pointer text-left">
                          {hoveringNode.meta.function.name}
                        </button>
                      </CopyToClipboard>
                    ) : (
                      <>
                        {hoveringNode.meta.location !== undefined &&
                        parseInt(hoveringNode.meta.location.address, 10) !== 0 ? (
                          <CopyToClipboard
                            onCopy={onCopy}
                            text={hexifyAddress(hoveringNode.meta.location.address)}
                          >
                            <button className="cursor-pointer text-left">
                              {hexifyAddress(hoveringNode.meta.location.address)}
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
              <table className="table-fixed pr-0 text-gray-700 dark:text-gray-300 my-2 w-full">
                <tbody>
                  <tr>
                    <td className="w-1/4">Cumulative</td>

                    <td className="w-3/4">
                      <CopyToClipboard
                        onCopy={onCopy}
                        text={getTextForCumulative(hoveringNodeCumulative)}
                      >
                        <button className="cursor-pointer">
                          {getTextForCumulative(hoveringNodeCumulative)}
                        </button>
                      </CopyToClipboard>
                    </td>
                  </tr>
                  {hoveringNode.diff !== undefined && diff !== 0 && (
                    <tr>
                      <td className="w-1/4">Diff</td>
                      <td className="w-3/4">
                        <CopyToClipboard onCopy={onCopy} text={diffText}>
                          <button className="cursor-pointer">{diffText}</button>
                        </CopyToClipboard>
                      </td>
                    </tr>
                  )}
                  {metaRows}
                </tbody>
              </table>
            </div>
          </div>
          <span className="block text-gray-500 text-xs mx-2">
            {isCopied ? 'Copied!' : 'Hold shift and click on a value to copy.'}
          </span>
        </div>
      </div>
    </div>
  );
};

const GraphTooltip = ({
  x,
  y,
  unit,
  total,
  hoveringNode,
  contextElement,
  isFixed = false,
  virtualContextElement = true,
  strings,
  mappings,
  locations,
  functions,
}: GraphTooltipProps): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(
    virtualContextElement ? virtualElement : contextElement,
    popperElement,
    {
      placement: 'auto-start',
      strategy: 'absolute',
      modifiers: [
        {
          name: 'preventOverflow',
          options: {
            tether: false,
            altAxis: true,
          },
        },
        {
          name: 'offset',
          options: {
            offset: [30, 30],
          },
        },
      ],
    }
  );

  const update = popperProps.update;
  const {isShiftDown} = useKeyDown();

  useEffect(() => {
    if (contextElement != null) {
      if (isShiftDown) return;

      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(contextElement, x, y);
      void update?.();
    }
  }, [x, y, contextElement, update, isShiftDown]);

  if (hoveringNode === undefined || hoveringNode == null) return <></>;

  return isFixed ? (
    <GraphTooltipContent hoveringNode={hoveringNode} unit={unit} total={total} isFixed={isFixed} />
  ) : (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper}>
      <GraphTooltipContent
        hoveringNode={hoveringNode}
        unit={unit}
        total={total}
        isFixed={isFixed}
        strings={strings}
        mappings={mappings}
        locations={locations}
        functions={functions}
      />
    </div>
  );
};

export default GraphTooltip;
