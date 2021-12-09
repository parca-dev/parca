import { useState, useEffect, forwardRef, useImperativeHandle } from 'react'
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

const ProfileIcicleGraph = forwardRef<{ resetIcicleGraph: () => void }, ProfileIcicleGraphProps>((props, ref) => {
    const [curPath, setCurPath] = useState<string[]>([])

    const { width, graph } = props

    useEffect(() => {
      setCurPath([])
    }, [graph])

    useImperativeHandle(ref, () => ({
      resetIcicleGraph() {
        setCurPath([])
      }
    }))

    if (graph === undefined) return <div>no data...</div>
    const total = graph.total
    if (total === 0) return <>Profile has no samples</>

    const setNewCurPath = (path: string[]) => {
      if (!arrayEquals(curPath, path)) {
        setCurPath(path)
      }
    }

    return <IcicleGraph width={width} graph={graph} curPath={curPath} setCurPath={setNewCurPath} />
  }
)

export default ProfileIcicleGraph
