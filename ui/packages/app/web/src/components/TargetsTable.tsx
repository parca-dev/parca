import React, {useState, useEffect} from 'react';
import {TargetsResponse} from '@parca/client';
import Pill from './ui/Pill';
import Button from './ui/Button';

export interface ITargetEndpoint {}

const LabelsCell = ({key, labels, discoveredLabels}) => {
  const [areDiscoveredLabelsVisible, setAreDiscoveredLabelsVisible] = useState<boolean>(false);
  console.log(areDiscoveredLabelsVisible);

  return (
    <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 flex flex-wrap">
      {labels.length &&
        labels.map(item => {
          return <Pill key={item['name']} variant="info">{`${item.name}="${item.value}"`}</Pill>;
        })}
      {areDiscoveredLabelsVisible &&
        discoveredLabels.length &&
        discoveredLabels.map(item => {
          return <Pill key={item['name']} variant="info">{`${item.name}="${item.value}"`}</Pill>;
        })}
      <Button
        onClick={() => {
          areDiscoveredLabelsVisible
            ? setAreDiscoveredLabelsVisible(false)
            : setAreDiscoveredLabelsVisible(true);
        }}
      >{`${areDiscoveredLabelsVisible ? 'Hide' : 'Show'} Discovered Labels`}</Button>
    </td>
  );
};

export enum HealthStatus {
  'Unspecified',
  'Good',
  'Bad',
}

const getHealthStatus = (numericValue: number) => {
  const label = HealthStatus[numericValue];
  const colorVariants = {
    Unspecified: 'neutral',
    Good: 'success',
    Bad: 'danger',
  };
  return {label, colorVariant: colorVariants[label]};
};

const getRowContentByHeader = ({header, value, key}: {header: string; value: any; key: string}) => {
  switch (header) {
    case 'URL': {
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap">
          <p className="text-sm text-gray-900 text-bold">{value}</p>
        </td>
      );
    }
    case 'Labels': {
      const {
        labels: {labelsList: labels},
        discoveredLabels: {labelsList: discoveredLabels},
      } = value;
      return <LabelsCell labels={labels} discoveredLabels={discoveredLabels} key={key} />;
    }
    case 'Last Error': {
      <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
        {value}
      </td>;
    }
    case 'Last Scrape':
    case 'Last Scrape Duration': {
      const {seconds, nanos} = value;
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
          {seconds}
        </td>
      );
    }
    case 'Health Status': {
      const {label, colorVariant} = getHealthStatus(value);
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap">
          <Pill variant={colorVariant}>{label}</Pill>
        </td>
      );
    }
    default: {
      return <td />;
    }
  }
};

const TargetsTable = ({endpoints}: {endpoints: ITargetEndpoint[]}) => {
  const tableHeaders = {
    url: 'URL',
    health: 'Health Status',
    labels: 'Labels',
    lastScrape: 'Last Scrape',
    lastScrapeDuration: 'Last Scrape Duration',
    lastError: 'Last Error',
  };

  return (
    <table className="min-w-full divide-y divide-gray-200">
      <thead className="bg-gray-50">
        <tr>
          {Object.keys(tableHeaders).map((header: string) => (
            <th
              key={header}
              scope="col"
              className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              {header}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {endpoints.map((endpoint: TargetsResponse.AsObject) => {
          return (
            <tr key={endpoint['url']}>
              {Object.keys(tableHeaders).map((header: string) => {
                const key = `table-cell-${header}-${endpoint['url']}`;
                const value =
                  tableHeaders[header] === 'Labels'
                    ? {labels: endpoint[header], discoveredLabels: endpoint['discoveredLabels']}
                    : endpoint[header];
                return getRowContentByHeader({header: tableHeaders[header], value, key});
              })}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

export default TargetsTable;
