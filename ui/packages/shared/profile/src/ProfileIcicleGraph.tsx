import {Flamegraph} from '@parca/client';
import {useAppSelector, selectCompareMode} from '@parca/store';
import {getDebugInfoSourceCode} from '@parca/functions';

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
      <button
        onClick={() => {
          getDebugInfoSourceCode('02c66c637105cd4016513abc4c8e79d14bcc5d87');
        }}
      >
        get source code
      </button>
      {compareMode && <DiffLegend />}
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
