import Sankey from './Sankey';
import {useContainerDimensions} from '@parca/dynamicsize';
import Dag from './Dag';
import {fgSimple as testData} from './testData';

interface Props {
  data: any;
}

const Callgraph = ({data}: Props): JSX.Element => {
  const {ref, dimensions} = useContainerDimensions();
  return (
    <div ref={ref}>
      {/* <Sankey data={testData2} width={dimensions?.width} height={600} /> */}
      {/* @ts-ignore */}
      <Dag width={dimensions?.width} height={600} data={testData.flamegraph.root} />
    </div>
  );
};

export default Callgraph;
