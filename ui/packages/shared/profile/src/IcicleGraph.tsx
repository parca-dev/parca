import React, {MouseEvent, useEffect, useRef, useState} from 'react';
import {throttle} from 'lodash';
import {pointer} from 'd3-selection';
import {scaleLinear} from 'd3-scale';
import {Flamegraph, FlamegraphNode, FlamegraphRootNode} from '@parca/client';
import {usePopper} from 'react-popper';
import {getLastItem, valueFormatter, diffColor} from '@parca/functions';
import {useAppSelector, selectDarkMode} from '@parca/store';

interface IcicleGraphProps {
  graph: Flamegraph;
  sampleUnit: string;
  width?: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
}

interface IcicleGraphNodesProps {
  data: FlamegraphNode[];
  x: number;
  y: number;
  total: number;
  totalWidth: number;
  level: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (node: FlamegraphNode | FlamegraphRootNode | undefined) => void;
  path: string[];
  xScale: (value: number) => number;
}

interface FlamegraphTooltipProps {
  x: number;
  y: number;
  unit: string;
  total: number;
  hoveringNode: FlamegraphNode | FlamegraphRootNode | undefined;
  contextElement: Element | null;
}

interface IcicleGraphRootNodeProps {
  node: FlamegraphRootNode;
  xScale: (value: number) => number;
  total: number;
  totalWidth: number;
  curPath: string[];
  setCurPath: (path: string[]) => void;
  setHoveringNode: (node: FlamegraphNode | FlamegraphRootNode | undefined) => void;
}

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

export function nodeLabel(node: FlamegraphNode): string {
  if (node.meta === undefined) return '<unknown>';
  const mapping = `${
    node.meta?.mapping?.file !== undefined && node.meta?.mapping?.file !== ''
      ? '[' + getLastItem(node.meta.mapping.file) + '] '
      : ''
  }`;
  if (node.meta.function?.name !== undefined && node.meta.function?.name !== '')
    return mapping + node.meta.function.name;

  const address = `${
    node.meta.location?.address !== undefined && node.meta.location?.address !== 0
      ? '0x' + node.meta.location.address.toString(16)
      : ''
  }`;
  const fallback = `${mapping}${address}`;

  return fallback === '' ? '<unknown>' : fallback;
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
        const cumulative = parseFloat(d.cumulative);
        const diff = parseFloat(d.diff);
        const start = nodes.slice(0, i).reduce((sum, d) => sum + parseFloat(d.cumulative), 0);

        const nextCurPath = curPath.length === 0 ? [] : curPath.slice(1);
        const width =
          nextCurPath.length > 0 || (nextCurPath.length === 0 && curPath.length === 1)
            ? totalWidth
            : xScale(cumulative);

        if (width <= 1) {
          return <></>;
        }

        const name = nodeLabel(d);
        const key = `${level}-${i}`;
        const nextPath = path.concat([name]);

        const color = diffColor(diff, cumulative, isDarkMode);

        const onClick = () => {
          setCurPath(nextPath);
        };

        const xStart = xScale(start);
        const newXScale =
          nextCurPath.length === 0 && curPath.length === 1
            ? scaleLinear().domain([0, cumulative]).range([0, totalWidth])
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
                data={d.children}
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

interface SourceURL {
  filename: string;
  line: string;
}

const getGithubURLFromNode = (node: FlamegraphNode): SourceURL | null => {
  if (
    node?.meta?.function?.name == null ||
    !(node.meta.function.name.startsWith('github.com') as boolean) ||
    node?.meta?.line?.line == null ||
    !(parseInt(node.meta.line.line, 10) > 0)
  ) {
    if (node?.children?.length === 0) {
      return null;
    }
    console.log('Getting from children', node?.children[0]);
    return getGithubURLFromNode(node?.children[0]);
  }

  console.log('Getting from node', node);

  // TODO consider the versioning.
  let filename = `https://${node.meta.function.name.split('pkg/')[0] ?? ''}tree/main/pkg/${
    node.meta.function.filename.split('pkg/')[1] ?? ''
  }`;

  return {
    filename,
    line: node?.meta?.line?.line,
  };
};

const FlamegraphNodeTooltipTableRows = ({
  hoveringNode,
}: {
  hoveringNode: FlamegraphNode;
}): JSX.Element => {
  useEffect(() => {
    if (hoveringNode === undefined) {
      return;
    }
    console.log('hoveringNode', hoveringNode);
    const githubURL = getGithubURLFromNode(hoveringNode);

    console.log('githubURL', githubURL);
  }, [hoveringNode]);

  if (hoveringNode.meta === undefined) return <></>;
  const githubURL = getGithubURLFromNode(hoveringNode);
  console.log(`${githubURL.filename}${githubURL?.line != null ? `#L${githubURL?.line}` : ''}`);

  return (
    <>
      {githubURL?.filename != null ? (
        <a href={`${githubURL?.filename}${githubURL?.line != null ? `#L${githubURL?.line}` : ''}`}>
          Open Github
        </a>
      ) : (
        <p>No Github URl</p>
      )}
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

  const hoveringNodeCumulative = parseFloat(hoveringNode.cumulative);
  const diff = hoveringNode.diff === undefined ? 0 : parseFloat(hoveringNode.diff);
  const prevValue = hoveringNodeCumulative - diff;
  const diffRatio = Math.abs(diff) > 0 ? diff / prevValue : 0;
  const diffSign = diff > 0 ? '+' : '';
  const diffValueText = diffSign + valueFormatter(diff, unit, 1);
  const diffPercentageText = diffSign + (diffRatio * 100).toFixed(2) + '%';
  const diffText = `${diffValueText} (${diffPercentageText})`;

  const hoveringFlamegraphNode = hoveringNode as FlamegraphNode;
  const metaRows =
    hoveringFlamegraphNode.meta === undefined ? (
      <></>
    ) : (
      <FlamegraphNodeTooltipTableRows hoveringNode={hoveringNode as FlamegraphNode} />
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
                      {hoveringFlamegraphNode.meta.function !== undefined &&
                      hoveringFlamegraphNode.meta.function.name !== '' ? (
                        <p>{hoveringFlamegraphNode.meta.function.name}</p>
                      ) : (
                        <>
                          {hoveringFlamegraphNode.meta.location !== undefined &&
                          parseInt(hoveringFlamegraphNode.meta.location.address, 10) !== 0 ? (
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
                          {valueFormatter(hoveringNodeCumulative, unit, 2)} (
                          {((hoveringNodeCumulative * 100) / total).toFixed(2)}%)
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

  const cumulative = parseFloat(node.cumulative);
  const diff = parseFloat(node.diff);
  const color = diffColor(diff, cumulative, isDarkMode);

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
        data={node.children}
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

export default function IcicleGraph({
  graph,
  width,
  setCurPath,
  curPath,
  sampleUnit,
}: IcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<
    FlamegraphNode | FlamegraphRootNode | undefined
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

  const total = parseFloat(graph.total);
  const xScale = scaleLinear().domain([0, total]).range([0, width]);

  return (
    <div onMouseLeave={() => setHoveringNode(undefined)}>
      <FlamegraphTooltip
        unit={sampleUnit}
        total={total}
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
            total={total}
            totalWidth={width}
          />
        </g>
      </svg>
    </div>
  );
}
