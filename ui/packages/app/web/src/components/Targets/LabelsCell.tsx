import React, {useState} from 'react';
import {ChevronDoubleDownIcon, ChevronDoubleUpIcon} from '@heroicons/react/solid';
import Pill from '../ui/Pill';

export interface ITargetEndpoint {}

const LabelsCell = ({key, labels, discoveredLabels}) => {
  const [areDiscoveredLabelsVisible, setAreDiscoveredLabelsVisible] = useState<boolean>(false);

  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 flex flex-col w-96">
      <div className="flex flex-wrap">
        {labels.length &&
          labels.map(item => {
            return (
              <div className="pb-1 pr-1">
                <Pill key={item['name']} variant="info">{`${item.name}="${item.value}"`}</Pill>
              </div>
            );
          })}
        {areDiscoveredLabelsVisible &&
          discoveredLabels.length &&
          discoveredLabels.map(item => {
            return (
              <div className="pb-1 pr-1">
                <Pill key={item['name']} variant="info">{`${item.name}="${item.value}"`}</Pill>
              </div>
            );
          })}
      </div>

      <div
        className="flex rounded-lg bg-neutral-100 p-1 justify-center items-center mt-1"
        onClick={() =>
          areDiscoveredLabelsVisible
            ? setAreDiscoveredLabelsVisible(false)
            : setAreDiscoveredLabelsVisible(true)
        }
      >
        <span className="mr-1">{`${
          areDiscoveredLabelsVisible ? 'Hide' : 'Show'
        } Discovered Labels`}</span>
        {areDiscoveredLabelsVisible ? (
          <ChevronDoubleUpIcon className="h-5 w-5" aria-hidden="true" />
        ) : (
          <ChevronDoubleDownIcon className="h-5 w-5" aria-hidden="true" />
        )}
      </div>
    </td>
  );
};

export default LabelsCell;
