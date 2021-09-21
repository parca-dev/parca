import React, { MouseEvent, useCallback, useEffect, useRef, useState } from 'react'
import { throttle, debounce } from 'lodash'
import { pointer } from 'd3-selection'
import { scaleLinear } from 'd3-scale'
import { Flamegraph, FlamegraphNode, FlamegraphRootNode } from '@parca/client'

const transitionTime = '50ms'
const transitionCurve = 'cubic-bezier(0.85, 0.69, 0.71, 1.32)'

const widthTransition = `width ${transitionTime} ${transitionCurve}`
const transformTransition = `transform  ${transitionTime} ${transitionCurve}`
const RowHeight = 20

interface IcicleRectProps {
  x: number
  y: number
  width: number
  height: number
  color: string
  name: string
  onMouseEnter: (e: MouseEvent) => void
  onMouseLeave: (e: MouseEvent) => void
  onClick: (e: MouseEvent) => void
}

function IcicleRect ({
  x,
  y,
  width,
  height,
  color,
  name,
  onMouseEnter,
  onMouseLeave,
  onClick
}: IcicleRectProps) {
  return (
    <g
      transform={`translate(${x+1}, ${y+1})`}
      style={{ cursor: 'pointer' }}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      onClick={onClick}
    >
      <rect
        x={0}
        y={0}
        width={width-1}
        height={height-1}
        style={{
          fill: color,
        }}
      />
      { width > 5 && (
        <svg width={width - 5} height={height}>
          <text
            x={5}
            y={13}
            style={{ fontSize: '12px' }}
          >
            {name}
          </text>
        </svg>
      )}
    </g>
  )
}

interface IcicleGraphNodesProps {
  data: FlamegraphNode.AsObject[]
  x: number
  y: number
  width: number
  level: number
  curPath: string[]
  setCurPath: (path: string[]) => void
  setHoveringNode: (node: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined) => void
  path: string[]
}

function diffColor(diff: number, cumulative: number): string {
    const prevValue = cumulative - diff
    const diffRatio = prevValue > 0 ? (Math.abs(diff) > 0 ? (diff / prevValue) : 0) : (1.0)

    const diffTransparency = Math.abs(diff) > 0 ? Math.min(((Math.abs(diffRatio) / 2) + 0.5)*0.8, 0.8) : 0
    const color = diff == 0 ? "#90c7e0" : (diff > 0 ? `rgba(221, 46, 69, ${diffTransparency})` : `rgba(59, 165, 93, ${diffTransparency})`)

    return color
}

const getLastItem = thePath => thePath.substring(thePath.lastIndexOf('/') + 1)

export function nodeLabel(node: FlamegraphNode.AsObject): string {
    if (node.meta === undefined) return '<unknown>'
    const mapping = `${(node.meta.mapping !== undefined && node.meta.mapping.file != '') ? '['+getLastItem(node.meta.mapping.file)+'] ' : ''}`
    if (node.meta.pb_function !== undefined && node.meta.pb_function.name != '') return mapping+node.meta.pb_function.name

    const address = `${(node.meta.location !== undefined && node.meta.location.address !== undefined && node.meta.location.address != 0) ? '0x'+node.meta.location.address.toString(16) : ''}`
    const fallback = `${mapping}${address}`

    return fallback == '' ? '<unknown>' : fallback
}

export function IcicleGraphNodes ({
  data,
  x,
  y,
  width,
  level,
  setHoveringNode,
  path,
  setCurPath,
  curPath
}: IcicleGraphNodesProps) {
  const nodes = curPath.length == 0 ? data : data.filter((d, i) => d != null && curPath[0] == nodeLabel(d))

  const xScale = scaleLinear()
    .domain([0, nodes.reduce((sum, d) => sum + (d ? d.cumulative : 0), 0)])
    .range([0, width])

  const nextLevel = level + 1

  return (
    <g
      transform={`translate(${x}, ${y})`}
    >
      {nodes.map((d, i) => {
        const start = nodes
          .slice(0, i)
          .reduce((sum, d) => sum + (d ? d.cumulative : 0), 0)

        const width = xScale(d.cumulative)

        if (width <= 1) {
          return
        }

        const key = `${level}-${i}`
        const name = nodeLabel(d)
        const nextPath = path.concat([name])

        const color = diffColor(d.diff === undefined ? 0 : d.diff, d.cumulative)

        const onClick = (e) => {
            setCurPath(nextPath)
        }

        const onHover = (e) => {
            setHoveringNode(d)
        }

        const xStart = xScale(start)
        const nextWidth = xScale(d.cumulative)
        const nextCurPath = curPath.length == 0 ? [] : curPath.slice(1)

        const onMouseEnter = (e) => setHoveringNode(d)
        const onMouseLeave = (e) => setHoveringNode(undefined)

        return (
            <React.Fragment key={key}>
                <IcicleRect
                    x={xStart}
                    y={0}
                    width={width}
                    height={RowHeight}
                    name={name}
                    color={color}
                    onClick={onClick}
                    onMouseEnter={onMouseEnter}
                    onMouseLeave={onMouseLeave}
                />
                {(data !== undefined && data.length > 0) && (
                    <IcicleGraphNodes
                        data={d.childrenList}
                        x={xStart}
                        y={RowHeight}
                        width={nextWidth}
                        level={nextLevel}
                        setHoveringNode={setHoveringNode}
                        path={nextPath}
                        curPath={nextCurPath}
                        setCurPath={setCurPath}
                    />
                )}
            </React.Fragment>
        )
      })}
    </g>
  )
}

const MemoizedIcicleGraphNodes = React.memo(IcicleGraphNodes)

interface FlamegraphTooltipProps {
    x: number
    y: number
    unit: string
    total: number
    hoveringNode: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined
}

const FlamegraphNodeTooltipTableRows = ({
    hoveringNode
}: {
    hoveringNode: FlamegraphNode.AsObject
}): JSX.Element => {
    if (hoveringNode.meta === undefined) return <></>

    return (
        <>
            {(hoveringNode.meta.pb_function !== undefined && hoveringNode.meta.pb_function.filename !== undefined && hoveringNode.meta.pb_function.filename != '') && (
                <tr>
                    <td className="w-1/5">File</td>
                    <td className="w-4/5">{hoveringNode.meta.pb_function.filename}{(hoveringNode.meta.line !== undefined && hoveringNode.meta.line.line !== undefined && hoveringNode.meta.line.line != 0) ? (` +${hoveringNode.meta.line.line.toString()}`) : (
                        `${(hoveringNode.meta.pb_function !== undefined && hoveringNode.meta.pb_function.startLine !== undefined && hoveringNode.meta.pb_function.startLine != 0) ? ` +${hoveringNode.meta.pb_function.startLine.toString()}` : '' }`
                    )}</td>
                </tr>
            )}
            {(hoveringNode.meta.location !== undefined && hoveringNode.meta.location.address !== undefined && hoveringNode.meta.location.address != 0) && (
                <tr>
                    <td className="w-1/5">Address</td>
                    <td className="w-4/5">{' 0x'+hoveringNode.meta.location.address.toString(16)}</td>
                </tr>
            )}
            {(hoveringNode.meta.mapping !== undefined && hoveringNode.meta.mapping.file != '') && (
                <tr>
                    <td className="w-1/5">Binary</td>
                    <td className="w-4/5">{getLastItem(hoveringNode.meta.mapping.file)}</td>
                </tr>
            )}
        </>
    )
}

export const FlamegraphTooltip = ({
    x,
    y,
    unit,
    total,
    hoveringNode,
}: FlamegraphTooltipProps): JSX.Element => {
    const knownValueFormatter = knownValueFormatters[unit]
    const valueFormatter = knownValueFormatter !== undefined ? knownValueFormatter : formatDefault

    if (hoveringNode === undefined || hoveringNode == null) return <></>

    const diff = hoveringNode.diff === undefined ? 0 : hoveringNode.diff
    const prevValue = hoveringNode.cumulative - diff
    const diffRatio = Math.abs(diff) > 0 ? (diff / prevValue) : 0

    const hoveringFlamegraphNode = hoveringNode as FlamegraphNode.AsObject
    const metaRows = (hoveringFlamegraphNode.meta === undefined) ? (
        <></>) : (<FlamegraphNodeTooltipTableRows hoveringNode={hoveringNode as FlamegraphNode.AsObject} />)

    return (
        <div style={{position: "absolute", left: x+30+24, top: y-20+24}}>
            <div className="flex">
                <div className="m-auto">
                    <div className="border-gray-300 dark:border-gray-500 bg-gray-50 dark:bg-gray-900 rounded-lg p-3 shadow-lg opacity-90" style={{ borderWidth: 1 }}>
                        <div className="flex flex-row">
                            <div className="ml-2 mr-6">
                                <span className="text-gray-700 dark:text-gray-300 my-2">
                                    {(hoveringFlamegraphNode.meta === undefined) ? (
                                            <p>root</p>
                                    ) : (
                                        <>
                                            {(hoveringFlamegraphNode.meta.pb_function !== undefined && hoveringFlamegraphNode.meta.pb_function.name != '') ? (
                                                <p>{hoveringFlamegraphNode.meta.pb_function.name}</p>
                                            ) : (
                                                <>
                                                    {(hoveringFlamegraphNode.meta.location !== undefined && hoveringFlamegraphNode.meta.location.address != 0) ? (
                                                        <p>{'0x'+hoveringFlamegraphNode.meta.location.address.toString(16)}</p>
                                                    ) : (<p>unknown</p>)}
                                                </>
                                            )}
                                        </>
                                    )}
                                    <table className="table-fixed">
                                        <tbody>
                                            <tr>
                                                <td className="w-1/5">Cumulative</td>
                                                <td className="w-4/5">{valueFormatter(hoveringNode.cumulative)} ({((hoveringNode.cumulative * 100) / total).toFixed(2)}%)</td>
                                            </tr>
                                            {(hoveringNode.diff !== undefined && diff != 0) && (
                                                <tr>
                                                    <td className="w-1/5">Diff</td>
                                                    <td className="w-4/5">{`${diff > 0 ? '+' : ''}${valueFormatter(diff)}`} ({`${diff > 0 ? '+' : ''}${(diffRatio*100).toFixed(2)}%`})</td>
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
    )
}


interface IcicleGraphRootNodeProps {
  node: FlamegraphRootNode.AsObject
  width: number
  curPath: string[]
  setCurPath: (path: string[]) => void
  setHoveringNode: (node: FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined) => void
}

export function IcicleGraphRootNode ({
  node,
  width,
  setHoveringNode,
  setCurPath,
  curPath
}: IcicleGraphRootNodeProps) {
    const color = diffColor(node.diff === undefined ? 0 : node.diff, node.cumulative)

    const onClick = (e) => setCurPath([])
    const onMouseEnter = (e) => setHoveringNode(node)
    const onMouseLeave = (e) => setHoveringNode(undefined)
    const path = []

    return (
        <g
            transform={`translate(0, 0)`}
        >
            <IcicleRect
                x={0}
                y={0}
                width={width}
                height={RowHeight}
                name={'root'}
                color={color}
                onClick={onClick}
                onMouseEnter={onMouseEnter}
                onMouseLeave={onMouseLeave}
            />
            <MemoizedIcicleGraphNodes
                data={node.childrenList}
                x={0}
                y={RowHeight}
                width={width}
                level={0}
                setHoveringNode={setHoveringNode}
                path={path}
                curPath={curPath}
                setCurPath={setCurPath}
            />
        </g>
    )
}

const MemoizedIcicleGraphRootNode = React.memo(IcicleGraphRootNode)

interface IcicleGraphProps {
  graph: Flamegraph.AsObject
  width?: number
  curPath: string[]
  setCurPath: (path: string[]) => void
}

function useClientRect () {
  const [rect, setRect] = useState(null)
  const ref = useCallback(node => {
    if (node !== null) {
      setRect(node.getBoundingClientRect())
    }
  }, [])
  return [rect, ref]
}

function formatBytes(bytes: number): string {
    const decimals = 2;
    if (bytes === 0) return '0 Bytes';

    const k = 1000;
    const dm = decimals < 0 ? 0 : decimals;

    // https://physics.nist.gov/cuu/Units/binary.html

    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB'];

    const i = Math.floor(Math.log(Math.abs(bytes)) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

const knownValueFormatters = {
  'bytes': formatBytes,
}

function formatDefault(value: number): string {
    return value.toString()
}

export default function IcicleGraph ({
  graph,
  width,
  setCurPath,
  curPath
}: IcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<FlamegraphNode.AsObject | FlamegraphRootNode.AsObject | undefined>()
  const [pos, setPos] = useState([0, 0])
  const [height, setHeight] = useState(0)
  const ref = useRef<SVGGElement>(null)

  if (graph.root === undefined) return <></>

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height)
    }
  }, [width])

  const throttledSetPos = throttle(setPos, 20)
  const onMouseMove = (e: React.MouseEvent<SVGSVGElement|HTMLDivElement>): void => {
    // X/Y coordinate array relative to svg
    const rel = pointer(e)

    throttledSetPos([rel[0], rel[1]])
  }

  return (
      <div
          onMouseLeave={() => setHoveringNode(undefined)}
      >
          <div
              onMouseMove={onMouseMove}
          >
                  <FlamegraphTooltip
                      unit={graph.unit}
                      total={graph.total}
                      x={pos[0]}
                      y={pos[1]}
                      hoveringNode={hoveringNode}
                  />
          </div>
          <svg
              width={width}
              height={height}
              onMouseMove={onMouseMove}
          >
              <g ref={ref}>
                  <MemoizedIcicleGraphRootNode
                      node={graph.root}
                      setHoveringNode={setHoveringNode}
                      curPath={curPath}
                      setCurPath={setCurPath}
                      width={width !== undefined ? width : 0}
                  />
              </g>
          </svg>
      </div>
  )
}
