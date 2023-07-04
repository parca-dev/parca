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

import {useEffect, useMemo, useState} from 'react';

import {pointer} from 'd3-selection';
import {CopyToClipboard} from 'react-copy-to-clipboard';
import {usePopper} from 'react-popper';

import {
  CallgraphNode,
  CallgraphNodeMeta,
  FlamegraphNode,
  FlamegraphNodeMeta,
  FlamegraphRootNode,
} from '@parca/client';
import {
  Location,
  Mapping,
  Function as ParcaFunction,
} from '@parca/client/dist/parca/metastore/v1alpha1/metastore';
import {useKeyDown} from '@parca/components';
import {selectHoveringNode, useAppSelector} from '@parca/store';
import {divide, getLastItem, valueFormatter} from '@parca/utilities';

import {hexifyAddress, truncateString, truncateStringReverse} from '../';
import {ExpandOnHover} from './ExpandOnHoverValue';

const NoData = (): JSX.Element => {
  return <span className="rounded bg-gray-200 px-2 dark:bg-gray-800">Not available</span>;
};

interface ExtendedCallgraphNodeMeta extends CallgraphNodeMeta {
  lineIndex: number;
  locationIndex: number;
}

interface HoveringNode extends FlamegraphRootNode, FlamegraphNode, CallgraphNode {
  diff: bigint;
  meta?: FlamegraphNodeMeta | ExtendedCallgraphNodeMeta;
}

interface GraphTooltipProps {
  x?: number;
  y?: number;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  hoveringNode?: HoveringNode;
  contextElement: Element | null;
  isFixed?: boolean;
  virtualContextElement?: boolean;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
  type?: string;
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
  type = 'flamegraph',
}: {
  hoveringNode: HoveringNode;
  onCopy: () => void;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
  type?: string;
}): JSX.Element => {
  // populate meta from the flamegraph metadata tables
  if (
    type === 'flamegraph' &&
    locations !== undefined &&
    hoveringNode.meta?.locationIndex !== undefined &&
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

  const getTextForFile = (hoveringNode: HoveringNode): string => {
    if (hoveringNode.meta?.function == null) return '<unknown>';

    return `${hoveringNode.meta.function.filename} ${
      hoveringNode.meta.line?.line !== undefined && hoveringNode.meta.line?.line !== 0n
        ? ` +${hoveringNode.meta.line.line.toString()}`
        : `${
            hoveringNode.meta.function?.startLine !== undefined &&
            hoveringNode.meta.function?.startLine !== 0n
              ? ` +${hoveringNode.meta.function.startLine}`
              : ''
          }`
    }`;
  };
  const file = getTextForFile(hoveringNode);

  return (
    <>
      <tr>
        <td className="w-1/4">File</td>
        <td className="w-3/4 break-all">
          {hoveringNode.meta?.function?.filename == null ||
          hoveringNode.meta?.function.filename === '' ? (
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
          {hoveringNode.meta?.location?.address == null ||
          hoveringNode.meta?.location.address === 0n ? (
            <NoData />
          ) : (
            <CopyToClipboard
              onCopy={onCopy}
              text={hexifyAddress(hoveringNode.meta.location.address)}
            >
              <button className="cursor-pointer">
                {hexifyAddress(hoveringNode.meta.location.address)}
              </button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
      <tr>
        <td className="w-1/4">Binary</td>
        <td className="w-3/4 break-all">
          {hoveringNode.meta?.mapping == null || hoveringNode.meta.mapping.file === '' ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={hoveringNode.meta.mapping.file}>
              <button className="cursor-pointer">
                {getLastItem(hoveringNode.meta.mapping.file)}
              </button>
            </CopyToClipboard>
          )}
        </td>
      </tr>

      <tr>
        <td className="w-1/4">Build Id</td>
        <td className="w-3/4 break-all">
          {hoveringNode.meta?.mapping == null || hoveringNode.meta?.mapping.buildId === '' ? (
            <NoData />
          ) : (
            <CopyToClipboard onCopy={onCopy} text={hoveringNode.meta.mapping.buildId}>
              <button className="cursor-pointer">
                {truncateString(getLastItem(hoveringNode.meta.mapping.buildId) as string, 28)}
              </button>
            </CopyToClipboard>
          )}
        </td>
      </tr>
    </>
  );
};

let timeoutHandle: ReturnType<typeof setTimeout> | null = null;

export const GraphTooltipContent = ({
  hoveringNode,
  unit,
  total,
  totalUnfiltered,
  isFixed,
  strings,
  mappings,
  locations,
  functions,
  type = 'flamegraph',
}: {
  hoveringNode: HoveringNode;
  unit: string;
  total: bigint;
  totalUnfiltered: bigint;
  isFixed: boolean;
  strings?: string[];
  mappings?: Mapping[];
  locations?: Location[];
  functions?: ParcaFunction[];
  type?: string;
}): JSX.Element => {
  const [isCopied, setIsCopied] = useState<boolean>(false);

  const onCopy = (): void => {
    setIsCopied(true);

    if (timeoutHandle !== null) {
      clearTimeout(timeoutHandle);
    }
    timeoutHandle = setTimeout(() => setIsCopied(false), 3000);
  };

  const hoveringNodeCumulative = hoveringNode.cumulative;
  const diff = hoveringNode.diff;
  const prevValue = hoveringNodeCumulative - diff;
  const diffRatio = diff !== 0n ? divide(diff, prevValue) : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;

  const getTextForCumulative = (hoveringNodeCumulative: bigint): string => {
    const filtered =
      totalUnfiltered > total
        ? ` / ${divide(hoveringNodeCumulative * 100n, total).toFixed(2)}% of filtered`
        : '';
    return `${valueFormatter(hoveringNodeCumulative, unit, 2)}
    (${divide(hoveringNodeCumulative * 100n, totalUnfiltered).toFixed(2)}%${filtered})`;
  };

  return (
    <div className={`flex text-sm ${isFixed ? 'w-full' : ''}`}>
      <div className={`m-auto w-full ${isFixed ? 'w-full' : ''}`}>
        <div className="min-h-52 flex w-[500px] flex-col justify-between rounded-lg border border-gray-300 bg-gray-50 p-3 shadow-lg dark:border-gray-500 dark:bg-gray-900">
          <div className="flex flex-row">
            <div className="mx-2">
              <div className="flex h-10 items-center break-all font-semibold">
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
                        hoveringNode.meta.location.address !== 0n ? (
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
              <table className="my-2 w-full table-fixed pr-0 text-gray-700 dark:text-gray-300">
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
                  {hoveringNode.diff !== undefined && diff !== 0n && (
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
                    onCopy={onCopy}
                    hoveringNode={hoveringNode}
                    strings={strings}
                    mappings={mappings}
                    locations={locations}
                    functions={functions}
                    type={type}
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

const GraphTooltip = ({
  x,
  y,
  unit,
  total,
  totalUnfiltered,
  hoveringNode: hoveringNodeProp,
  contextElement,
  isFixed = false,
  virtualContextElement = true,
  strings,
  mappings,
  locations,
  functions,
  type = 'flamegraph',
}: GraphTooltipProps): JSX.Element => {
  const hoveringNodeState = useAppSelector(selectHoveringNode);
  // @ts-expect-error
  const hoveringNode = useMemo<HoveringNode>(() => {
    const h = hoveringNodeProp ?? hoveringNodeState;
    if (h == null) {
      return h;
    }

    // Cloning the object to avoid the mutating error as this is Redux store object and we are modifying the meta object in GraphTooltipContent component.
    return {
      ...h,
      meta: {
        ...h.meta,
      },
    };
  }, [hoveringNodeProp, hoveringNodeState]);

  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(
    virtualContextElement ? virtualElement : contextElement,
    popperElement,
    {
      placement: 'bottom-start',
      strategy: 'absolute',
      modifiers: [
        {
          name: 'preventOverflow',
          options: {
            tether: false,
            altAxis: true,
            boundary: contextElement ?? undefined,
          },
        },
        {
          name: 'offset',
          options: {
            offset: [30, 30],
          },
        },
        {
          name: 'flip',
          options: {
            boundary: contextElement ?? undefined,
          },
        },
      ],
    }
  );

  const {isShiftDown} = useKeyDown();

  useEffect(() => {
    if (contextElement === null) return;
    const onMouseMove: EventListenerOrEventListenerObject = (e: Event) => {
      if (isShiftDown) {
        return;
      }

      let tooltipX = x;
      let tooltipY = y;
      if (tooltipX == null || tooltipY == null) {
        const rel = pointer(e);
        tooltipX = rel[0];
        tooltipY = rel[1];
      }
      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(
        contextElement,
        tooltipX,
        tooltipY
      );

      void popperProps.update?.();
    };

    contextElement.addEventListener('mousemove', onMouseMove);
    return () => {
      contextElement.removeEventListener('mousemove', onMouseMove);
    };
  }, [contextElement, popperProps, isShiftDown, x, y]);

  if (hoveringNode === undefined || hoveringNode == null) return <></>;

  return isFixed ? (
    <GraphTooltipContent
      hoveringNode={hoveringNode}
      unit={unit}
      total={total}
      totalUnfiltered={totalUnfiltered}
      isFixed={isFixed}
      type={type}
    />
  ) : (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper}>
      <GraphTooltipContent
        hoveringNode={hoveringNode}
        unit={unit}
        total={total}
        totalUnfiltered={totalUnfiltered}
        isFixed={isFixed}
        strings={strings}
        mappings={mappings}
        locations={locations}
        functions={functions}
        type={type}
      />
    </div>
  );
};

export default GraphTooltip;
