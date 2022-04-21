import React, {MouseEvent, useEffect, useRef, useState} from 'react';
import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';
import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {usePopper} from 'react-popper';
import {getLastItem, valueFormatter} from '@parca/functions';
import {useAppSelector, selectDarkMode} from '@parca/store';

const RowHeight = 26;

const icicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
};
const fadedIcicleRectStyles = {
  cursor: 'pointer',
  transition: 'opacity .15s linear',
  opacity: '0.5',
};

interface IcicleRectProps {
  x: number;
  y: number;
  width: number;
  height: number;
  color: string;
  name: string;
  onMouseEnter: (e: MouseEvent) => void;
  onMouseLeave: (e: MouseEvent) => void;
  onClick: (e: MouseEvent) => void;
  curPath: string[];
}

function IcicleRect({
  x,
  y,
  width,
  height,
  color,
  name,
  onMouseEnter,
  onMouseLeave,
  onClick,
  curPath,
}: IcicleRectProps) {
  const isFaded = curPath.length > 0 && name !== curPath[curPath.length - 1];
  const styles = isFaded ? fadedIcicleRectStyles : icicleRectStyles;

  return (
    <g
      transform={`translate(${x + 1}, ${y + 1})`}
      style={styles}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      onClick={onClick}
    >
      <rect
        x={0}
        y={0}
        width={width - 1}
        height={height - 1}
        style={{
          fill: color,
        }}
      />
      {width > 5 && (
        <svg width={width - 5} height={height}>
          <text x={5} y={15} style={{fontSize: '12px'}}>
            {name}
          </text>
        </svg>
      )}
    </g>
  );
}

interface IcicleGraphNodesProps {
  data: FlamegraphNode.AsObject[];
  x: number;
  y: number;
  total: number;
  totalWidth: number;
  level: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (
    node: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined
  ) => void;
  path: string[];
  xScale: (value: number) => number;
}

export function nodeLabel(node: FlamegraphNode.AsObject): string {
  if (node.meta === undefined) return '<unknown>';
  const mapping = `${
    node.meta?.mapping?.file !== undefined && node.meta?.mapping?.file !== ''
      ? '[' + getLastItem(node.meta.mapping.file) + '] '
      : ''
  }`;
  if (node.meta.pb_function?.name !== undefined && node.meta.pb_function?.name !== '')
    return mapping + node.meta.pb_function.name;

  const address = `${
    node.meta.location?.address !== undefined && node.meta.location?.address !== 0
      ? '0x' + node.meta.location.address.toString(16)
      : ''
  }`;
  const fallback = `${mapping}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
}

function diffColor(diff: number, cumulative: number, isDarkMode: boolean): string {
  const prevValue = cumulative - diff;
  const diffRatio = prevValue > 0 ? (Math.abs(diff) > 0 ? diff / prevValue : 0) : 1.0;

  const diffTransparency =
    Math.abs(diff) > 0 ? Math.min((Math.abs(diffRatio) / 2 + 0.5) * 0.8, 0.8) : 0;

  const newSpanColor = isDarkMode ? '#B3BAE1' : '#929FEB';
  const increasedSpanColor = isDarkMode
    ? `rgba(255, 177, 204, ${diffTransparency})`
    : `rgba(254, 153, 187, ${diffTransparency})`;
  const reducedSpanColor = isDarkMode
    ? `rgba(103, 158, 92, ${diffTransparency})`
    : `rgba(164, 214, 153, ${diffTransparency})`;

  const color = diff === 0 ? newSpanColor : diff > 0 ? increasedSpanColor : reducedSpanColor;

  return color;
}

export function IcicleGraphNodes({
  data,
  x,
  y,
  xScale,
  total,
  totalWidth,
  level,
  setHoveringNode,
  path,
  setCurPath,
  curPath,
}: IcicleGraphNodesProps) {
  const isDarkMode = useAppSelector(selectDarkMode);

  const nodes =
    curPath.length === 0 ? data : data.filter(d => d != null && curPath[0] === nodeLabel(d));

  const nextLevel = level + 1;

  return (
    <g transform={`translate(${x}, ${y})`}>
      {nodes.map((d, i) => {
        const start = nodes.slice(0, i).reduce((sum, d) => sum + d.cumulative, 0);

        const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
        const width =
          nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
            ? totalWidth
            : xScale(d.cumulative);

        if (width <= 1) {
          return <></>;
        }

        const key = `${level}-${i}`;
        const name = nodeLabel(d);
        const nextPath = path.concat([name]);

        const color = diffColor(d.diff === undefined ? 0 : d.diff, d.cumulative, isDarkMode);

        const onClick = () => {
          setCurPath(nextPath);
        };

        const xStart = xScale(start);
        const newXScale =
          nextCurPath.length === 0 && curPath.length === 1
            ? scaleLinear().domain([0, d.cumulative]).range([0, totalWidth])
            : xScale;

        const onMouseEnter = () => setHoveringNode(d);
        const onMouseLeave = () => setHoveringNode(undefined);

        return (
          <React.Fragment>
            <IcicleRect
              key={`rect-${key}`}
              x={xStart}
              y={0}
              width={width}
              height={RowHeight}
              name={name}
              color={color}
              onClick={onClick}
              onMouseEnter={onMouseEnter}
              onMouseLeave={onMouseLeave}
              curPath={curPath}
            />
            {data !== undefined && data.length > 0 && (
              <IcicleGraphNodes
                key={`node-${key}`}
                data={d.childrenList}
                x={xStart}
                y={RowHeight}
                xScale={newXScale}
                total={total}
                totalWidth={totalWidth}
                level={nextLevel}
                setHoveringNode={setHoveringNode}
                path={nextPath}
                curPath={nextCurPath}
                setCurPath={setCurPath}
              />
            )}
          </React.Fragment>
        );
      })}
    </g>
  );
}

const MemoizedIcicleGraphNodes = React.memo(IcicleGraphNodes);

interface FlamegraphTooltipProps {
  x: number;
  y: number;
  unit: string;
  total: number;
  hoveringNode: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined;
  contextElement: Element | null;
}

const FlamegraphNodeTooltipTableRows = ({
  hoveringNode,
}: {
  hoveringNode: FlamegraphNode.AsObject;
}): JSX.Element => {
  if (hoveringNode.meta === undefined) return <></>;

  return (
    <>
      {hoveringNode.meta.pb_function?.filename !== undefined &&
        hoveringNode.meta.pb_function?.filename !== '' && (
          <tr>
            <td className="w-1/5">File</td>
            <td className="w-4/5">
              {hoveringNode.meta.pb_function.filename}
              {hoveringNode.meta.line?.line !== undefined && hoveringNode.meta.line?.line !== 0
                ? ` +${hoveringNode.meta.line.line.toString()}`
                : `${
                    hoveringNode.meta.pb_function?.startLine !== undefined &&
                    hoveringNode.meta.pb_function?.startLine !== 0
                      ? ` +${hoveringNode.meta.pb_function.startLine.toString()}`
                      : ''
                  }`}
            </td>
          </tr>
        )}
      {hoveringNode.meta.location?.address !== undefined &&
        hoveringNode.meta.location?.address !== 0 && (
          <tr>
            <td className="w-1/5">Address</td>
            <td className="w-4/5">{' 0x' + hoveringNode.meta.location.address.toString(16)}</td>
          </tr>
        )}
      {hoveringNode.meta.mapping !== undefined && hoveringNode.meta.mapping.file !== '' && (
        <tr>
          <td className="w-1/5">Binary</td>
          <td className="w-4/5">{getLastItem(hoveringNode.meta.mapping.file)}</td>
        </tr>
      )}
    </>
  );
};

function generateGetBoundingClientRect(contextElement: Element, x = 0, y = 0) {
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
    } as ClientRect);
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
    } as ClientRect),
};

export const FlamegraphTooltip = ({
  x,
  y,
  unit,
  total,
  hoveringNode,
  contextElement,
}: FlamegraphTooltipProps): JSX.Element => {
  const [popperElement, setPopperElement] = useState<HTMLDivElement | null>(null);

  const {styles, attributes, ...popperProps} = usePopper(virtualElement, popperElement, {
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
  });

  const update = popperProps.update;

  useEffect(() => {
    if (contextElement != null) {
      virtualElement.getBoundingClientRect = generateGetBoundingClientRect(contextElement, x, y);
      update?.();
    }
  }, [x, y, contextElement, update]);

  if (hoveringNode === undefined || hoveringNode == null) return <></>;

  const diff = hoveringNode.diff === undefined ? 0 : hoveringNode.diff;
  const prevValue = hoveringNode.cumulative - diff;
  const diffRatio = Math.abs(diff) > 0 ? diff / prevValue : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;

  const hoveringFlamegraphNode = hoveringNode as FlamegraphNode.AsObject;
  const metaRows =
    hoveringFlamegraphNode.meta === undefined ? (
      <></>
    ) : (
      <FlamegraphNodeTooltipTableRows hoveringNode={hoveringNode as FlamegraphNode.AsObject} />
    );

  return (
    <div ref={setPopperElement} style={styles.popper} {...attributes.popper}>
      <div className="flex">
        <div className="m-auto">
          <div
            className="border-gray-300 dark:border-gray-500 bg-gray-50 dark:bg-gray-900 rounded-lg p-3 shadow-lg opacity-90"
            style={{borderWidth: 1}}
          >
            <div className="flex flex-row">
              <div className="ml-2 mr-6">
                <span className="font-semibold">
                  {hoveringFlamegraphNode.meta === undefined ? (
                    <p>root</p>
                  ) : (
                    <>
                      {hoveringFlamegraphNode.meta.pb_function !== undefined &&
                      hoveringFlamegraphNode.meta.pb_function.name !== '' ? (
                        <p>{hoveringFlamegraphNode.meta.pb_function.name}</p>
                      ) : (
                        <>
                          {hoveringFlamegraphNode.meta.location !== undefined &&
                          hoveringFlamegraphNode.meta.location.address !== 0 ? (
                            <p>
                              {'0x' + hoveringFlamegraphNode.meta.location.address.toString(16)}
                            </p>
                          ) : (
                            <p>unknown</p>
                          )}
                        </>
                      )}
                    </>
                  )}
                </span>
                <span className="text-gray-700 dark:text-gray-300 my-2">
                  <table className="table-fixed">
                    <tbody>
                      <tr>
                        <td className="w-1/5">Cumulative</td>
                        <td className="w-4/5">
                          {valueFormatter(hoveringNode.cumulative, unit, 2)} (
                          {((hoveringNode.cumulative * 100) / total).toFixed(2)}%)
                        </td>
                      </tr>
                      {hoveringNode.diff !== undefined && diff !== 0 && (
                        <tr>
                          <td className="w-1/5">Diff</td>
                          <td className="w-4/5">{diffText}</td>
                        </tr>
                      )}
                      {metaRows}
                    </tbody>
                  </table>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

interface IcicleGraphRootNodeProps {
  node: FlamegraphRootNode.AsObject;
  xScale: (value: number) => number;
  total: number;
  totalWidth: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (
    node: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined
  ) => void;
}

export function IcicleGraphRootNode({
  node,
  xScale,
  total,
  totalWidth,
  setHoveringNode,
  setCurPath,
  curPath,
}: IcicleGraphRootNodeProps) {
  const isDarkMode = useAppSelector(selectDarkMode);

  const color = diffColor(node.diff === undefined ? 0 : node.diff, node.cumulative, isDarkMode);

  const onClick = () => setCurPath([]);
  const onMouseEnter = () => setHoveringNode(node);
  const onMouseLeave = () => setHoveringNode(undefined);
  const path = [];

  return (
    <g transform={'translate(0, 0)'}>
      <IcicleRect
        x={0}
        y={0}
        width={totalWidth}
        height={RowHeight}
        name={'root'}
        color={color}
        onClick={onClick}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        curPath={curPath}
      />
      <MemoizedIcicleGraphNodes
        data={node.childrenList}
        x={0}
        y={RowHeight}
        xScale={xScale}
        total={total}
        totalWidth={totalWidth}
        level={0}
        setHoveringNode={setHoveringNode}
        path={path}
        curPath={curPath}
        setCurPath={setCurPath}
      />
    </g>
  );
}

const MemoizedIcicleGraphRootNode = React.memo(IcicleGraphRootNode);

interface IcicleGraphProps {
  graph: Flamegraph.AsObject;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
}

export default function IcicleGraph({graph, width, setCurPath, curPath}: IcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<
    FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined
  >();
  const [pos, setPos] = useState([0, 0]);
  const [height, setHeight] = useState(0);
  const svg = useRef(null);
  const ref = useRef<SVGGElement>(null);

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height);
    }
  }, [width]);

  if (graph.root === undefined || width === undefined) return <></>;

  const throttledSetPos = throttle(setPos, 20);
  const onMouseMove = (e: React.MouseEvent<SVGSVGElement | HTMLDivElement>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e);

    throttledSetPos([rel[0], rel[1]]);
  };

  const xScale = scaleLinear().domain([0, graph.total]).range([0, width]);

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)}>
      <FlamegraphTooltip
        unit={graph.unit}
        total={graph.total}
        x={pos[0]}
        y={pos[1]}
        hoveringNode={hoveringNode}
        contextElement={svg.current}
      />
      <svg
        className="font-robotoMono"
        width={width}
        height={height}
        onMouseMove={onMouseMove}
        ref={svg}
      >
        <g ref={ref}>
          <MemoizedIcicleGraphRootNode
            node={graph.root}
            setHoveringNode={setHoveringNode}
            curPath={curPath}
            setCurPath={setCurPath}
            xScale={xScale}
            total={graph.total}
            totalWidth={width}
          />
        </g>
      </svg>
    </div>
  );
}
