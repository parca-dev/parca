import { useState } from 'react'
import { throttle } from 'lodash'
import IcicleGraph from './IcicleGraph'
import { ProfileSource } from './ProfileSource'
import { Spinner } from 'react-bootstrap'
import { CalcWidth } from '@parca/dynamicsize'
import { Flamegraph, FlamegraphNode } from '@parca/client'

interface ProfileIcicleGraphProps {
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

function formatDefault(value: number): string {
    return value.toString()
}

export default function ProfileIcicleGraph ({
  graph,
}: ProfileIcicleGraphProps) {
  const [hoveringNode, setHoveringNode] = useState<FlamegraphNode.AsObject | undefined>()
  const [curPath, setCurPath] = useState<string[]>([])

  if (graph === undefined) return <div>no data...</div>
  const total = graph.total
  if (total == 0) return <>Profile has no samples</>

  const knownValueFormatter = {
    'bytes': formatBytes,
  }[graph.unit]

  const valueFormatter = knownValueFormatter !== undefined ? knownValueFormatter : formatDefault

  function nodeAsText (node: FlamegraphNode.AsObject | undefined): string {
    if (node === undefined) return ''

    const diff = node.diff === undefined ? 0 : node.diff
    const prevValue = node.cumulative - diff
    const diffRatio = Math.abs(diff) > 0 ? (diff / prevValue) : 0
    const diffRatioText = prevValue > 0 ? ` (${node.diff > 0 ? '+' : ''}${(diffRatio*100).toFixed(2)}%)` : ''

    const diffText = (node.diff !== undefined && node.diff != 0) ? ` Diff: ${node.diff > 0 ? '+' : ''}${valueFormatter(node.diff)}${diffRatioText}` : ''

    return `${node.name.split(' ')[0]} (${((node.cumulative * 100) / total).toFixed(2)}%) ${valueFormatter(node.cumulative)}${diffText}`
  }

  const nodeLabel = hoveringNode == null ? nodeAsText(graph.root) : nodeAsText(hoveringNode)

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
