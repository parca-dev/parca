import IcicleGraph from './IcicleGraph';
import {Flamegraph} from '@parca/client';

interface ProfileIcicleGraphProps {
  width?: number;
  graph: Flamegraph.AsObject | undefined;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
}

const ProfileIcicleGraph = ({width, graph, curPath, setNewCurPath}: ProfileIcicleGraphProps) => {
  if (graph === undefined) return <div>no data...</div>;
  const total = graph.total;
  if (total === 0) return <>Profile has no samples</>;

  return <IcicleGraph width={width} graph={graph} curPath={curPath} setCurPath={setNewCurPath} />;
};

export default ProfileIcicleGraph;
