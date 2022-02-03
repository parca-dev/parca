import React, {useState} from 'react';
import {ChevronDoubleDownIcon, ChevronDoubleUpIcon} from '@heroicons/react/solid';
import Pill, {Variant} from '../ui/Pill';

const LabelsCell = ({key, labels, discoveredLabels}) => {
  const [areDiscoveredLabelsVisible, setAreDiscoveredLabelsVisible] = useState<boolean>(false);
  const allLabels = areDiscoveredLabelsVisible ? [...labels, ...discoveredLabels] : labels;

  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 flex flex-col w-96">
      <div className="flex flex-wrap">
        {allLabels.length &&
          allLabels.map((item: {name: string; value: string}) => {
            return (
              <div className="pb-1 pr-1">
                <Pill
                  key={item.name}
                  variant={'info' as Variant}
                >{`${item.name}="${item.value}"`}</Pill>
              </div>
            );
          })}
      </div>
      {areDiscoveredLabelsVisible ? (
        <div
          className="flex rounded-lg bg-neutral-100 p-1 justify-center items-center mt-1"
          onClick={() => setAreDiscoveredLabelsVisible(false)}
        >
          <span className="mr-1">Hide Discovered Labels</span>
          <ChevronDoubleUpIcon className="h-5 w-5" aria-hidden="true" />
        </div>
      ) : (
        <div
          className="flex rounded-lg bg-neutral-100 p-1 justify-center items-center mt-1"
          onClick={() => setAreDiscoveredLabelsVisible(true)}
        >
          <span className="mr-1">Show Discovered Labels</span>
          <ChevronDoubleDownIcon className="h-5 w-5" aria-hidden="true" />
        </div>
      )}
    </td>
  );
};

export default LabelsCell;
