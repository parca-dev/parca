import IcicleGraph from './IcicleGraph';
import DiffLegend from './components/DiffLegend';
import {Flamegraph} from '@parca/client';

interface ProfileIcicleGraphProps {
  width?: number;
  graph: Flamegraph | undefined;
  sampleUnit: string;
  curPath: string[] | [];
  setNewCurPath: (path: string[]) => void;
}

const ProfileIcicleGraph = ({
  width,
  graph,
  curPath,
  setNewCurPath,
  sampleUnit,
}: ProfileIcicleGraphProps) => {
  if (graph === undefined) return <div>no data...</div>;
  const total = graph.total;
  if (parseFloat(total) === 0) return <>Profile has no samples</>;

  return (
    <>
      <DiffLegend />
      <IcicleGraph
        width={width}
        graph={graph}
        curPath={curPath}
        setCurPath={setNewCurPath}
        sampleUnit={sampleUnit}
      />
    </>
  );
};

export default ProfileIcicleGraph;
