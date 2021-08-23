import { useState } from 'react'
import { throttle } from 'lodash'
import IcicleGraph from './IcicleGraph'
import { ProfileSource } from './ProfileSource'
import { Spinner } from 'react-bootstrap'
import { CalcWidth } from '@parca/dynamicsize'
import { QueryResponse, FlamegraphNode } from '@parca/client'

interface ProfileIcicleGraphProps {
  queryResponse: QueryResponse
  profileSource: ProfileSource
}

function arrayEquals (a, b): boolean {
  return (
    Array.isArray(a) &&
    Array.isArray(b) &&
    a.length === b.length &&
    a.every((val, index) => val === b[index])
  )
}

export default function ProfileIcicleGraph ({
  queryResponse,
  profileSource,
}: ProfileIcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<FlamegraphNode.AsObject | undefined>()
  const [curPath, setCurPath] = useState<string[]>([])

  if (!queryResponse) {
    return (
      <div
        style={{
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          height: 'inherit'
        }}
      >
        <Spinner animation='border' role='status'>
          <span className='sr-only'>Loading...</span>
        </Spinner>
      </div>
    )
  }

  const graph = queryResponse.getFlamegraph()
  if (graph === undefined) return <div>no data...</div>
  const total = graph.getTotal()
  if (total == 0) return <>Profile has no samples</>

  function nodeAsText (node: FlamegraphNode.AsObject | undefined): string {
    if (node === undefined) return ''
    return `${node.name.split(' ')[0]} (${((node.cumulative * 100) / total).toFixed(2)}%)`
  }

  const nodeLabel = hoveringNode == null ? nodeAsText(graph.getRoot()?.toObject()) : nodeAsText(hoveringNode)

  const setNewCurPath = (path: string[]) => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path)
    }
  }

  return (
    <div className='container-fluid' style={{ padding: 0 }}>
      <p>Node: {nodeLabel}</p>
      <CalcWidth throttle={300} delay={2000}>
        <IcicleGraph
          graph={graph}
          setHoveringNode={throttle(setHoveringNode, 100)}
          curPath={curPath}
          setCurPath={throttle(setNewCurPath, 100)}
        />
      </CalcWidth>
    </div>
  )
}
