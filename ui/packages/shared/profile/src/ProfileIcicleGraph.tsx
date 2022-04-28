import {Flamegraph} from '@parca/client';
import {useAppSelector, selectCompareMode} from '@parca/store';

import DiffLegend from './components/DiffLegend';
import IcicleGraph from './IcicleGraph';

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
    const compareMode = useAppSelector(selectCompareMode);

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
