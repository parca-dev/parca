import {Flamegraph} from '@parca/client';
import {useAppSelector, selectCompareMode} from '@parca/store';
import {useContainerDimensions} from '@parca/dynamicsize';

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
  graph,
  curPath,
  setNewCurPath,
  sampleUnit,
}: ProfileIcicleGraphProps) => {
  const compareMode = useAppSelector(selectCompareMode);
  const {ref, dimensions} = useContainerDimensions();

  if (graph === undefined) return <div>no data...</div>;
  const total = graph.total;
  if (parseFloat(total) === 0) return <>Profile has no samples</>;

  return (
    <>
      {compareMode && <DiffLegend />}
      <div ref={ref}>
        <IcicleGraph
          width={dimensions?.width}
          graph={graph}
          curPath={curPath}
          setCurPath={setNewCurPath}
          sampleUnit={sampleUnit}
        />
      </div>
    </>
  );
};

export default ProfileIcicleGraph;
