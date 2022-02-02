import React from 'react';
import {TargetsResponse} from '@parca/client';
import Pill from '../ui/Pill';
import LabelsCell from './LabelsCell';
import LastScrapeCell from './LastScrapeCell';
import {getHealthStatus} from './utils';

enum TargetsTableHeader {
  url = 'URL',
  health = 'Health Status',
  labels = 'Labels',
  lastScrape = 'Last Scrape',
  lastError = 'Last Error',
}

const getRowContentByHeader = ({
  header,
  target,
  key,
}: {
  header: string;
  target: any;
  key: string;
}) => {
  switch (header) {
    case TargetsTableHeader.url: {
      const {url} = target;
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap">
          <p className="text-sm text-gray-900 text-bold">{url}</p>
        </td>
      );
    }
    case TargetsTableHeader.labels: {
      const {
        labels: {labelsList: labels},
        discoveredLabels: {labelsList: discoveredLabels},
      } = target;
      return <LabelsCell labels={labels} discoveredLabels={discoveredLabels} key={key} />;
    }
    case TargetsTableHeader.lastError: {
      const {lastError} = target;
      return (
        <td key={key} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
          {lastError}
        </td>
      );
    }
    case TargetsTableHeader.lastScrape: {
      const {lastScrape, lastScrapeDuration} = target;
      return (
        <LastScrapeCell key={key} lastScrape={lastScrape} lastScrapeDuration={lastScrapeDuration} />
      );
    }
    case TargetsTableHeader.health: {
      const {health} = target;
      const {label, colorVariant} = getHealthStatus(health);
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

const TargetsTable = ({targets}: {targets: TargetsResponse.AsObject[]}) => {
  const headers = Object.keys(TargetsTableHeader) as Array<keyof typeof TargetsTableHeader>;

  return (
    <table className="min-w-full divide-y divide-gray-200">
      <thead className="bg-gray-50">
        <tr>
          {headers.map((header: string) => (
            <th
              key={header}
              scope="col"
              className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider"
            >
              {TargetsTableHeader[header]}
            </th>
          ))}
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {targets.map((target: TargetsResponse.AsObject) => {
          return (
            <tr key={target['url']}>
              {headers.map((header: string) => {
                const key = `table-cell-${header}-${target['url']}`;
                return getRowContentByHeader({header: TargetsTableHeader[header], target, key});
              })}
            </tr>
          );
        })}
      </tbody>
    </table>
  );
};

export default TargetsTable;
