import React, { MouseEvent, useCallback, useEffect, useRef, useState } from 'react'
import { scaleLinear } from 'd3-scale'
import { Flamegraph, FlamegraphNode, FlamegraphRootNode } from '@parca/client'

const transitionTime = '250ms'
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
  onHover: (e: MouseEvent) => void
  onClick: (e: MouseEvent) => void
}

function IcicleRect ({
  x,
  y,
  width,
  height,
  color,
  name,
  onHover,
  onClick
}: IcicleRectProps) {
  return (
    <g
      transform={`translate(${x}, ${y})`}
      style={{ cursor: 'pointer', transition: transformTransition }}
      onMouseEnter={(e) => onHover(e)}
      onClick={(e) => onClick(e)}
    >
      <rect
        x={0}
        y={0}
        width={width}
        height={height}
        style={{
          transition: widthTransition,
          stroke: 'white',
          fill: color
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
  setHoveringNode: (node: FlamegraphNode.AsObject) => void
  path: () => string[]
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
    const mapping = `${(node.meta.mapping !== undefined && node.meta.mapping.file != '') ? '['+getLastItem(node.meta.mapping.file)+']' : ''}`
    if (node.meta.pb_function !== undefined && node.meta.pb_function.name != '') return mapping+' '+node.meta.pb_function.name

    const address = `${(node.meta.location !== undefined && node.meta.location.address !== undefined && node.meta.location.address != 0) ? ' 0x'+node.meta.location.address.toString(16) : ''}`
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
  if (data === undefined || data.length == 0) return <></>

  const nodes = curPath.length == 0 ? data : data.filter((d, i) => d != null && curPath[0] == nodeLabel(d))

  const xScale = scaleLinear()
    .domain([0, nodes.reduce((sum, d) => sum + (d ? d.cumulative : 0), 0)])
    .range([0, width])

  return (
    <g
      transform={`translate(${x}, ${y})`}
      style={{ transition: transformTransition }}
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
        const nextPath = () => {
          return path().concat([name])
        }

        const color = diffColor(d.diff === undefined ? 0 : d.diff, d.cumulative)

        return (
          <React.Fragment key={key}>
            <IcicleRect
              x={xScale(start)}
              y={0}
              width={width}
              height={RowHeight}
              name={name}
              color={color}
              onClick={function (e) {
                const p = nextPath()
                setCurPath(p)
              }}
              onHover={(e) => setHoveringNode(d)}
            />
            <IcicleGraphNodes
              data={d.childrenList}
              x={xScale(start)}
              y={RowHeight}
              width={xScale(d.cumulative)}
              level={level + 1}
              setHoveringNode={setHoveringNode}
              path={() => nextPath()}
              curPath={curPath.length == 0 ? [] : curPath.slice(1)}
              setCurPath={setCurPath}
            />
          </React.Fragment>
        )
      })}
    </g>
  )
}

interface IcicleGraphRootNodeProps {
  node: FlamegraphRootNode.AsObject
  width: number
  curPath: string[]
  setCurPath: (path: string[]) => void
  setHoveringNode: (node: FlamegraphNode.AsObject) => void
}

export function IcicleGraphRootNode ({
  node,
  width,
  setHoveringNode,
  setCurPath,
  curPath
}: IcicleGraphRootNodeProps) {
    const color = diffColor(node.diff === undefined ? 0 : node.diff, node.cumulative)

    return (
        <g
            transform={`translate(0, 0)`}
            style={{ transition: transformTransition }}
        >
            <IcicleRect
                x={0}
                y={0}
                width={width}
                height={RowHeight}
                name={'root'}
                color={color}
                onClick={function (e) {
                    setCurPath([])
                }}
                onHover={(e) => setHoveringNode(node)}
            />
            <IcicleGraphNodes
                data={node.childrenList}
                x={0}
                y={RowHeight}
                width={width}
                level={1}
                setHoveringNode={setHoveringNode}
                path={() => []}
                curPath={curPath}
                setCurPath={setCurPath}
            />
        </g>
    )
}

interface IcicleGraphProps {
  graph: Flamegraph.AsObject
  width?: number
  curPath: string[]
  setCurPath: (path: string[]) => void
  setHoveringNode: (node: FlamegraphNode.AsObject) => void
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

export default function IcicleGraph ({
  graph,
  width,
  setHoveringNode,
  setCurPath,
  curPath
}: IcicleGraphProps) {
  const [height, setHeight] = useState(0)
  const ref = useRef<SVGGElement>(null)

  if (graph.root === undefined) return <></>

  useEffect(() => {
    if (ref.current != null) {
      setHeight(ref?.current.getBoundingClientRect().height)
    }
  }, [width])

  return (
    <svg width={width} height={height}>
      <g ref={ref}>
        <IcicleGraphRootNode
          node={graph.root}
          setHoveringNode={setHoveringNode}
          curPath={curPath}
          setCurPath={setCurPath}
          width={width !== undefined ? width : 0}
        />
      </g>
    </svg>
  )
}
