import { useState, useEffect } from 'react'
import IcicleGraph from './IcicleGraph'
import { Flamegraph } from '@parca/client'

interface ProfileIcicleGraphProps {
  width?: number
  graph: Flamegraph.AsObject | undefined
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
  width,
  graph
}: ProfileIcicleGraphProps) {
  const [curPath, setCurPath] = useState<string[]>([])

  useEffect(() => {
    setCurPath([])
  }, [graph])

  if (graph === undefined) return <div>no data...</div>
  const total = graph.total
  if (total === 0) return <>Profile has no samples</>

  const setNewCurPath = (path: string[]) => {
    if (!arrayEquals(curPath, path)) {
      setCurPath(path)
    }
  }

  return (
        <IcicleGraph
          width={width}
          graph={graph}
          curPath={curPath}
          setCurPath={setNewCurPath}
        />
  )
}
